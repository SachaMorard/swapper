package main

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestParseSwapperYaml(t *testing.T) {
	input, _ := ioutil.ReadFile("tests/v1/swapper.invalid.yml")
	_, err := ParseSwapperYaml(string(input))
	if err.Error() != errorMessages["yaml_invalid"] {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/swapper.invalid.0.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != errorMessages["yaml_version"] {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/swapper.invalid.1.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != fmt.Sprintf(errorMessages["ports_invalid"], "80dq:80") {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/swapper.invalid.2.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != fmt.Sprintf(errorMessages["port_conflict"], "80") {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/swapper.invalid.3.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != errorMessages["ports_empty"] {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/swapper.invalid.4.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != fmt.Sprintf(errorMessages["service_field_needed"], "ports", "nginx") {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/swapper.invalid.5.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != fmt.Sprintf(errorMessages["service_field_needed"], "image", "nginx") {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/swapper.invalid.6.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != fmt.Sprintf(errorMessages["service_field_needed"], "tag", "nginx") {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/swapper.valid.1.yml")
	_, err = ParseSwapperYaml(string(input))
	if err != nil {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/swapper.valid.2.yml")
	_, err = ParseSwapperYaml(string(input))
	if err != nil {
		t.Fail()
	}
}

//func TestInterpretV1(t *testing.T) {
//	input, _ := ioutil.ReadFile("swapper.yml")
//	_, _ = ParseSwapperYaml(string(input))
//}
