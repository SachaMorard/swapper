package main

import (
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestHelp(t *testing.T) {
	response := Help()
	if response.Message != usage {
		t.Fail()
	}
	if response.Code != 0 {
		t.Fail()
	}
}

func TestHelpMaster(t *testing.T) {
	response := HelpMaster()
	if response.Message != masterUsage {
		t.Fail()
	}
	if response.Code != 0 {
		t.Fail()
	}
}
func TestHelpNode(t *testing.T) {
	response := HelpNode()
	if response.Message != nodeUsage {
		t.Fail()
	}
	if response.Code != 0 {
		t.Fail()
	}
}

func TestGetHostname(t *testing.T) {
	_, err := GetHostname()
	if err != nil {
		t.Fail()
	}
}

func TestCommand(t *testing.T) {
	out, _ := Command("echo hello")
	if out != "hello\n" {
		t.Fail()
	}

	_, err := Command("ecsho hello")
	if err == nil {
		t.Fail()
	}
}

func TestSuccess(t *testing.T) {
	response := Success("ok")
	if response.Message != "ok" {
		t.Fail()
	}
	if response.Code != 0 {
		t.Fail()
	}
}

func TestFail(t *testing.T) {
	response := Fail("ko")
	if response.Message != "ko" {
		t.Fail()
	}
	if response.Code != 1 {
		t.Fail()
	}
}

func TestFileExists(t *testing.T) {
	exists := FileExists("doc/swapper.example.yml")
	if exists == false {
		t.Fail()
	}

	exists = FileExists("doc/swappers.example.yml")
	if exists == true {
		t.Fail()
	}
}

func TestShutUpOut(t *testing.T) {
	oldOut := ShutUpOut()
	fmt.Println("nothing")
	RestoreOut(oldOut)
}

func TestPrepareSwapperYaml(t *testing.T) {
	vars := []string{}
	_, err := PrepareSwapperYaml("doc/swappers.example.yml", vars)
	if err.Error() != fmt.Sprintf(errorMessages["file_not_exist"], "doc/swappers.example.yml") {
		t.Fail()
	}

	_, err = PrepareSwapperYaml("tests/v1/swapper.invalid.yml", vars)
	if err.Error() != errorMessages["yaml_invalid"] {
		t.Fail()
	}

	_, err = PrepareSwapperYaml("doc/swapper.example.yml", vars)
	if err.Error() != fmt.Sprintf(errorMessages["var_missing"], "ENV TAG", "--var ENV=<value> --var TAG=<value>") {
		t.Fail()
	}

	vars = []string{"TAG=1.0.1"}
	_, err = PrepareSwapperYaml("doc/swapper.example.yml", vars)
	if err.Error() != fmt.Sprintf(errorMessages["var_missing"], "ENV", "--var ENV=<value>") {
		t.Fail()
	}

	vars = []string{"TAG=1.0.1", "ENV=prod"}
	_, err = PrepareSwapperYaml("doc/swapper.example.yml", vars)
	if err != nil {
		t.Fail()
	}
}

func TestCreateHaproxyConf(t *testing.T) {
	vars := []string{"NGINXTAG=1.17.0", "ENV=prod"}
	cleanYaml, err := PrepareSwapperYaml("tests/v1/swapper.valid.1.yml", vars)
	if err != nil {
		t.Fail()
	}

	yamlConf, err := ParseSwapperYaml(cleanYaml)
	if err != nil {
		t.Fail()
	}

	_, err = CreateHaproxyConf(yamlConf)
	if err.Error() != fmt.Sprintf(errorMessages["container_failed"], "swapper-container..nginx2.0") &&
		err.Error() != fmt.Sprintf(errorMessages["container_failed"], "swapper-container..nginx.0") {
		t.Fail()
	}

	names := []string{"swapper-container..nginx.0", "swapper-container..nginx.1", "swapper-container..nginx2.0"}
	for _, name := range names {
		var command []string
		command = append(command, "docker")
		command = append(command, "rm")
		command = append(command, "-f")
		command = append(command, name)
		cmd := exec.Command(command[0], command[1:]...)
		_, err = cmd.Output()
	}
	for _, name := range names {
		var command []string
		command = append(command, "docker")
		command = append(command, "run")
		command = append(command, "--rm")
		command = append(command, "--name")
		command = append(command, name)
		command = append(command, "--hostname")
		command = append(command, name)
		command = append(command, "-d")
		command = append(command, "nginx:latest")
		cmd := exec.Command(command[0], command[1:]...)
		_, err = cmd.Output()
	}

	ips := map[string]string{}
	for _, name := range names {
		ipCommand := "docker inspect -f {{.NetworkSettings.IPAddress}} " + name
		outIp, _ := Command(ipCommand)
		ips[name] = strings.TrimSpace(outIp)
	}

	compare1 := `
global
    log 127.0.0.1 local5 debug

defaults
    log     global
    option  dontlognull
    timeout connect 5000
    timeout client  50000
    timeout server  50000

frontend frontend_80
    option forwardfor
    mode tcp
    option tcplog
    maxconn 800
    bind 0.0.0.0:80
    default_backend backend_80_80

frontend frontend_443
    option forwardfor
    mode tcp
    option tcplog
    maxconn 800
    bind 0.0.0.0:443
    default_backend backend_443_443

frontend frontend_800
    option forwardfor
    mode tcp
    option tcplog
    maxconn 800
    bind 0.0.0.0:800
    default_backend backend_800_80

frontend frontend_200
    option forwardfor
    mode tcp
    option tcplog
    maxconn 800
    bind 0.0.0.0:200
    default_backend backend_200_80

backend backend_80_80
    balance roundrobin
    server container_0 {{nginx.0}}:80 check observe layer4 weight 100
    server container_1 {{nginx.1}}:80 check observe layer4 weight 100
backend backend_443_443
    balance roundrobin
    server container_0 {{nginx.0}}:443 check observe layer4 weight 100
    server container_1 {{nginx.1}}:443 check observe layer4 weight 100
backend backend_800_80
    balance roundrobin
    server container_0 {{nginx2.0}}:80 check observe layer4 weight 100
backend backend_200_80
    balance roundrobin
    server container_0 {{nginx2.0}}:80 check observe layer4 weight 100`

	compare2 := `
global
    log 127.0.0.1 local5 debug

defaults
    log     global
    option  dontlognull
    timeout connect 5000
    timeout client  50000
    timeout server  50000

frontend frontend_800
    option forwardfor
    mode tcp
    option tcplog
    maxconn 800
    bind 0.0.0.0:800
    default_backend backend_800_80

frontend frontend_200
    option forwardfor
    mode tcp
    option tcplog
    maxconn 800
    bind 0.0.0.0:200
    default_backend backend_200_80

frontend frontend_80
    option forwardfor
    mode tcp
    option tcplog
    maxconn 800
    bind 0.0.0.0:80
    default_backend backend_80_80

frontend frontend_443
    option forwardfor
    mode tcp
    option tcplog
    maxconn 800
    bind 0.0.0.0:443
    default_backend backend_443_443

backend backend_800_80
    balance roundrobin
    server container_0 {{nginx2.0}}:80 check observe layer4 weight 100
backend backend_200_80
    balance roundrobin
    server container_0 {{nginx2.0}}:80 check observe layer4 weight 100
backend backend_80_80
    balance roundrobin
    server container_0 {{nginx.0}}:80 check observe layer4 weight 100
    server container_1 {{nginx.1}}:80 check observe layer4 weight 100
backend backend_443_443
    balance roundrobin
    server container_0 {{nginx.0}}:443 check observe layer4 weight 100
    server container_1 {{nginx.1}}:443 check observe layer4 weight 100`

	compare1 = strings.Replace(compare1, "{{nginx.0}}", ips["swapper-container..nginx.0"], -1)
	compare1 = strings.Replace(compare1, "{{nginx.1}}", ips["swapper-container..nginx.1"], -1)
	compare1 = strings.Replace(compare1, "{{nginx2.0}}", ips["swapper-container..nginx2.0"], -1)

	compare2 = strings.Replace(compare2, "{{nginx.0}}", ips["swapper-container..nginx.0"], -1)
	compare2 = strings.Replace(compare2, "{{nginx.1}}", ips["swapper-container..nginx.1"], -1)
	compare2 = strings.Replace(compare2, "{{nginx2.0}}", ips["swapper-container..nginx2.0"], -1)

	conf, err := CreateHaproxyConf(yamlConf)
	if conf != compare1 && conf != compare2 {
		t.Fail()
	}

	for _, name := range names {
		var command []string
		command = append(command, "docker")
		command = append(command, "rm")
		command = append(command, "-f")
		command = append(command, name)
		cmd := exec.Command(command[0], command[1:]...)
		_, err = cmd.Output()
	}
}

func TestSlackPush(t *testing.T) {
	err := SlackPush("test", "#FF0000", "test", "https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS")
	if err.Error() != "Post https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS: dial tcp: lookup hooks.slackss.com: no such host" {
		t.Fail()
	}
}

func TestSlackSendSuccess(t *testing.T) {
	slack := Slack{WebHookUrl: "https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS", Channel: "test"}
	yamlConf := YamlConf{Slack: slack}
	err := SlackSendSuccess("test", yamlConf)
	if err.Error() != "Post https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS: dial tcp: lookup hooks.slackss.com: no such host" {
		t.Fail()
	}
}

func TestSlackSendError(t *testing.T) {
	slack := Slack{WebHookUrl: "https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS", Channel: "test"}
	yamlConf := YamlConf{Slack: slack}
	err := SlackSendError("test", yamlConf)
	if err.Error() != "Post https://hooks.slackss.com/services/SJKLDS/JKLDS/QLIKJS: dial tcp: lookup hooks.slackss.com: no such host" {
		t.Fail()
	}
}

func TestInterfaceToArray(t *testing.T) {
	var args interface{}
	args = "[TAG=ok ENV=prod]"
	envs := InterfaceToArray(args)
	var vs = []string{}
	if reflect.TypeOf(envs) != reflect.TypeOf(vs) {
		t.Fail()
	}
}
