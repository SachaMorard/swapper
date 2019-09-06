package utils

import (
	"os"
	"os/exec"
	"reflect"
	"strings"
)

func GetHostname() (hostname string, err error) {
	hostname, err = os.Hostname()
	return
}

func Command(command string) (string, error) {
	args := strings.Split(strings.TrimSpace(command), " ")
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.Output()
	return string(out), err
}

func FileExists(filePath string) bool {
	if _, err := os.Stat(filePath); err == nil {
		return true
	}
	return false
}

func ShutUpOut() *os.File {
	oldOut := os.Stdout // keep backup of the real stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	return oldOut
}

func RestoreOut(oldOut *os.File)  {
	os.Stdout = oldOut // restoring the real stdout
}

func InterfaceToArray(arguments interface{}) (arr []string) {
	switch reflect.TypeOf(arguments).Kind() {
	case reflect.Slice:

		s := reflect.ValueOf(arguments)

		for i := 0; i < s.Len(); i++ {
			arr = append(arr, (s.Index(i)).String())
		}

	}
	return arr
}
