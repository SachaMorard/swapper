package commands

import (
	"github.com/docopt/docopt-go"
	"github.com/sachamorard/swapper/response"
	"github.com/sachamorard/swapper/utils"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)


func TestNodeStart(t *testing.T) {
	_ = os.Remove(YamlDirectory+"/swapper-1207.yml")
	// Stop running master (if exists)
	oldOut := utils.ShutUpOut()
	args := []string{"master", "stop"}
	MasterStop(args)
	utils.RestoreOut(oldOut)

	args = []string{"node", "start"}
	oldOut = utils.ShutUpOut()
	resp := NodeStart(args)
	utils.RestoreOut(oldOut)
	if resp.Message != response.ErrorMessages["need_master_addr"] {
		t.Fail()
	}

	args = []string{"node", "start", "--join", "localhost"}
	oldOut = utils.ShutUpOut()
	resp = NodeStart(args)
	utils.RestoreOut(oldOut)
	if resp.Message != response.ErrorMessages["cannot_contact_master"] {
		t.Fail()
	}
}


func TestNodeStart2(t *testing.T) {
	// Now start master
	args := []string{"master", "start", "-d"}
	oldOut := utils.ShutUpOut()
	response := MasterStart(args)
	utils.RestoreOut(oldOut)
	if response.Code != 0 {
		t.Fail()
	}

	time.Sleep(2000 * time.Millisecond)

	args = []string{"node", "start", "--join", "localhost", "-d"}
	oldOut = utils.ShutUpOut()
	response = NodeStart(args)
	utils.RestoreOut(oldOut)

	time.Sleep(3000 * time.Millisecond)
	oldOut = utils.ShutUpOut()
	args = []string{"node", "stop"}
	NodeStop(args)
	utils.RestoreOut(oldOut)
}

func TestCurlSwapperYaml(t *testing.T) {
	swapperYaml := CurlSwapperYaml("localhost:1207")
	if swapperYaml == "" {
		t.Fail()
	}
}

func TestReplaceCommandIfExist(t *testing.T) {
	str, err := ReplaceCommandIfExist("$(hostname):24224")
	if err != nil {
		t.Fail()
	}
	hostname, _ := utils.GetHostname()
	if str != hostname+":24224" {
		t.Fail()
	}

	input, _ := ioutil.ReadFile("../doc/swapper.yml.examples/7.swapper.with.command.yml")
	str, err = ReplaceCommandIfExist(string(input))
	if err != nil {
		t.Fail()
	}
}

func TestNodeStartArgs(t *testing.T) {
	argv := []string{"node", "start"}
	arguments := NodeStartArgs(argv)
	args := docopt.Opts{
		"--join":    nil,
		"--detach":  false,
		"--help":    false,
		"start":     true,
		"node":      true,
	}
	eq := reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}

	argv = []string{"node", "start", "--join", "localhost"}
	arguments = NodeStartArgs(argv)
	args = docopt.Opts{
		"--join":    "localhost",
		"--detach":  false,
		"--help":    false,
		"start":     true,
		"node":      true,
	}
	eq = reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}

	argv = []string{"node", "start", "--join", "localhost", "-d"}
	arguments = NodeStartArgs(argv)
	args = docopt.Opts{
		"--join":    "localhost",
		"--detach":  true,
		"--help":    false,
		"start":     true,
		"node":      true,
	}
	eq = reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}

	argv = []string{"node", "start", "--join", "localhost", "--detach"}
	arguments = NodeStartArgs(argv)
	args = docopt.Opts{
		"--join":    "localhost",
		"--detach":  true,
		"--help":    false,
		"start":     true,
		"node":      true,
	}
	eq = reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}
}
