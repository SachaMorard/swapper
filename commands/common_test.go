package commands

import (
	"fmt"
	"github.com/sachamorard/swapper/response"
	"github.com/sachamorard/swapper/utils"
	"github.com/sachamorard/swapper/yaml"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCreateHaproxyConf(t *testing.T) {
	vars := []string{"NGINXTAG=1.17.0", "ENV=prod"}
	cleanYaml, err := yaml.PrepareSwapperYaml("../yaml/tests/v1/valid.1.yml", vars)
	if err != nil {
		t.Fail()
	}

	yamlConf, err := yaml.ParseSwapperYaml(cleanYaml)
	if err != nil {
		t.Fail()
	}

	_, err = CreateHaproxyConf(yamlConf)
	if err != nil && err.Error() != fmt.Sprintf(response.ErrorMessages["container_failed"], "swapper-container..nginx2.0") &&
		err.Error() != fmt.Sprintf(response.ErrorMessages["container_failed"], "swapper-container..nginx.0") {
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
		outIp, _ := utils.Command(ipCommand)
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

func TestWriteSwapperYaml(t *testing.T) {
	err := WriteSwapperYaml("default.yml","jklfd fdsf: fds", "1207", []string{"ok", "c", "a", "c"}, 0)
	if err != nil && err.Error() != response.ErrorMessages["yaml_version"] {
		t.Fail()
	}

	err = WriteSwapperYaml("default.yml", baseYaml, "1207", []string{"ok", "c", "a", "c"}, 0)
	if err != nil {
		t.Fail()
	}
}

func TestGetQuorum(t *testing.T) {
	quorum := GetQuorum([]string{"host1", "host2", "host3", "host4", "host5", "host6", "host7"}, "1207")
	if len(quorum) != 4 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1", "host2", "host3", "host4", "host5", "host6"}, "1207")
	if len(quorum) != 4 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1", "host2", "host3", "host4", "host5"}, "1207")
	if len(quorum) != 3 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1", "host2", "host3", "host4"}, "1207")
	if len(quorum) != 3 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1", "host2", "host3"}, "1207")
	if len(quorum) != 2 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1", "host2"}, "1207")
	if len(quorum) != 2 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1"}, "1207")
	if len(quorum) != 0 {
		t.Fail()
	}
}

func TestRefreshMaster(t *testing.T) {

	port := "1111"
	sourceFile := YamlDirectory+"/default.yml_"+port
	swapperYaml := `
version: "1"

services:
  hello:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: latest
masters:
  - host1:1207
  - host2:1207
  - host3:1207
  - host4:1207
  - host5:1207
`
	_ = ioutil.WriteFile(sourceFile, []byte(swapperYaml), 0644)

	err := RefreshMaster(port)
	if err != nil {
		t.Fail()
	}

	_ = os.Remove(sourceFile)
}

func TestPingMasters(t *testing.T) {
	port := "1111"
	sourceFile := YamlDirectory+"/default.yml_"+port
	swapperYaml := `
version: "1"

services:
  hello:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: latest
masters:
  - host1:1207
  - host2:1207
  - host3:1207
  - host4:1207
  - host5:1207
`
	_ = ioutil.WriteFile(sourceFile, []byte(swapperYaml), 0644)

	quorum := PingMasters(port)
	if len(quorum) != 3 {
		t.Fail()
	}

	_ = os.Remove(sourceFile)
}

func TestAddMaster(t *testing.T) {
	port := "1111"
	sourceFile := YamlDirectory+"/default.yml_"+port
	swapperYaml := `
version: "1"

services:
  hello:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: latest
masters:
  - host1:1207
  - host2:1207
  - host3:1207
  - host4:1207
  - host5:1207
`

	swapperYamlExpected := `
version: "1"

services:
  hello:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: latest
masters: 
  - ahost:1
  - host1:1207
  - host2:1207
  - host3:1207
  - host4:1207
  - host5:1207
  - ok
  - ok2`

	_ = ioutil.WriteFile(sourceFile, []byte(swapperYaml), 0644)
	AddMaster([]string{"ok", "ok2", "host1:1207", "ahost:1"}, port)

	swapperYamlNew, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		t.Fail()
	}

	if string(swapperYamlNew) != swapperYamlExpected {
		t.Fail()
	}
	_ = os.Remove(sourceFile)
}
