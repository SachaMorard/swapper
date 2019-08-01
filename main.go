package main

import (
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

	switch arg {
	case "node":
		switch arg2 {
		case "start":
			response.Send(commands.NodeStart(os.Args[1:]))
		case "stop":
			response.Send(commands.NodeStop(os.Args[1:]))
		default:
			response.Send(HelpNode())
		}
	case "master":
		switch arg2 {
		case "start":
			response.Send(commands.MasterStart(os.Args[1:]))
		case "stop":
			response.Send(commands.MasterStop(os.Args[1:]))
		default:
			response.Send(HelpMaster())
		}
	case "deploy":
		response.Send(commands.Deploy(os.Args[1:]))
	case "status":
		response.Send(commands.Status())
	case "version":
		response.Send(commands.Version())
	default:
		response.Send(Help())
	}
}
