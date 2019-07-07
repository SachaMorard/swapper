package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docopt/docopt-go"
)

var (
	nodeUsage = `
swapper node COMMAND [OPTIONS].

Manage a node

Commands:
 start     Start a node and join it to the swapper cluster
 stop      Stop a node

Run 'swapper node COMMAND --help' for more information on a command.

`
	nodeStartUsage = `
swapper node start [OPTIONS].

Start a swapper node

Usage:
 swapper node start [--join <hostnames>] [--detach]
 swapper node start (-h|--help)

Options:
 -h --help                Show this screen.
 --join=HOSTNAMES         Masters' hostnames (separated by comma)
 -d --detach              Run node in background

Examples:
 To start a new node, connected with a local masters:
 $ swapper node start --join master-hostname-1

 To start a new node, connected with two local masters:
 $ swapper node start --join master-hostname-1,master-hostname-2

`
	nodeStopUsage = `
swapper node stop.

Stop node on this machine

Usage:
 swapper node stop
 swapper node stop (-h|--help)

Options:
 -h --help    Show this screen.

Examples:
 $ swapper node stop

`
)

func NodeStartArgs(argv []string) docopt.Opts {
	arguments, _ := docopt.ParseArgs(nodeStartUsage, argv, "")
	return arguments
}

func NodeStart(argv []string) Response {

	pid := os.Getpid()
	d1 := []byte(strconv.Itoa(pid))
	_ = ioutil.WriteFile(pidDirectory+"/swapper-node.pid", d1, 0644)

	arguments := NodeStartArgs(argv)

	if arguments["--join"] == nil {
		return Fail(errorMessages["need_master_addr"])
	}
	mastersHostname := strings.Split(arguments["--join"].(string), ",")

	// Check mastersHostname ports
	var masters []string
	for _, a := range mastersHostname {
		if a == "localhost" {
			a, _ = GetHostname()
		}
		if a == "127.0.0.1" {
			a, _ = GetHostname()
		}

		i := strings.Index(a, ":")
		if i == -1 {
			a = a + ":1207"
		}
		masters = append(masters, a)
	}

	// Get swapper.yml from master(s)
	yamlConf, err := getYamlConfFromMasters(masters)
	if err != nil {
		return Fail(err.Error())
	}

	// run containers
	err = runContainers(yamlConf)
	if err != nil {
		return Fail(err.Error())
	}

	// create frontend haproxy conf
	haproxyConf, err := CreateHaproxyConf(yamlConf)
	if err != nil {
		return Fail(err.Error())
	}

	// start haproxy
	err = startProxy(yamlConf)
	if err != nil {
		return Fail(err.Error())
	}

	// write file into swapper-proxy to automatically start the haproxy
	cmd := exec.Command("docker", "exec", "swapper-proxy", "bash", "-c", "echo '"+haproxyConf+"' > /app/src/haproxy.tmp.cfg")
	_, err = cmd.Output()
	if err != nil {
		return Fail(errorMessages["proxy_failed"])
	}

	currentHash = yamlConf.Hash

	// update regularly
	fmt.Println("Now, listening changes...")
	if arguments["--detach"] == false {
		ListenToMasters(yamlConf)
	} else {
		joinArg := arguments["--join"]
		cmd := exec.Command("swapper","node", "start", "--join", joinArg.(string))
		_ = cmd.Start()
	}

	return Success("")
}

func startProxy(yamlConf YamlConf) (err error) {

	Id, _ := Command("docker ps --format {{.ID}} --filter name=swapper-proxy")
	if Id == "" {
		fmt.Print("Starting swapper-proxy... ")
		var command []string
		command = append(command, "docker run --rm")
		command = append(command, "--name swapper-proxy")
		command = append(command, "--hostname swapper-proxy")
		for _, frontend := range yamlConf.Frontends  {
			command = append(command, "-p "+strconv.Itoa(frontend.Listen)+":"+strconv.Itoa(frontend.Listen))
		}
		command = append(command, "-d")
		command = append(command, "gcr.io/docker-swapper/swapper-proxy:1.0.0")

		commandStr := strings.Join(command, " ")
		_, err = Command(commandStr)
		// todo: if errors, print docker log
		if err != nil {
			return errors.New(errorMessages["proxy_failed"])
		}

		fmt.Print("Started\n")
	} else {
		fmt.Println("swapper-proxy already started")

		// Check if it's necessary to recreate proxy
		cmd := exec.Command("docker", "inspect", "--format", "{{ .Config.ExposedPorts }}", "swapper-proxy")
		out, err := cmd.Output()
		if err != nil {
			return errors.New(errorMessages["proxy_failed"])
		}
		restart := false
		for _, frontend := range yamlConf.Frontends  {
			if strings.Contains(string(out), strconv.Itoa(frontend.Listen)+"/tcp") == false {
				restart = true
			}
		}
		if restart == true {
			fmt.Println("[CAREFULL] Frontend ports changed, recreate swapper-proxy with short interruption!!!")
			_, err = Command("docker rm -f swapper-proxy")
			if err != nil {
				return errors.New(errorMessages["proxy_stop_failed"])
			}
			return startProxy(yamlConf)
		}
	}
	return err
}

func runContainers(yamlConf YamlConf) (err error) {

	for _, service := range yamlConf.Services  {
		for _, container := range service.Containers  {

			// If container image hasn't pulled yet
			Image, _ := Command("docker images " + container.Image + ":" + container.Tag + " --format {{.ID}}")
			if Image == "" {
				fmt.Printf("Pulling %s... ", container.Image + ":" + container.Tag)
				_, _ = Command("docker pull " + container.Image + ":" + container.Tag)
				fmt.Print("Pulled\n")
			}

			containerName := "swapper-container."+yamlConf.Hash+"."+container.Name+"."+strconv.Itoa(container.Index)
			Id, _ := Command("docker ps --format {{.ID}} --filter name=" + containerName)
			if Id == "" {
				fmt.Printf("Starting %s... ", containerName)
				var command []string
				command = append(command, "docker")
				command = append(command, "run")
				command = append(command, "--rm")
				command = append(command, "--name")
				command = append(command, containerName)
				command = append(command, "--hostname")
				command = append(command, containerName)
				// todo add extra commands

				if container.LoggingDriver != "" {
					command = append(command, "--log-driver")
					command = append(command, container.LoggingDriver)
				}

				for k, v := range container.LoggingOptions {
					command = append(command, "--log-opt")
					command = append(command, k.(string) + "=" + v.(string))
				}

				if container.HealthCmd != "" {
					command = append(command, "--health-cmd")
					command = append(command, container.HealthCmd)
				}

				if container.HealthInterval != "" {
					command = append(command, "--health-interval")
					command = append(command, container.HealthInterval)
				}

				if container.HealthRetries != 0 {
					command = append(command, "--health-retries")
					command = append(command, strconv.Itoa(container.HealthRetries))
				}

				if container.HealthTimeout != "" {
					command = append(command, "--health-timeout")
					command = append(command, container.HealthTimeout)
				}

				for _, v := range container.ExtraHosts {
					command = append(command, "--add-host")
					command = append(command, v.(string))
				}

				for k, v := range container.Envs {
					command = append(command, "-e")
					var value string
					if reflect.TypeOf(v).String() == "string" {
						value = v.(string)
					} else if reflect.TypeOf(v).String() == "bool" {
						if v == true {
							value = "true"
						} else {
							value = "false"
						}
					} else if reflect.TypeOf(v).String() == "int" {
						if val, ok := (v).(int); ok {
							value = strconv.Itoa(val)
						}
					}
					command = append(command, k.(string) + "=" + value)
				}
				command = append(command, "-d")
				command = append(command, container.Image+":"+container.Tag)

				cmd := exec.Command(command[0], command[1:]...)
				_, err := cmd.Output()
				// todo: if errors, print docker log
				if err != nil {
					fmt.Println(err)
					return errors.New(fmt.Sprintf(errorMessages["container_failed"], containerName))
				}

				fmt.Print("Started\n")
			} else {
				fmt.Printf("%s already started\n", containerName)
			}
		}
	}

	return err
}

func ListenToMasters(yamlConf YamlConf) {
	masters := yamlConf.Masters
	previousYamlConf := yamlConf
	time.Sleep(3000 * time.Millisecond)

	// Get swapper.yml from master(s)
	yamlConf, err := getYamlConfFromMasters(masters)
	if err != nil {
		fmt.Println(err.Error())
		_ = SlackSendError(err.Error(), previousYamlConf)
		time.Sleep(5000 * time.Millisecond)
		ListenToMasters(previousYamlConf)
		return
	}

	if yamlConf.Hash != currentHash {
		fmt.Println("\n>>> Updating node...")

		// start containers
		err = runContainers(yamlConf)
		if err != nil {
			fmt.Println(err.Error())
			_ = SlackSendError("Node failed to update\n"+err.Error(), yamlConf)
			ListenToMasters(yamlConf)
			return
		}

		// create frontend haproxy conf
		haproxyConf, err := CreateHaproxyConf(yamlConf)
		if err != nil {
			fmt.Println(err.Error())
			_ = SlackSendError("Node failed to update\n"+err.Error(), yamlConf)
			ListenToMasters(yamlConf)
			return
		}

		// start haproxy if necessary
		err = startProxy(yamlConf)
		if err != nil {
			fmt.Println(err.Error())
			_ = SlackSendError("Node failed to update\n"+err.Error(), yamlConf)
			ListenToMasters(yamlConf)
			return
		}

		// write file into swapper-proxy to automatically reload the haproxy
		cmd := exec.Command("docker", "exec", "swapper-proxy", "bash", "-c", "echo '"+haproxyConf+"' > /app/src/haproxy.tmp.cfg")
		_, err = cmd.Output()
		if err != nil {
			// todo rollback ?
			fmt.Println(errorMessages["proxy_failed"])
			_ = SlackSendError("Node failed to update\n"+errorMessages["proxy_failed"], yamlConf)
			ListenToMasters(yamlConf)
			return
		}

		// update currentHash
		currentHash = yamlConf.Hash

		// remove old containers and images
		fmt.Println("Remove unused nodes")
		cmd = exec.Command("docker", "container", "ls", "--format", "{{.ID}} {{.Names}}", "--filter", "name=swapper-container.")
		out, err := cmd.Output()
		if err != nil {
			fmt.Println(err.Error())
			ListenToMasters(yamlConf)
			return
		}
		if strings.TrimSpace(string(out)) == "" {
			ListenToMasters(yamlConf)
			return
		}
		psStr := strings.Split(strings.TrimSpace(string(out)), "\n")

		var rmCmd []string
		rmCmd = append(rmCmd, "docker")
		rmCmd = append(rmCmd, "rm")
		rmCmd = append(rmCmd, "-f")
		rmCmd = append(rmCmd, "-v")
		for _, v := range psStr  {
			containerPs := strings.Split(v, " ")
			if strings.Contains(containerPs[1], "swapper-container." + yamlConf.Hash) == false {
				rmCmd = append(rmCmd, containerPs[0])
			}
		}
		cmd = exec.Command(rmCmd[0], rmCmd[1:]...)
		out, err = cmd.Output()
		if err != nil {
			fmt.Println(err.Error())
			ListenToMasters(yamlConf)
			return
		}

		// remove unused docker images to save space
		fmt.Println("Remove unused images")
		cmd = exec.Command("docker", "system", "prune", "--all", "--force")
		out, err = cmd.Output()
		if err != nil {
			fmt.Println(err.Error())
			ListenToMasters(yamlConf)
			return
		}

		fmt.Println(">>> Node updated")
		_ = SlackSendSuccess("Node updated", yamlConf)

	}

	ListenToMasters(yamlConf)
}

func NodeStop(argv []string) Response {
	_, _ = docopt.ParseArgs(nodeStopUsage, argv, "")

	dat, err := ioutil.ReadFile(pidDirectory+"/swapper-node.pid")
	if err == nil {
		fmt.Print("Stopping swapper-node... ")
		p := string(dat)
		pid, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return Fail(err.Error())
		}
		proc, err := os.FindProcess(int(pid))

		//double check if process is running and alive
		//by sending a signal 0
		//NOTE : syscall.Signal is not available in Windows
		err = proc.Signal(syscall.Signal(0))
		if err == nil {
			err = proc.Kill()
			if err == nil {
				fmt.Print("Stopped\n")
			} else {
				fmt.Print("Already Stopped\n")
			}
		} else {
			fmt.Print("Already Stopped\n")
		}

		if err == syscall.ESRCH {
			fmt.Print("Already Stopped\n")
		}

		_ : os.Remove(pidDirectory+"/swapper-node.pid")
	}

	fmt.Print("Stopping swapper-proxy... ")
	command := "docker stop swapper-proxy"
	out, _ := Command(command)
	if out != "" {
		fmt.Println("Stopped")
	}

	fmt.Print("Stopping swapper-container(s)... ")
	out, _ = Command("docker container ls --format {{.ID}} --filter name=swapper-container")
	if out == "" {
		return Fail(errorMessages["containers_not_running"])
	}

	ids := strings.Replace(out, "\n", " ", -1)
	command = "docker stop " + strings.TrimSpace(ids)
	out, _ = Command(command)
	return Success("Stopped\n")
}
