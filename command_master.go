package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/valyala/fasthttp"
)

var (
	masterUsage = `
swapper master COMMAND [OPTIONS].

Manage a master

Commands:
 start     Start a master
 stop      Stop a master

Run 'swapper master COMMAND --help' for more information on a command.

`
	masterStartUsage = `
swapper master start [OPTIONS].

Start a swapper master

Usage:
 swapper master start [--join <hostnames>] [-p <port>] [--detach]
 swapper master start (-h|--help)

Options:
 -h --help                Show this screen.
 -p PORT --port=PORT      Master's port [default: 1207]
 --join=HOSTNAMES         Masters' hostnames (separated by comma)
 -d --detach              Run master in background

Examples:
 To start a master, execute:
 $ swapper master start

 To start a master and join it to an other, execute:
 $ swapper master start --join master-hostname-1

 To start a master and join it to many others, execute:
 $ swapper master start --join master-hostname-1,master-hostname-2

 To start a master with custom port:
 $ swapper master start -p 1208

`
	masterStopUsage = `
swapper master stop.

Stop master(s) on this machine

Usage:
 swapper master stop
 swapper master stop (-h|--help)

Options:
 -h --help    Show this screen.

Examples:
 $ swapper master stop

`
	masterPort = "1207"
)

func MasterStartArgs(argv []string) docopt.Opts {
	arguments, _ := docopt.ParseArgs(masterStartUsage, argv, "")
	return arguments
}

func MasterStart(argv []string) Response {
	arguments := MasterStartArgs(argv)
	port := arguments["--port"].(string)

	if port == "0" {
		return Fail(fmt.Sprintf(errorMessages["master_failed"], "port 0 is not valid"))
	}

	if arguments["--join"] == nil {
		// Master start
		err := PrepareNewMaster(port)
		if err != nil {
			return Fail(err.Error())
		}
		if arguments["--detach"] == false {
			return NewMaster(port)
		} else {
			cmd := exec.Command("swapper","master", "start", "-p", port)
			_ = cmd.Start()
		}
	} else {
		// Master join
		join := arguments["--join"].(string)
		err := PrepareJoinMaster(port, join)
		if err != nil {
			return Fail(err.Error())
		}
		if arguments["--detach"] == false {
			return MasterJoin(port, join)
		} else {
			cmd := exec.Command("swapper","master", "start", "-p", port, "--join", join)
			_ = cmd.Start()
		}
	}
	return Success("Starting swapper in detach mode")
}

func PrepareNewMaster(port string) error {
	// check if at least one master is already running
	files, err := ioutil.ReadDir(pidDirectory)
	if err != nil {
		return err
	}
	var runningMasters []string
	for _, f := range files {
		if strings.Contains(f.Name(), "swapper-master-") {
			port := strings.Replace(f.Name(),"swapper-master-","", -1)
			port = strings.Replace(port,".pid","", -1)
			dat, err := ioutil.ReadFile(pidDirectory+"/"+f.Name())
			if err == nil {
				p := string(dat)
				pid, err := strconv.ParseInt(p, 10, 64)
				if err != nil {
					return err
				}
				proc, err := os.FindProcess(int(pid))

				//double check if process is running and alive
				//by sending a signal 0
				//NOTE : syscall.Signal is not available in Windows
				err = proc.Signal(syscall.Signal(0))
				if err == nil {
					runningMasters = append(runningMasters, f.Name())
				} else {
					_ = os.Remove(pidDirectory+"/"+f.Name())
				}
			}
		}
	}
	if len(runningMasters) > 0 {
		return errors.New(errorMessages["master_already_started"])
	}

	// get the old one swapper.yml if exists
	sourceFile := yamlDirectory+"/swapper-"+port+".yml"
	swapperYaml := firstYaml
	forceTime := int64(0)
	if _, err := os.Stat(sourceFile); err == nil {
		oldYaml, ioErr := ioutil.ReadFile(sourceFile)
		if ioErr == nil {
			yamlConf, err := ParseSwapperYaml(string(oldYaml))
			if err != nil {
				return err
			}
			forceTime = yamlConf.Time
			splittedOldYaml := strings.Split(string(oldYaml), "\nhash: ")
			swapperYaml = splittedOldYaml[0]
		}
	}

	// set swapper.yml
	err = WriteSwapperYaml(swapperYaml, port, []string{}, forceTime)
	if err != nil {
		return err
	}

	return nil
}

func NewMaster(port string) Response {
	fmt.Print("Swapper master is running... ")

	pid := os.Getpid()
	d1 := []byte(strconv.Itoa(pid))
	_ = ioutil.WriteFile(pidDirectory+"/swapper-master-"+port+".pid", d1, 0644)

	// refresh master Loop
	go RefreshMasterLoop(port)
	go PingMasterLoop(port)

	// launch http server
	masterPort = port
	h := masterRequestHandler
	if err := fasthttp.ListenAndServe(":"+port, h); err != nil {
		return Fail(fmt.Sprintf(errorMessages["master_failed"], err.Error()))
	}

	return Success("")
}

func PrepareJoinMaster(port string, join string) error {
	// check if a master is already running with this port
	pidFile := pidDirectory+"/swapper-master-"+port+".pid"
	dat, err := ioutil.ReadFile(pidFile)
	if err == nil {
		p := string(dat)
		pid, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return err
		}
		proc, err := os.FindProcess(int(pid))

		//double check if process is running and alive
		//by sending a signal 0
		//NOTE : syscall.Signal is not available in Windows
		err = proc.Signal(syscall.Signal(0))
		if err == nil {
			return errors.New(fmt.Sprintf(errorMessages["wrong_port"], join))
		} else {
			_ = os.Remove(pidFile)
		}
	}

	// Check mastersHostname ports
	mastersHostname := strings.Split(join, ",")
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

	// shuffle masters array
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(masters), func(i, j int) { masters[i], masters[j] = masters[j], masters[i] })

	// Get swapper.yml from master(s)
	swapperYaml := ""
	for _, master := range masters {
		swapperYaml = CurlSwapperYaml(master)
		if swapperYaml != "" {
			break
		}
	}
	if swapperYaml == "" {
		return errors.New(errorMessages["cannot_contact_master"])
	}

	// remove old swapper.yml
	sourceFile := yamlDirectory+"/swapper-"+port+".yml"
	_ = os.Remove(sourceFile)

	// set swapper.yml
	splitedYaml := strings.Split(swapperYaml, "\nhash: ")
	swapperYamlStr := splitedYaml[0]
	hostname, _ := GetHostname()
	yamlConf, err := ParseSwapperYaml(swapperYaml)
	masters = append(yamlConf.Masters, hostname+":"+port)
	err = WriteSwapperYaml(swapperYamlStr, port, masters, 0)
	if err != nil {
		return err
	}

	return nil
}

func MasterJoin(port string, join string) Response {
	fmt.Print("Swapper master is running... ")

	pid := os.Getpid()
	d1 := []byte(strconv.Itoa(pid))
	_ = ioutil.WriteFile(pidDirectory+"/swapper-master-"+port+".pid", d1, 0644)

	go FirstPing(port)
	// refresh master Loop
	go RefreshMasterLoop(port)
	// first Ping
	go PingMasterLoop(port)

	// launch http server
	masterPort = port
	h := masterRequestHandler
	if err := fasthttp.ListenAndServe(":"+port, h); err != nil {
		return Fail(fmt.Sprintf(errorMessages["master_failed"], err.Error()))
	}

	return Success("")
}

func PingMasterLoop(port string) {
	time.Sleep(30000 * time.Millisecond)
	PingMasters(port)
	PingMasterLoop(port)
}

func RefreshMasterLoop(port string) {
	time.Sleep(5000 * time.Millisecond)
	RefreshMaster(port)
	RefreshMasterLoop(port)
}

func MasterStop(argv []string) Response {
	_, _ = docopt.ParseArgs(masterStopUsage, argv, "")

	files, err := ioutil.ReadDir(pidDirectory)
	if err != nil {
		return Fail(err.Error())
	}

	var masters []string
	for _, f := range files {
		if strings.Contains(f.Name(), "swapper-master-") {
			masters = append(masters, f.Name())
			port := strings.Replace(f.Name(),"swapper-master-","", -1)
			port = strings.Replace(port,".pid","", -1)

			hostname, _ := GetHostname()
			fmt.Print("Stopping Swapper Master ("+hostname+":"+port+")... ")

			dat, err := ioutil.ReadFile(pidDirectory+"/"+f.Name())
			if err == nil {
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

				_ : os.Remove(pidDirectory+"/"+f.Name())
			}
		}
	}
	if len(masters) == 0 {
		return Fail(errorMessages["master_not_running"])
	}
	return Success("")
}
