package main

import (
	"github.com/sachamorard/swapper/commands"
	"testing"
)

func TestHelp(t *testing.T) {
	response := Help()
	if response.Message != usage {
		t.Fail()
	}
	if response.Code != 0 {
		t.Fail()
	}
}

func TestHelpMaster(t *testing.T) {
	response := HelpMaster()
	if response.Message != commands.MasterUsage {
		t.Fail()
	}
	if response.Code != 0 {
		t.Fail()
	}
}
func TestHelpNode(t *testing.T) {
	response := HelpNode()
	if response.Message != commands.NodeUsage {
		t.Fail()
	}
	if response.Code != 0 {
		t.Fail()
	}
}
