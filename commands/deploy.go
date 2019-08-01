package commands

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/sachamorard/swapper/response"
	"github.com/sachamorard/swapper/utils"
	"github.com/sachamorard/swapper/yaml"
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
	hostname, _ := utils.GetHostname()
	deployUsage = strings.Replace(deployUsage, "{{hostname}}", hostname, -1)

	arguments, _ := docopt.ParseArgs(deployUsage, argv, "")
	return arguments
}

func Deploy(argv []string) response.Response {
	arguments := DeployArgs(argv)
	vars := utils.InterfaceToArray(arguments["--var"])
	file := arguments["--file"].(string)
	masterHostname := arguments["--master"].(string)
	cleanYaml, err := yaml.PrepareSwapperYaml(file, vars)
	if err != nil {
		return response.Fail(err.Error())
	}

	yamlConf, err := yaml.ParseSwapperYaml(cleanYaml)
	if err != nil {
		return response.Fail(err.Error())
	}
	if len(yamlConf.Masters) != 0 {
		return response.Fail(response.ErrorMessages["no_masters_field"])
	}

	if yamlConf.Time != 0 {
		return response.Fail(response.ErrorMessages["no_time_field"])
	}

	if yamlConf.Hash != "" {
		return response.Fail(response.ErrorMessages["no_hash_field"])
	}

	if masterHostname == "localhost" {
		masterHostname, _ = utils.GetHostname()
	}
	if masterHostname == "127.0.0.1" {
		masterHostname, _ = utils.GetHostname()
	}

	i := strings.Index(masterHostname, ":")
	if i == -1 {
		masterHostname = masterHostname + ":1207"
	}

	swapperYml := CurlSwapperYaml(masterHostname)
	if swapperYml == "" {
		return response.Fail(fmt.Sprintf(response.ErrorMessages["bad_master_addr"], masterHostname))
	}

	masterSplit := strings.Split(masterHostname, ":")
	port := masterSplit[1]

	return DeployFile(cleanYaml, port, yamlConf)
}

func DeployFile(cleanYaml string, port string, yamlConf yaml.YamlConf) response.Response {
	err := DeployReq(cleanYaml, port)
	if err != nil {
		_ = utils.SlackSendError("Deployment failed\n"+err.Error(), yamlConf)
		return response.Fail(err.Error())
	}
	_ = utils.SlackSendSuccess("Deployment succeed", yamlConf)
	return response.Success("\n>> Deployment succeed\n")
}

func DeployReq(cleanYaml string, port string) (err error) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:"+port+"/swapper.yml", bytes.NewBuffer([]byte(cleanYaml)))
	if err != nil {
		return errors.New(fmt.Sprintf(response.ErrorMessages["request_failed"], err))
	}
	req.Header.Set("Content-Type", "text/yml")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New(fmt.Sprintf(response.ErrorMessages["deploy_failed"], err))
	}
	defer resp.Body.Close()

	if resp.Status != "200 OK" {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(fmt.Sprintf(response.ErrorMessages["deploy_failed"], body))
	} else {
		return nil
	}
}