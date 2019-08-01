package commands

import (
	"fmt"
	"github.com/sachamorard/swapper/response"
	"github.com/sachamorard/swapper/utils"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/docopt/docopt-go"
)

func TestMasterStart(t *testing.T) {
	_ = os.Remove(YamlDirectory+"/swapper-1207.yml")
	// Stop running master (if exists)
	oldOut := utils.ShutUpOut()
	args := []string{"master", "stop"}
	MasterStop(args)
	utils.RestoreOut(oldOut)
}

func TestMasterStart2(t *testing.T) {
	// Now start master
	args := []string{"master", "start", "-p", "0"}
	oldOut := utils.ShutUpOut()
	resp := MasterStart(args)
	utils.RestoreOut(oldOut)
	if resp.Message != fmt.Sprintf(response.ErrorMessages["master_failed"], "port 0 is not valid") {
		t.Fail()
	}
}

func TestMasterStart3(t *testing.T) {
	// Now start master
	args := []string{"master", "start", "-d"}
	oldOut := utils.ShutUpOut()
	resp := MasterStart(args)
	utils.RestoreOut(oldOut)
	if resp.Code != 0 {
		t.Fail()
	}
	time.Sleep(2000 * time.Millisecond)

	// Now start master
	args = []string{"master", "start", "-d"}
	oldOut = utils.ShutUpOut()
	resp = MasterStart(args)
	utils.RestoreOut(oldOut)
	if resp.Message != response.ErrorMessages["master_already_started"] {
		t.Fail()
	}

	// Stop running master (if exists)
	oldOut = utils.ShutUpOut()
	args = []string{"master", "stop"}
	resp = MasterStop(args)
	utils.RestoreOut(oldOut)
	if resp.Code != 0 {
		t.Fail()
	}
}

func TestMasterJoin(t *testing.T) {
	args := []string{"master", "start", "-d"}
	oldOut := utils.ShutUpOut()
	resp := MasterStart(args)
	utils.RestoreOut(oldOut)
	if resp.Code != 0 {
		t.Fail()
	}

	time.Sleep(2000 * time.Millisecond)

	args = []string{"master", "start", "-d", "--join", "jklf", "-p", "1208"}
	oldOut = utils.ShutUpOut()
	resp = MasterStart(args)
	utils.RestoreOut(oldOut)
	if resp.Message != response.ErrorMessages["cannot_contact_master"] {
		t.Fail()
	}

	hostname, _ := utils.GetHostname()
	args = []string{"master", "start", "-d", "--join", hostname}
	oldOut = utils.ShutUpOut()
	resp = MasterStart(args)
	utils.RestoreOut(oldOut)
	if resp.Message != fmt.Sprintf(response.ErrorMessages["wrong_port"], hostname) {
		t.Fail()
	}

	args = []string{"master", "start", "-d", "--join", hostname, "-p", "0"}
	oldOut = utils.ShutUpOut()
	resp = MasterStart(args)
	utils.RestoreOut(oldOut)
	if resp.Message != fmt.Sprintf(response.ErrorMessages["master_failed"], "port 0 is not valid") {
		t.Fail()
	}

	args = []string{"master", "start", "-d", "--join", hostname, "-p", "1208"}
	oldOut = utils.ShutUpOut()
	resp = MasterStart(args)
	utils.RestoreOut(oldOut)
	if resp.Code != 0 {
		t.Fail()
	}

	time.Sleep(2000 * time.Millisecond)
	// Stop running masters
	oldOut = utils.ShutUpOut()
	args = []string{"master", "stop"}
	resp = MasterStop(args)
	utils.RestoreOut(oldOut)
	if resp.Code != 0 {
		t.Fail()
	}
}

func TestMasterStartArgs(t *testing.T) {
	argv := []string{"master", "start", "-d"}
	arguments := MasterStartArgs(argv)
	args := docopt.Opts{
		"--help": false,
		"--port": "1207",
		"--join": nil,
		"--detach": true,
		"start":  true,
		"master": true,
	}
	eq := reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}

	argv = []string{"master", "start", "-p", "1208"}
	arguments = MasterStartArgs(argv)
	args = docopt.Opts{
		"--help": false,
		"--port": "1208",
		"--join": nil,
		"--detach": false,
		"start":  true,
		"master": true,
	}
	eq = reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}

	argv = []string{"master", "start", "--join", "localhost"}
	arguments = MasterStartArgs(argv)
	args = docopt.Opts{
		"--help": false,
		"--port": "1207",
		"--join": "localhost",
		"--detach": false,
		"start":  true,
		"master": true,
	}
	eq = reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}

	argv = []string{"master", "start", "--join", "localhost", "-p", "1208"}
	arguments = MasterStartArgs(argv)
	args = docopt.Opts{
		"--help": false,
		"--port": "1208",
		"--join": "localhost",
		"--detach": false,
		"start":  true,
		"master": true,
	}
	eq = reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}
}
