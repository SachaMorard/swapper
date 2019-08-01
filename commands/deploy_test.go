package commands

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/sachamorard/swapper/response"
	"github.com/sachamorard/swapper/utils"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestDeploy(t *testing.T) {
	_ = os.Mkdir(YamlDirectory, 0777)
	_ = os.Mkdir(PidDirectory, 0777)
	fmt.Println("remove "+YamlDirectory+"/swapper-1207.yml")

	// Stop running master (if exists)
	oldOut := utils.ShutUpOut()
	args := []string{"master", "stop"}
	MasterStop(args)
	utils.RestoreOut(oldOut)

	args = []string{"deploy", "-f", "../doc/swapper.example.yml", "--var", "TAG=1.0.2", "--var", "ENV=prod"}
	resp := Deploy(args)
	fmt.Println(resp.Message)
	hostname, _ := utils.GetHostname()
	if resp.Message != fmt.Sprintf(response.ErrorMessages["bad_master_addr"], hostname+":1207") {
		t.Fail()
	}

	// Now start master
	args = []string{"master", "start", "-d"}
	oldOut = utils.ShutUpOut()
	resp = MasterStart(args)
	utils.RestoreOut(oldOut)
	fmt.Println(resp.Message)
	if resp.Code != 0 {
		t.Fail()
	}
}

func TestDeploy2(t *testing.T) {
	args := []string{"deploy"}
	resp := Deploy(args)
	if resp.Message != fmt.Sprintf(response.ErrorMessages["file_not_exist"], "swapper.yml") {
		t.Fail()
	}
}
func TestDeploy3(t *testing.T) {
	args := []string{"deploy", "-f", "../doc/swapper.example.yml", "--var", "TAG=1.0.1"}
	resp := Deploy(args)
	if resp.Message != fmt.Sprintf(response.ErrorMessages["var_missing"], "ENV", "--var ENV=<value>") {
		t.Fail()
	}
}

func TestDeploy4(t *testing.T) {
	args := []string{"deploy", "-f", "../yaml/tests/v1/swapper.invalid.7.yml"}
	resp := Deploy(args)
	if resp.Message != response.ErrorMessages["no_time_field"] {
		t.Fail()
	}

	args = []string{"deploy", "-f", "../yaml/tests/v1/swapper.invalid.8.yml"}
	resp = Deploy(args)
	if resp.Message != response.ErrorMessages["no_hash_field"] {
		t.Fail()
	}

	args = []string{"deploy", "-f", "../yaml/tests/v1/swapper.invalid.9.yml"}
	resp = Deploy(args)
	if resp.Message != response.ErrorMessages["no_masters_field"] {
		t.Fail()
	}
}

func TestDeploy5(t *testing.T) {
	time.Sleep(2000 * time.Millisecond)
	args := []string{"deploy", "-f", "../doc/swapper.example.yml", "--var", "TAG=1.0.2", "--var", "ENV=prod"}
	oldOut := utils.ShutUpOut()
	resp := Deploy(args)
	utils.RestoreOut(oldOut)
	if resp.Code != 0 {
		t.Fail()
	}
}

func TestDeploy6(t *testing.T) {
	args := []string{"deploy", "-f", "../doc/swapper.example.yml", "--var", "TAG=1.0.2", "--var", "ENV=prod", "--master", "localhost"}
	oldOut := utils.ShutUpOut()
	resp := Deploy(args)
	utils.RestoreOut(oldOut)
	if resp.Code != 0 {
		t.Fail()
	}
}

func TestDeployArgs(t *testing.T) {
	argv := []string{"deploy"}
	arguments := DeployArgs(argv)
	hostname, _ := utils.GetHostname()
	args := docopt.Opts{
		"--var":  []string{},
		"--file": "swapper.yml",
		"--help": false,
		"--master": hostname,
		"deploy": true,
	}
	eq := reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}

	argv = []string{"deploy", "-f", "ok.yml"}
	arguments = DeployArgs(argv)
	args = docopt.Opts{
		"--var":  []string{},
		"--file": "ok.yml",
		"--help": false,
		"--master": hostname,
		"deploy": true,
	}
	eq = reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}

	argv = []string{"deploy", "--file", "ok.yml"}
	arguments = DeployArgs(argv)
	args = docopt.Opts{
		"--var":  []string{},
		"--file": "ok.yml",
		"--help": false,
		"--master": hostname,
		"deploy": true,
	}
	eq = reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}

	argv = []string{"deploy", "-f", "ok.yml", "--var", "ENV=prod", "--var", "KEY=mykey"}
	arguments = DeployArgs(argv)
	args = docopt.Opts{
		"--var":  []string{"ENV=prod", "KEY=mykey"},
		"--file": "ok.yml",
		"--help": false,
		"--master": hostname,
		"deploy": true,
	}
	eq = reflect.DeepEqual(arguments, args)
	if !eq {
		t.Fail()
	}
}
