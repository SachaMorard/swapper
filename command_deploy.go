package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/docopt/docopt-go"
	"cloud.google.com/go/storage"
)

var (
	deployUsage = `
swapper deploy [OPTIONS].

Deploy new swapper configuration and start swapping containers.

Usage:
 swapper deploy [-f <file>] [--var <variable>...] [--master <hostname>]
 swapper deploy (-h|--help)

Options:
 -h --help                 Show this screen.
 -f NAME --file=NAME       Swapper yml config file [default: swapper.yml]
 --var VAR=VALUE           To inject variable into swapper.yml file
 --master=HOSTNAME         Master's hostname [default: {{hostname}}]

Examples:
 To deploy new swapper configuration and start swapping containers, create a new version of swapper.yml, then:
 $ swapper deploy --file swapper.yml

 To deploy new dynamic swapper configuration and start swapping containers, create a new version of swapper.yml with variables in it, then:
 $ swapper deploy --file swapper.yml --var ENV=prod --var TAG=1.0.2
`
)

func DeployArgs(argv []string) docopt.Opts {
	hostname, _ := GetHostname()
	deployUsage = strings.Replace(deployUsage, "{{hostname}}", hostname, -1)

	arguments, _ := docopt.ParseArgs(deployUsage, argv, "")
	return arguments
}

func Deploy(argv []string) Response {
	arguments := DeployArgs(argv)
	vars := InterfaceToArray(arguments["--var"])
	file := arguments["--file"].(string)
	masterHostname := arguments["--master"].(string)
	cleanYaml, err := PrepareSwapperYaml(file, vars)
	if err != nil {
		return Fail(err.Error())
	}

	yamlConf, err := ParseSwapperYaml(cleanYaml)
	if err != nil {
		return Fail(err.Error())
	}

	if len(yamlConf.Masters) != 0 {
		return Fail(errorMessages["no_masters_field"])
	}

	if yamlConf.Time != 0 {
		return Fail(errorMessages["no_time_field"])
	}

	if yamlConf.Hash != "" {
		return Fail(errorMessages["no_hash_field"])
	}

	if yamlConf.Master.Driver == "local" {
		if masterHostname == "localhost" {
			masterHostname, _ = GetHostname()
		}
		if masterHostname == "127.0.0.1" {
			masterHostname, _ = GetHostname()
		}

		i := strings.Index(masterHostname, ":")
		if i == -1 {
			masterHostname = masterHostname + ":1207"
		}

		swapperYml := CurlSwapperYaml(masterHostname)
		if swapperYml == "" {
			return Fail(fmt.Sprintf(errorMessages["bad_master_addr"], masterHostname))
		}

		masterSplit := strings.Split(masterHostname, ":")
		port := masterSplit[1]

		return DeployFile(cleanYaml, port, yamlConf)
	}

	hasher := md5.New()
	hasher.Write([]byte(cleanYaml))
	hash := hex.EncodeToString(hasher.Sum(nil))
	cleanYaml = cleanYaml+"hash: "+hash

	if yamlConf.Master.Driver == "gcp" {
		ctx := context.Background()

		if yamlConf.Master.CredentialsFile != "" {
			_ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", yamlConf.Master.CredentialsFile)
		}

		// Creates a client.
		client, errClient := storage.NewClient(ctx)
		if errClient != nil {
			return Fail(fmt.Sprintf("Failed to create client: %v", errClient))
		}

		// Sets the name for the new bucket.
		bucketName := "swapper-master-"+yamlConf.Master.ProjectId

		// Creates a Bucket instance if not exists
		bucket := client.Bucket(bucketName)
		_, err = bucket.Attrs(ctx)
		if err != nil {
			// Creates the new bucket.
			if err := bucket.Create(ctx, yamlConf.Master.ProjectId, nil); err != nil {
				msg := fmt.Sprintf("Deployment failed\nFailed to create bucket: %v", err)
				_ = SlackSendError(msg, yamlConf)
				return Fail(msg)
			}
			fmt.Printf("Bucket %v created.\n", bucketName)
		}


		if err := write(cleanYaml, client, bucketName, yamlConf.Master.ClusterName+"/swapper.yml"); err != nil {
			msg := fmt.Sprintf("Deployment failed\nCannot write object: %v", err)
			_ = SlackSendError(msg, yamlConf)
			return Fail(msg)
		}

		_ = SlackSendSuccess("Deployment succeed", yamlConf)
		nodeInstruction := "To start a node, execute:\nswapper node start --join gs://"+bucketName+"/"+yamlConf.Master.ClusterName+"/swapper.yml"
		return Success("\n>> Deployment succeed\n"+nodeInstruction+"\n")
	}

	return Fail("")
}

func write(swapperYml string, client *storage.Client, bucketName string, object string) error {

	ctx := context.Background()
	src := strings.NewReader(swapperYml)
	wc := client.Bucket(bucketName).Object(object).NewWriter(ctx)
	if _, err := io.Copy(wc, src); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	// [END upload_file]
	return nil
}

func DeployFile(cleanYaml string, port string, yamlConf YamlConf) Response {
	err := DeployReq(cleanYaml, port)
	if err != nil {
		_ = SlackSendError("Deployment failed\n"+err.Error(), yamlConf)
		return Fail(err.Error())
	}
	_ = SlackSendSuccess("Deployment succeed", yamlConf)
	return Success("\n>> Deployment succeed\n")
}

func DeployReq(cleanYaml string, port string) (err error) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:"+port+"/swapper.yml", bytes.NewBuffer([]byte(cleanYaml)))
	if err != nil {
		return errors.New(fmt.Sprintf(errorMessages["request_failed"], err))
	}
	req.Header.Set("Content-Type", "text/yml")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New(fmt.Sprintf(errorMessages["deploy_failed"], err))
	}
	defer resp.Body.Close()

	if resp.Status != "200 OK" {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(fmt.Sprintf(errorMessages["deploy_failed"], body))
	} else {
		return nil
	}
}
