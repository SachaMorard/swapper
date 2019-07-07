package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func Status() Response {
	fmt.Println("")
	fmt.Println("MASTER(S)")
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
					fmt.Println(hostname+":"+port+" is running (pid: "+p+")")
				} else {
					_ : os.Remove(pidDirectory+"/"+f.Name())
				}
			}
		}
	}
	if len(masters) == 0 {
		fmt.Println("-")
	}

	fmt.Println("")
	fmt.Println("NODE")
	dat, err := ioutil.ReadFile(pidDirectory+"/swapper-node.pid")
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
			fmt.Println("swapper-node is running (pid: "+p+")")
		} else {
			_ : os.Remove(pidDirectory+"/swapper-node.pid")
		}
	} else {
		fmt.Println("-")
	}

	fmt.Println("")
	fmt.Println("PROXY")
	Ports, _ := Command("docker ps --format {{.Ports}} --filter name=swapper-proxy")
	if Ports != "" {
		fmt.Println("swapper-proxy is running (ports: "+strings.Replace(Ports, "\n", "", -1)+")")
	} else {
		fmt.Println("-")
	}

	fmt.Println("")
	Ids, _ := Command("docker ps --format {{.ID}} --filter name=swapper-container.")
	if Ids != "" {
		containers, _ := Command("docker ps --filter name=swapper-container.")
		fmt.Println(containers)
	} else {
		fmt.Println("CONTAINER(S)")
		fmt.Println("-")
	}

	return Success("")
}

