package main

import (
	"testing"
)

func TestVersion(t *testing.T) {
	response := Version()
	if response.Message != version {
		t.Fail()
	}
}
