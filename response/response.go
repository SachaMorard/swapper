package response

import (
	"fmt"
	"os"
)

var (
	ErrorMessages = map[string]string{
		"master_already_started": `
[ERROR] A swapper master is already running on this machine! To add new master to the ring, execute:
  swapper master start --join <previous-master-hostname>
`,

		"master_failed": `
[ERROR] Swapper master failed to start: %s
`,

		"proxy_failed": `
[ERROR] Swapper proxy failed to start
`,

		"proxy_stop_failed": `
[ERROR] Swapper proxy failed to stop
`,

		"no_masters_field": `
[ERROR] Your yaml is invalid. you cannot use "masters" field
`,

		"no_hash_field": `
[ERROR] Your yaml is invalid. You cannot use "hash" field
`,

		"no_time_field": `
[ERROR] Your yaml is invalid. You cannot use "time" field
`,

		"need_master_addr": `
[ERROR] Swapper node can't start without joining anything. Try with:
  swapper node start --join <master-hostname>
`,

		"container_failed": `
[ERROR] Container %s failed to start
`,

		"container_ip_failed": `
[ERROR] Can't find container's IP for %s
`,

		"master_not_running": `
[ERROR] Swapper master is not running! Try to start with:
  swapper master start
`,

		"bad_master_addr": `
[ERROR] Swapper master is not running, or its hostname "%s" is not responding. 
Start master with:
  swapper master start
Or you can specify its address with following command:
  swapper master start --master master-hostname:1207
`,

		"cannot_contact_master": `
[ERROR] Swapper master is not responding!
`,

		"containers_not_running": `
[ERROR] Any swapper-container running! Try to start with:
  swapper node start --join <master-hostname>
`,

		"yaml_invalid": `
[ERROR] Your Yaml file is invalid
`,

		"haproxy_conf_empty": `
[ERROR] Swapper Proxy's conf is empty
`,

		"file_not_exist": `
[ERROR] File %s does not exist
`,

		"var_missing": `
[ERROR] Missing variable (%s), try same command ended by:
  %s
`,

		"wrong_port": `
[ERROR] A swapper master is already running on this machine with this port! You have to specify a new one:
  swapper master start --join %s -p <FREE PORT>
`,

		"request_failed": `
[ERROR] Request failed.
  %s
`,

		"deploy_failed": `
[ERROR] Deploy failed.
  %s
`,

		"yaml_version": `
[ERROR] Yaml Error, unknown version
`,

		"service_field_needed": `
[ERROR] '%s' for service '%s' is required or invalid
`,

		"port_conflict": `
[ERROR] You try to bind entry port %s multiple times
`,

		"ports_invalid": `
[ERROR] It seems binding "%s" is invalid
`,

		"ports_empty": `
[ERROR] Ports cannot be an empty string
`,

		"command_failed": `
[ERROR] A command inside your yaml failed:
%s
`,
	}
)


type Response struct {
	Code    int
	Message string
}

func Success(message string) Response {
	return Response{Code: 0, Message: message}
}

func Fail(message string) Response {
	return Response{Code: 1, Message: message}
}

func Send(response Response) {
	fmt.Println(response.Message)
	os.Exit(response.Code)
}
