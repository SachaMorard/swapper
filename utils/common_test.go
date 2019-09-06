package utils

import (
	"fmt"
	"reflect"
	"testing"
)

func TestGetHostname(t *testing.T) {
	_, err := GetHostname()
	if err != nil {
		t.Fail()
	}
}

func TestCommand(t *testing.T) {
	out, _ := Command("echo hello")
	if out != "hello\n" {
		t.Fail()
	}

	_, err := Command("ecsho hello")
	if err == nil {
		t.Fail()
	}
}

func TestFileExists(t *testing.T) {
	exists := FileExists("../doc/example.yml")
	if exists == false {
		t.Fail()
	}

	exists = FileExists("../doc/ssssexample.yml")
	if exists == true {
		t.Fail()
	}
}

func TestShutUpOut(t *testing.T) {
	oldOut := ShutUpOut()
	fmt.Println("nothing")
	RestoreOut(oldOut)
}

func TestInterfaceToArray(t *testing.T) {
	var args interface{}
	args = "[TAG=ok ENV=prod]"
	envs := InterfaceToArray(args)
	var vs = []string{}
	if reflect.TypeOf(envs) != reflect.TypeOf(vs) {
		t.Fail()
	}
}
