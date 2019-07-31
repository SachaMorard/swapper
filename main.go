package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"gopkg.in/yaml.v2"
)

var (
	usage = `
Usage: swapper COMMAND [OPTIONS]

Commands:
 master     Manage master
 node       Manage node
 status     Status of your 
 deploy     Deploy a new Swapper configuration
 version    Show the Swapper version information

Run 'swapper COMMAND --help' for more information on a command.

`
	haproxyBaseConf = `
global
    log 127.0.0.1 local5 debug

defaults
    log     global
    option  dontlognull
    timeout connect 5000
    timeout client  50000
    timeout server  50000
`
	currentHash = ""
	firstYaml = `
version: "1"

services:
  hello:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: latest`
)

const (
	version    = "1.0.1"
	pidDirectory = "/tmp/swapper"
	yamlDirectory = "/tmp/swapper"
)

type Response struct {
	Code    int
	Message string
}

func Help() Response {
	return Success(usage)
}

func HelpMaster() Response {
	return Success(masterUsage)
}
func HelpNode() Response {
	return Success(nodeUsage)
}
func GetHostname() (hostname string, err error) {
	hostname, err = os.Hostname()
	return
}

func Command(command string) (string, error) {
	args := strings.Split(strings.TrimSpace(command), " ")
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.Output()
	return string(out), err
}

func Success(message string) Response {
	return Response{Code: 0, Message: message}
}

func Fail(message string) Response {
	return Response{Code: 1, Message: message}
}

func Respond(response Response) {
	fmt.Println(response.Message)
	os.Exit(response.Code)
}

func FileExists(filePath string) bool {
	if _, err := os.Stat(filePath); err == nil {
		return true
	}
	return false
}

func ShutUpOut() *os.File {
	oldOut := os.Stdout // keep backup of the real stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	return oldOut
}

func RestoreOut(oldOut *os.File)  {
	os.Stdout = oldOut // restoring the real stdout
}

func PrepareSwapperYaml(sourceFile string, vars []string) (cleanYaml string, err error) {

	input, ioErr := ioutil.ReadFile(sourceFile)
	if ioErr != nil {
		return cleanYaml, errors.New(fmt.Sprintf(errorMessages["file_not_exist"], sourceFile))
	}

	// Unmarshal Yaml to remove comments and clean it
	var val interface{}
	errUnmarshal := yaml.Unmarshal([]byte(string(input)), &val)
	if errUnmarshal != nil {
		return cleanYaml, errors.New(errorMessages["yaml_invalid"])
	}

	d, errMarshal := yaml.Marshal(&val)
	if errMarshal != nil {
		return cleanYaml, errors.New(errorMessages["yaml_invalid"])
	}
	cleanYaml = string(d)

	// Transform vars strings to map
	varMap := make(map[string]string)
	var keyValue []string
	for _, e := range vars {
		if e != "" {
			keyValue = strings.Split(e, "=")
			varMap[keyValue[0]] = keyValue[1]
		}
	}

	// Replace variables if exist
	re := regexp.MustCompile(`\${[a-zA-Z0-9_-]+}`)
	matches := re.FindAllString(cleanYaml, -1)
	var varname string
	for _, p := range matches {
		varname = strings.Replace(strings.Replace(p, "${", "", 1), "}", "", -1)
		if varMap[varname] != "" {
			cleanYaml = strings.Replace(cleanYaml, p, varMap[varname], -1)
		}
	}

	// Check whether variables still need to be defined
	matches = re.FindAllString(cleanYaml, -1)
	if len(matches) > 0 {
		varnames := ""
		examples := ""
		alreadyDone := map[string]bool{}
		for _, p := range matches {

			varname = strings.Replace(strings.Replace(p, "${", "", 1), "}", "", -1)
			if alreadyDone[varname] != true {
				varnames = varnames + varname + " "
				examples = examples + "--var " + varname + "=<value> "
				alreadyDone[varname] = true
			}
		}
		return cleanYaml, errors.New(fmt.Sprintf(errorMessages["var_missing"], strings.TrimSpace(varnames), strings.TrimSpace(examples)))
	}

	return cleanYaml, err
}

func CreateHaproxyConf(yamlConf YamlConf) (conf string, err error) {

	var haproxyConf []string
	// create frontend haproxy conf
	haproxyConf = append(haproxyConf, haproxyBaseConf)
	for _, frontend := range yamlConf.Frontends  {
		haproxyConf = append(haproxyConf, "frontend "+frontend.Name)
		haproxyConf = append(haproxyConf, "    option forwardfor")
		haproxyConf = append(haproxyConf, "    mode tcp")
		haproxyConf = append(haproxyConf, "    option tcplog")
		haproxyConf = append(haproxyConf, "    maxconn 800")
		haproxyConf = append(haproxyConf, "    bind 0.0.0.0:"+strconv.Itoa(frontend.Listen))
		haproxyConf = append(haproxyConf, "    default_backend "+frontend.BackendName)
		haproxyConf = append(haproxyConf, "")
	}


	for _, frontend := range yamlConf.Frontends {
		haproxyConf = append(haproxyConf, "backend "+frontend.BackendName)
		haproxyConf = append(haproxyConf, "    balance roundrobin")

		for _, container := range frontend.Containers {
			containerName := "swapper-container." + yamlConf.Hash + "." + frontend.ServiceName + "." + strconv.Itoa(container.Index)
			Id, err := Command("docker ps --format {{.ID}} --filter name=" + containerName)
			if err != nil || Id == "" {
				return conf, errors.New(fmt.Sprintf(errorMessages["container_failed"], containerName))
			}

			ipCommand := "docker inspect -f {{.NetworkSettings.IPAddress}} " + containerName
			outIp, err := Command(ipCommand)
			if err != nil {
				return conf, errors.New(fmt.Sprintf(errorMessages["container_ip_failed"], containerName))
			}
			ip := strings.TrimSpace(outIp)

			haproxyConf = append(haproxyConf, "    server container_"+strconv.Itoa(container.Index)+" "+ip+":"+strconv.Itoa(frontend.Bind)+" check observe layer4 weight "+strconv.Itoa(container.Weight))
		}
	}

	if len(haproxyConf) == 0 {
		return conf, errors.New(errorMessages["haproxy_conf_empty"])
	}

	return strings.Join(haproxyConf, "\n"), err
}

func InterfaceToArray(arguments interface{}) (arr []string) {
	switch reflect.TypeOf(arguments).Kind() {
	case reflect.Slice:

		s := reflect.ValueOf(arguments)

		for i := 0; i < s.Len(); i++ {
			arr = append(arr, (s.Index(i)).String())
		}

	}
	return arr
}

func SlackSendSuccess(message string, yamlConf YamlConf) (err error) {
	if yamlConf.Slack.WebHookUrl != "" && yamlConf.Slack.Channel != "" {
		return SlackPush(message, "#00FF00", yamlConf.Slack.Channel, yamlConf.Slack.WebHookUrl)
	}
	return nil
}

func SlackSendError(message string, yamlConf YamlConf) (err error) {
	if yamlConf.Slack.WebHookUrl != "" && yamlConf.Slack.Channel != "" {
		return SlackPush(message, "#D0021B", yamlConf.Slack.Channel, yamlConf.Slack.WebHookUrl)
	}
	return nil
}

func SlackPush(message string, color string, channel string, webhookUrl string) (err error) {

	hostname, _ := GetHostname()
	att := SlackAttachment{Text: "*"+hostname+"*\n"+message, Color: color}
	var atts []SlackAttachment
	atts = append(atts, att)
	slackBody, _ := json.Marshal(SlackRequestBody{
		Channel: channel,
		Username: "swapper",
		Emoji: ":robot_face:",
		Attachments: atts,
	})
	req, err := http.NewRequest(http.MethodPost, webhookUrl, bytes.NewBuffer(slackBody))
	if err != nil {
		return errors.New(fmt.Sprintf(errorMessages["request_failed"], err))
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func getYamlConfFromMasters(masters []string) (yamlConf YamlConf, err error) {

	// shuffle masters array
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(masters), func(i, j int) { masters[i], masters[j] = masters[j], masters[i] })

	// Get swapper.yml from master(s)
	for _, master := range masters {
		swapperYaml := CurlSwapperYaml(master)
		if swapperYaml != "" {
			yamlConf, err := ParseSwapperYaml(swapperYaml)
			return yamlConf, err
		}
	}
	return yamlConf, errors.New(errorMessages["cannot_contact_master"])
}

type SlackRequestBody struct {
	Text string `json:"text"`
	Channel string `json:"channel"`
	Username string  `json:"username"`
	Emoji string `json:"icon_emoji"`
	Attachments []SlackAttachment `json:"attachments"`
}

type SlackAttachment struct {
	Text string `json:"text"`
	Color string `json:"color"`
}

func GetSwapperYamlFromGCP(hostname string) string {
	ctx := context.Background()

	// Creates a client.
	client, errClient := storage.NewClient(ctx)
	if errClient != nil {
		return ""
	}

	// Sets the name for the new bucket.
	hostname = strings.Replace(hostname, "gs://", "", -1)
	bucketNames := strings.Split(hostname, "/")
	if len(bucketNames) != 3 {
		return ""
	}

	bucketName := bucketNames[0]
	object := bucketNames[1]+"/swapper.yml"
	rc, err := client.Bucket(bucketName).Object(object).NewReader(ctx)
	if err != nil {
		return ""
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return ""
	}
	return string(data)
}

func CurlSwapperYaml(hostname string) string {

	if strings.Contains(hostname, "gs://") {
		return GetSwapperYamlFromGCP(hostname)
	}

	resp, err := http.Get("http://"+hostname+"/swapper.yml")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.Status != "200 OK" {
		return ""
	} else {
		return string(body)
	}
}

func main() {
	_ = os.Mkdir(yamlDirectory, 0777)
	_ = os.Mkdir(pidDirectory, 0777)

	var arg string
	var arg2 string
	if len(os.Args) == 1 {
		arg = "help"
	} else {
		arg = os.Args[1]
	}
	if len(os.Args) > 2 {
		arg2 = os.Args[2]
	} else {
		arg2 = "help"
	}

	switch arg {
	case "node":
		switch arg2 {
		case "start":
			Respond(NodeStart(os.Args[1:]))
		case "stop":
			Respond(NodeStop(os.Args[1:]))
		default:
			Respond(HelpNode())
		}
	case "master":
		switch arg2 {
		case "start":
			Respond(MasterStart(os.Args[1:]))
		case "stop":
			Respond(MasterStop(os.Args[1:]))
		default:
			Respond(HelpMaster())
		}
	case "deploy":
		Respond(Deploy(os.Args[1:]))
	case "status":
		Respond(Status())
	case "version":
		Respond(Version())
	default:
		Respond(Help())
	}
}
