package commands

import (
	"fmt"
	"github.com/sachamorard/swapper/response"
	"github.com/sachamorard/swapper/utils"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func Status() response.Response {
	fmt.Println("")
	fmt.Println("MASTER(S)")
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
					fmt.Println(hostname+":"+port+" is running (pid: "+p+")")
				} else {
					_ : os.Remove(PidDirectory+"/"+f.Name())
				}
			}
		}
	}
	if len(masters) == 0 {
		fmt.Println("-")
	}

	fmt.Println("")
	fmt.Println("NODE")
	dat, err := ioutil.ReadFile(PidDirectory+"/swapper-node.pid")
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
			fmt.Println("swapper-node is running (pid: "+p+")")
		} else {
			_ : os.Remove(PidDirectory+"/swapper-node.pid")
		}
	} else {
		fmt.Println("-")
	}

	fmt.Println("")
	fmt.Println("PROXY")
	Ports, _ := utils.Command("docker ps --format {{.Ports}} --filter name=swapper-proxy")
	if Ports != "" {
		fmt.Println("swapper-proxy is running (ports: "+strings.Replace(Ports, "\n", "", -1)+")")
	} else {
		fmt.Println("-")
	}

	fmt.Println("")
	Ids, _ := utils.Command("docker ps --format {{.ID}} --filter name=swapper-container.")
	if Ids != "" {
		containers, _ := utils.Command("docker ps --filter name=swapper-container.")
		fmt.Println(containers)
	} else {
		fmt.Println("CONTAINER(S)")
		fmt.Println("-")
	}

	return response.Success("")
}

