package yaml

import (
	"fmt"
	"github.com/sachamorard/swapper/response"
	"io/ioutil"
	"testing"
)

func TestPrepareSwapperYaml(t *testing.T) {
	vars := []string{}
	_, err := PrepareSwapperYaml("tests/v1/swappers.valid.yml", vars)
	if err.Error() != fmt.Sprintf(response.ErrorMessages["file_not_exist"], "tests/v1/swappers.valid.yml") {
		t.Fail()
	}

	_, err = PrepareSwapperYaml("tests/v1/invalid.yml", vars)
	if err.Error() != response.ErrorMessages["yaml_invalid"] {
		t.Fail()
	}

	_, err = PrepareSwapperYaml("tests/v1/valid.yml", vars)
	if err.Error() != fmt.Sprintf(response.ErrorMessages["var_missing"], "ENV TAG", "--var ENV=<value> --var TAG=<value>") {
		t.Fail()
	}

	vars = []string{"TAG=1.0.1"}
	_, err = PrepareSwapperYaml("tests/v1/valid.yml", vars)
	if err.Error() != fmt.Sprintf(response.ErrorMessages["var_missing"], "ENV", "--var ENV=<value>") {
		t.Fail()
	}

	vars = []string{"TAG=1.0.1", "ENV=prod"}
	_, err = PrepareSwapperYaml("tests/v1/valid.yml", vars)
	if err != nil {
		t.Fail()
	}
}

func TestParseSwapperYaml(t *testing.T) {
	input, _ := ioutil.ReadFile("tests/v1/invalid.yml")
	_, err := ParseSwapperYaml(string(input))
	if err.Error() != response.ErrorMessages["yaml_invalid"] {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/invalid.0.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != response.ErrorMessages["yaml_version"] {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/invalid.1.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != fmt.Sprintf(response.ErrorMessages["ports_invalid"], "80dq:80") {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/invalid.2.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != fmt.Sprintf(response.ErrorMessages["port_conflict"], "80") {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/invalid.3.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != response.ErrorMessages["ports_empty"] {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/invalid.4.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != fmt.Sprintf(response.ErrorMessages["service_field_needed"], "ports", "nginx") {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/invalid.5.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != fmt.Sprintf(response.ErrorMessages["service_field_needed"], "image", "nginx") {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/invalid.6.yml")
	_, err = ParseSwapperYaml(string(input))
	if err.Error() != fmt.Sprintf(response.ErrorMessages["service_field_needed"], "tag", "nginx") {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/valid.1.yml")
	_, err = ParseSwapperYaml(string(input))
	if err != nil {
		t.Fail()
	}

	input, _ = ioutil.ReadFile("tests/v1/valid.2.yml")
	_, err = ParseSwapperYaml(string(input))
	if err != nil {
		t.Fail()
	}
}

func TestInterpretV1(t *testing.T) {
	input, _ := ioutil.ReadFile("swapper.yml")
	_, _ = ParseSwapperYaml(string(input))
}
