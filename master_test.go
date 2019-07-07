package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestWriteSwapperYaml(t *testing.T) {
	err := WriteSwapperYaml("jklfd fdsf: fds", "1207", []string{"ok", "c", "a", "c"}, 0)
	if err.Error() != errorMessages["yaml_version"] {
		t.Fail()
	}

	err = WriteSwapperYaml(firstYaml, "1207", []string{"ok", "c", "a", "c"}, 0)
	if err != nil {
		t.Fail()
	}
}

func TestGetQuorum(t *testing.T) {
	quorum := GetQuorum([]string{"host1", "host2", "host3", "host4", "host5", "host6", "host7"}, "1207")
	if len(quorum) != 4 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1", "host2", "host3", "host4", "host5", "host6"}, "1207")
	if len(quorum) != 4 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1", "host2", "host3", "host4", "host5"}, "1207")
	if len(quorum) != 3 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1", "host2", "host3", "host4"}, "1207")
	if len(quorum) != 3 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1", "host2", "host3"}, "1207")
	if len(quorum) != 2 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1", "host2"}, "1207")
	if len(quorum) != 2 {
		t.Fail()
	}

	quorum = GetQuorum([]string{"host1"}, "1207")
	if len(quorum) != 0 {
		t.Fail()
	}
}

func TestRefreshMaster(t *testing.T) {

	port := "1111"
	sourceFile := yamlDirectory+"/swapper-"+port+".yml"
	swapperYaml := `
version: "1"

services:
  hello:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: latest
masters:
  - host1:1207
  - host2:1207
  - host3:1207
  - host4:1207
  - host5:1207
`
	_ = ioutil.WriteFile(sourceFile, []byte(swapperYaml), 0644)

	quorum := RefreshMaster(port)
	if len(quorum) != 3 {
		t.Fail()
	}

	_ = os.Remove(sourceFile)
}

func TestPingMasters(t *testing.T) {
	port := "1111"
	sourceFile := yamlDirectory+"/swapper-"+port+".yml"
	swapperYaml := `
version: "1"

services:
  hello:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: latest
masters:
  - host1:1207
  - host2:1207
  - host3:1207
  - host4:1207
  - host5:1207
`
	_ = ioutil.WriteFile(sourceFile, []byte(swapperYaml), 0644)

	quorum := PingMasters(port)
	if len(quorum) != 3 {
		t.Fail()
	}

	_ = os.Remove(sourceFile)
}

func TestAddMaster(t *testing.T) {
	port := "1111"
	sourceFile := yamlDirectory+"/swapper-"+port+".yml"
	swapperYaml := `
version: "1"

services:
  hello:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: latest
masters:
  - host1:1207
  - host2:1207
  - host3:1207
  - host4:1207
  - host5:1207
`

	swapperYamlExpected := `
version: "1"

services:
  hello:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: latest
masters: 
  - ahost:1
  - host1:1207
  - host2:1207
  - host3:1207
  - host4:1207
  - host5:1207
  - ok
  - ok2`

	_ = ioutil.WriteFile(sourceFile, []byte(swapperYaml), 0644)
	AddMaster([]string{"ok", "ok2", "host1:1207", "ahost:1"}, port)

	swapperYamlNew, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		t.Fail()
	}

	if string(swapperYamlNew) != swapperYamlExpected {
		t.Fail()
	}
	_ = os.Remove(sourceFile)
}
