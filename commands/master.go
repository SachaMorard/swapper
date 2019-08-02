package commands

import (
	"errors"
	"fmt"
	"github.com/sachamorard/swapper/response"
	"github.com/sachamorard/swapper/utils"
	"github.com/sachamorard/swapper/yaml"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/valyala/fasthttp"
)

var (
	MasterUsage = `
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

func MasterStart(argv []string) response.Response {
	arguments := MasterStartArgs(argv)
	port := arguments["--port"].(string)

	if port == "0" {
		return response.Fail(fmt.Sprintf(response.ErrorMessages["master_failed"], "port 0 is not valid"))
	}

	if arguments["--join"] == nil {
		// Master start
		err := PrepareNewMaster(port)
		if err != nil {
			return response.Fail(err.Error())
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
			return response.Fail(err.Error())
		}
		if arguments["--detach"] == false {
			return MasterJoin(port, join)
		} else {
			cmd := exec.Command("swapper","master", "start", "-p", port, "--join", join)
			_ = cmd.Start()
		}
	}
	return response.Success("Starting swapper in detach mode")
}

func PrepareNewMaster(port string) error {
	// check if at least one master is already running
	files, err := ioutil.ReadDir(PidDirectory)
	if err != nil {
		return err
	}
	var runningMasters []string
	for _, f := range files {
		if strings.Contains(f.Name(), "swapper-master-") {
			port := strings.Replace(f.Name(),"swapper-master-","", -1)
			port = strings.Replace(port,".pid","", -1)
			dat, err := ioutil.ReadFile(PidDirectory+"/"+f.Name())
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
					_ = os.Remove(PidDirectory+"/"+f.Name())
				}
			}
		}
	}
	if len(runningMasters) > 0 {
		return errors.New(response.ErrorMessages["master_already_started"])
	}

	// get the old yamlConfs if exist
	files, err = ioutil.ReadDir(YamlDirectory)
	if err != nil {
		return err
	}
	var existingYamlConf []string
	var valid = regexp.MustCompile(`\.yml_` + port + `$`)
	for _, f := range files {
		if valid.MatchString(f.Name()) {
			sourceFile := YamlDirectory+"/"+f.Name()
			oldYaml, ioErr := ioutil.ReadFile(sourceFile)
			if ioErr == nil {
				yamlConf, err := yaml.ParseSwapperYaml(string(oldYaml))
				if err != nil {
					return err
				}
				forceTime := yamlConf.Time
				splittedOldYaml := strings.Split(string(oldYaml), "\nhash: ")
				swapperYaml := splittedOldYaml[0]
				filename := strings.Replace(f.Name(), "_"+port, "", -1)
				err = WriteSwapperYaml(filename, swapperYaml, port, []string{}, forceTime)
				existingYamlConf = append(existingYamlConf, f.Name())
			}
		}
	}

	// if no (valid) file is in YamlDirectory
	if len(existingYamlConf) == 0 || utils.FileExists(YamlDirectory+"/default.yml_"+port) == false {
		// set default.yml
		forceTime := int64(0)
		err = WriteSwapperYaml("default.yml", baseYaml, port, []string{}, forceTime)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewMaster(port string) response.Response {
	fmt.Print("Swapper master is running... ")

	pid := os.Getpid()
	d1 := []byte(strconv.Itoa(pid))
	_ = ioutil.WriteFile(PidDirectory+"/swapper-master-"+port+".pid", d1, 0644)

	// refresh master Loop
	go RefreshMasterLoop(port)
	go PingMasterLoop(port)

	// launch http server
	masterPort = port
	h := masterRequestHandler
	if err := fasthttp.ListenAndServe(":"+port, h); err != nil {
		return response.Fail(fmt.Sprintf(response.ErrorMessages["master_failed"], err.Error()))
	}

	return response.Success("")
}

func PrepareJoinMaster(port string, join string) error {
	// check if a master is already running with this port
	pidFile := PidDirectory+"/swapper-master-"+port+".pid"
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
			return errors.New(fmt.Sprintf(response.ErrorMessages["wrong_port"], join))
		} else {
			_ = os.Remove(pidFile)
		}
	}

	// Check mastersHostname ports
	mastersHostname := strings.Split(join, ",")
	var masters []string
	for _, a := range mastersHostname {
		if a == "localhost" {
			a, _ = utils.GetHostname()
		}
		if a == "127.0.0.1" {
			a, _ = utils.GetHostname()
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

	// remove old yamls
	files, err := ioutil.ReadDir(YamlDirectory)
	if err != nil {
		return err
	}
	var valid = regexp.MustCompile(`\.yml_` + port + `$`)
	for _, f := range files {
		if valid.MatchString(f.Name()) {
			sourceFile := YamlDirectory+"/"+f.Name()
			_ = os.Remove(sourceFile)
		}
	}

	// Check if masters are responding and import all yamls
	var conf Conf
	for _, master := range masters {
		conf = CurlRoot(master)
		if len(conf.Yamls) != 0 {
			for _, filename := range conf.Yamls {
				swapperYaml := CurlYaml(filename, master)
				if swapperYaml != "" {
					// set yaml conf
					splitedYaml := strings.Split(swapperYaml, "\nhash: ")
					swapperYamlStr := splitedYaml[0]
					hostname, _ := utils.GetHostname()
					yamlConf, err := yaml.ParseSwapperYaml(swapperYaml)
					masters = append(yamlConf.Masters, hostname+":"+port)
					err = WriteSwapperYaml(filename, swapperYamlStr, port, masters, 0)
					if err != nil {
						return err
					}
				}
			}
			break
		}
	}
	if len(conf.Yamls) == 0 {
		return errors.New(response.ErrorMessages["cannot_contact_master"])
	}

	return nil
}

func MasterJoin(port string, join string) response.Response {
	fmt.Print("Swapper master is running... ")

	pid := os.Getpid()
	d1 := []byte(strconv.Itoa(pid))
	_ = ioutil.WriteFile(PidDirectory+"/swapper-master-"+port+".pid", d1, 0644)

	go FirstPing(port)
	// refresh master Loop
	go RefreshMasterLoop(port)
	// first Ping
	go PingMasterLoop(port)

	// launch http server
	masterPort = port
	h := masterRequestHandler
	if err := fasthttp.ListenAndServe(":"+port, h); err != nil {
		return response.Fail(fmt.Sprintf(response.ErrorMessages["master_failed"], err.Error()))
	}

	return response.Success("")
}

func PingMasterLoop(port string) {
	time.Sleep(30000 * time.Millisecond)
	PingMasters(port)
	PingMasterLoop(port)
}

func RefreshMasterLoop(port string) {
	time.Sleep(5000 * time.Millisecond)
	_ = RefreshMaster(port)
	RefreshMasterLoop(port)
}

func MasterStop(argv []string) response.Response {
	_, _ = docopt.ParseArgs(masterStopUsage, argv, "")

	files, err := ioutil.ReadDir(PidDirectory)
	if err != nil {
		return response.Fail(err.Error())
	}

	var masters []string
	for _, f := range files {
		if strings.Contains(f.Name(), "swapper-master-") {
			masters = append(masters, f.Name())
			port := strings.Replace(f.Name(),"swapper-master-","", -1)
			port = strings.Replace(port,".pid","", -1)

			hostname, _ := utils.GetHostname()
			fmt.Print("Stopping Swapper Master ("+hostname+":"+port+")... ")

			dat, err := ioutil.ReadFile(PidDirectory+"/"+f.Name())
			if err == nil {
				p := string(dat)
				pid, err := strconv.ParseInt(p, 10, 64)
				if err != nil {
					return response.Fail(err.Error())
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

				_ : os.Remove(PidDirectory+"/"+f.Name())
			}
		}
	}
	if len(masters) == 0 {
		return response.Fail(response.ErrorMessages["master_not_running"])
	}
	return response.Success("")
}
