package main

import (
	"fmt"
	"github.com/docopt/docopt-go"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestDeploy(t *testing.T) {
	_ = os.Mkdir(yamlDirectory, 0777)
	_ = os.Mkdir(pidDirectory, 0777)
	fmt.Println("remove "+yamlDirectory+"/swapper-1207.yml")

	// Stop running master (if exists)
	oldOut := ShutUpOut()
	args := []string{"master", "stop"}
	MasterStop(args)
	RestoreOut(oldOut)

	args = []string{"deploy", "-f", "doc/swapper.example.yml", "--var", "TAG=1.0.2", "--var", "ENV=prod"}
	response := Deploy(args)
	fmt.Println(response.Message)
	hostname, _ := GetHostname()
	if response.Message != fmt.Sprintf(errorMessages["bad_master_addr"], hostname+":1207") {
		t.Fail()
	}

	// Now start master
	args = []string{"master", "start", "-d"}
	oldOut = ShutUpOut()
	response = MasterStart(args)
	RestoreOut(oldOut)
	fmt.Println(response.Message)
	if response.Code != 0 {
		t.Fail()
	}
}

func TestDeploy2(t *testing.T) {
	args := []string{"deploy"}
	response := Deploy(args)
	if response.Message != fmt.Sprintf(errorMessages["file_not_exist"], "swapper.yml") {
		t.Fail()
	}
}
func TestDeploy3(t *testing.T) {
	args := []string{"deploy", "-f", "doc/swapper.example.yml", "--var", "TAG=1.0.1"}
	response := Deploy(args)
	if response.Message != fmt.Sprintf(errorMessages["var_missing"], "ENV", "--var ENV=<value>") {
		t.Fail()
	}
}

func TestDeploy4(t *testing.T) {
	args := []string{"deploy", "-f", "doc/tests/v1/swapper.invalid.7.yml"}
	response := Deploy(args)
	if response.Message != errorMessages["no_time_field"] {
		t.Fail()
	}

	args = []string{"deploy", "-f", "doc/tests/v1/swapper.invalid.8.yml"}
	response = Deploy(args)
	if response.Message != errorMessages["no_hash_field"] {
		t.Fail()
	}

	args = []string{"deploy", "-f", "doc/tests/v1/swapper.invalid.9.yml"}
	response = Deploy(args)
	if response.Message != errorMessages["no_masters_field"] {
		t.Fail()
	}
}

func TestDeploy5(t *testing.T) {
	time.Sleep(2000 * time.Millisecond)
	args := []string{"deploy", "-f", "doc/swapper.example.yml", "--var", "TAG=1.0.2", "--var", "ENV=prod"}
	oldOut := ShutUpOut()
	response := Deploy(args)
	RestoreOut(oldOut)
	if response.Code != 0 {
		t.Fail()
	}
}

func TestDeploy6(t *testing.T) {
	args := []string{"deploy", "-f", "doc/swapper.example.yml", "--var", "TAG=1.0.2", "--var", "ENV=prod", "--master", "localhost"}
	oldOut := ShutUpOut()
	response := Deploy(args)
	RestoreOut(oldOut)
	if response.Code != 0 {
		t.Fail()
	}
}

func TestDeployArgs(t *testing.T) {
	argv := []string{"deploy"}
	arguments := DeployArgs(argv)
	hostname, _ := GetHostname()
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
