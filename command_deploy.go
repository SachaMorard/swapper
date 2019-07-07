package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/docopt/docopt-go"
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
