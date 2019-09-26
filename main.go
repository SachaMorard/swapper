package main

import (
	"fmt"
	"github.com/sachamorard/swapper/commands"
	"github.com/sachamorard/swapper/response"
	"os"
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
 upgrade    Upgrade version of swapper

Run 'swapper COMMAND --help' for more information on a command.

`
)

func Help() response.Response {
	return response.Success(usage)
}

func HelpMaster() response.Response {
	return response.Success(commands.MasterUsage)
}
func HelpNode() response.Response {
	return response.Success(commands.NodeUsage)
}

func main() {
	_ = os.Mkdir(commands.YamlDirectory, 0777)
	_ = os.Mkdir(commands.PidDirectory, 0777)

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

	var response response.Response
	switch arg {
	case "node":
		switch arg2 {
		case "start":
			response = commands.NodeStart(os.Args[1:])
		case "stop":
			response = commands.NodeStop(os.Args[1:])
		default:
			response = HelpNode()
		}
	case "master":
		switch arg2 {
		case "start":
			response = commands.MasterStart(os.Args[1:])
		case "stop":
			response = commands.MasterStop(os.Args[1:])
		default:
			response = HelpMaster()
		}
	case "deploy":
		response = commands.Deploy(os.Args[1:])
	case "status":
		response = commands.Status()
	case "version":
		response = commands.Version()
	case "upgrade":
		response = commands.Upgrade(os.Args[1:])
	default:
		response = Help()
	}
	fmt.Println(response.Message)
	os.Exit(response.Code)
}
