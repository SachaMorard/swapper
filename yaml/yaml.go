package yaml

import (
	"errors"
	"fmt"
	"github.com/sachamorard/swapper/response"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
)

type Yaml struct {
	data interface{}
}

type Container struct {
	Index int
	Name string
	Image string
	Tag string
	Weight int
	Envs map[interface{}]interface{}
	LoggingOptions map[interface{}]interface{}
	LoggingDriver string
	HealthCmd string
	HealthInterval string
	HealthRetries int
	HealthTimeout string
	ExtraHosts []interface{}
}

type Service struct {
	Name string
	Ports []string
	Containers []Container
}

type Frontend struct {
	Name string
	Listen int
	Bind int
	BackendName string
	ServiceName string
	Containers []Container
}

type Slack struct {
	WebHookUrl string
	Channel string
}

type YamlConf struct {
	Frontends []Frontend
	Services []Service
	Hash string
	Time int64
	Masters []string
	Slack Slack
	Master Master
}

type Master struct {
	Driver string
	ProjectId string
	CredentialsFile string
}

func PrepareSwapperYaml(sourceFile string, vars []string) (cleanYaml string, err error) {

	input, ioErr := ioutil.ReadFile(sourceFile)
	if ioErr != nil {
		return cleanYaml, errors.New(fmt.Sprintf(response.ErrorMessages["file_not_exist"], sourceFile))
	}

	// Unmarshal Yaml to remove comments and clean it
	var val interface{}
	errUnmarshal := yaml.Unmarshal([]byte(string(input)), &val)
	if errUnmarshal != nil {
		return cleanYaml, errors.New(response.ErrorMessages["yaml_invalid"])
	}

	d, errMarshal := yaml.Marshal(&val)
	if errMarshal != nil {
		return cleanYaml, errors.New(response.ErrorMessages["yaml_invalid"])
	}
	cleanYaml = string(d)

	// Transform vars strings to map
	varMap := make(map[string]string)
	var keyValue []string
	for _, e := range vars {
		if e != "" {
			keyValue = strings.Split(e, "=")
			varMap[keyValue[0]] = keyValue[1]
		}
	}

	// Replace variables if exist
	re := regexp.MustCompile(`\${[a-zA-Z0-9_-]+}`)
	matches := re.FindAllString(cleanYaml, -1)
	var varname string
	for _, p := range matches {
		varname = strings.Replace(strings.Replace(p, "${", "", 1), "}", "", -1)
		if varMap[varname] != "" {
			cleanYaml = strings.Replace(cleanYaml, p, varMap[varname], -1)
		}
	}

	// Check whether variables still need to be defined
	matches = re.FindAllString(cleanYaml, -1)
	if len(matches) > 0 {
		varnames := ""
		examples := ""
		alreadyDone := map[string]bool{}
		for _, p := range matches {

			varname = strings.Replace(strings.Replace(p, "${", "", 1), "}", "", -1)
			if alreadyDone[varname] != true {
				varnames = varnames + varname + " "
				examples = examples + "--var " + varname + "=<value> "
				alreadyDone[varname] = true
			}
		}
		return cleanYaml, errors.New(fmt.Sprintf(response.ErrorMessages["var_missing"], strings.TrimSpace(varnames), strings.TrimSpace(examples)))
	}

	return cleanYaml, err
}

func ParseSwapperYaml(yamlStr string) (yamlConf YamlConf, err error) {
	var val interface{}
	err = yaml.Unmarshal([]byte(yamlStr), &val)
	swapperYaml := &Yaml{val}
	if err != nil {
		return yamlConf, errors.New(response.ErrorMessages["yaml_invalid"])
	}

	yamlVersion, _ := swapperYaml.Get("version").String()
	if yamlVersion == "1" {
		yamlConf, err = InterpretV1(swapperYaml)
		return yamlConf, err
	}
	return yamlConf, errors.New(response.ErrorMessages["yaml_version"])
}

func InterpretV1(swapperYaml *Yaml) (yamlConf YamlConf, err error) {
	var services []Service
	var frontends []Frontend

	hash, _ := swapperYaml.Get("hash").String()
	if hash != "" {
		yamlConf.Hash = hash
	}

	yamlConf.Time = 0
	time, _ := swapperYaml.Get("time").Int()
	if time != 0 {
		yamlConf.Time = int64(time)
	}

	mastersInterface, _ := swapperYaml.Get("masters").Array()
	var masters []string
	for _, master := range mastersInterface {
		masters = append(masters, master.(string))
	}
	yamlConf.Masters = masters

	var master Master
	master.Driver = "local"
	driver, errDriver := swapperYaml.GetPath("master", "driver").String()
	if errDriver == nil && driver != "" {
		if driver == "gcp" {
			master.Driver = "gcp"

			// mandatory fields
			master.ProjectId, _ = swapperYaml.GetPath("master", "project-id").String()
			if master.ProjectId == "" {
				return yamlConf, errors.New(fmt.Sprintf(response.ErrorMessages["master_field_needed"], "project-id"))
			}

			credentialsFile, _ := swapperYaml.GetPath("master", "credentials-file").String()
			if credentialsFile != "" {
				master.CredentialsFile = credentialsFile
			}
		}
	}
	yamlConf.Master = master

	slackHook, _ := swapperYaml.GetPath("slack", "webhook-url").String()
	slackChannel, _ := swapperYaml.GetPath("slack", "channel").String()
	var slack Slack
	if slackHook != "" && slackChannel != "" {
		slack.Channel = slackChannel
		slack.WebHookUrl = slackHook
		yamlConf.Slack = slack
	}

	// Services
	serviceNames, _ := swapperYaml.GetPath("services").GetMapKeys()
	checkFrontendPort := map[string]bool{}
	for _, serviceName := range serviceNames {

		// Service
		serviceYml := swapperYaml.GetPath("services", serviceName)
		var Service Service
		Service.Name = serviceName

		containerLen, yamlErr := swapperYaml.GetPath("services", serviceName, "containers").GetArraySize()
		if yamlErr == nil && containerLen > 0 {
			for i:=0; i<containerLen ;i++  {
				containerYml := swapperYaml.GetPath("services", serviceName, "containers").GetIndex(i)

				var Container Container
				Container.Name = serviceName
				Container.Index = i

				// image
				Container.Image, _ = containerYml.Get("image").String()
				if Container.Image == "" {
					return yamlConf, errors.New(fmt.Sprintf(response.ErrorMessages["service_field_needed"], "image", serviceName))
				}

				// tag
				Container.Tag, _ = containerYml.Get("tag").String()
				if Container.Tag == "" {
					return yamlConf, errors.New(fmt.Sprintf(response.ErrorMessages["service_field_needed"], "tag", serviceName))
				}

				// weight
				Container.Weight, _ = containerYml.Get("weight").Int()
				if Container.Weight == 0 {
					Container.Weight = 100
				}

				// more
				Container.Envs, _ = containerYml.Get("environment").Map()
				Container.LoggingOptions, _ = containerYml.Get("logging").Get("options").Map()
				Container.LoggingDriver, _ = containerYml.Get("logging").Get("driver").String()
				Container.HealthCmd, _ = containerYml.Get("health-cmd").String()
				Container.HealthInterval, _ = containerYml.Get("health-interval").String()
				Container.HealthTimeout, _ = containerYml.Get("health-timeout").String()
				Container.HealthRetries, _ = containerYml.Get("health-retries").Int()
				if Container.HealthRetries == 0 {
					healthRetriesStr, _ := containerYml.Get("health-retries").String()
					if healthRetriesStr != "" {
						Container.HealthRetries, _ = strconv.Atoi(healthRetriesStr)
					}
				}
				Container.ExtraHosts, _ = containerYml.Get("extra_hosts").Array()
				// todo more options

				Service.Containers = append(Service.Containers, Container)
			}
		}
		services = append(services, Service)

		// Binding
		var servicePorts []string
		portsLen, _ := serviceYml.Get("ports").GetArraySize()
		if portsLen > 0 {
			for o:=0; o<portsLen ;o++  {
				portStr, _ := serviceYml.Get("ports").GetIndex(o).String()
				if portStr != "" {
					splittedPort := strings.Split(portStr, ":")
					if checkFrontendPort[splittedPort[0]] != true {
						frontendPort, _ := strconv.Atoi(splittedPort[0])
						containerPort, _ := strconv.Atoi(splittedPort[1])
						if frontendPort == 0 || containerPort == 0 {
							return yamlConf, errors.New(fmt.Sprintf(response.ErrorMessages["ports_invalid"], portStr))
						}
						servicePorts = append(servicePorts, portStr)
						checkFrontendPort[splittedPort[0]] = true
						// set Frontend
						var Front Frontend
						Front.Listen = frontendPort
						Front.Bind = containerPort
						Front.Name = "frontend_"+splittedPort[0]
						Front.BackendName = "backend_"+splittedPort[0]+"_"+splittedPort[1]
						Front.ServiceName = serviceName
						Front.Containers = Service.Containers
						frontends = append(frontends, Front)
					} else {
						return yamlConf, errors.New(fmt.Sprintf(response.ErrorMessages["port_conflict"], splittedPort[0]))
					}

				} else {
					return yamlConf, errors.New(response.ErrorMessages["ports_empty"])
				}
			}
		} else {
			return yamlConf, errors.New(fmt.Sprintf(response.ErrorMessages["service_field_needed"], "ports", serviceName))
		}
		Service.Ports = servicePorts
	}

	yamlConf.Services = services
	yamlConf.Frontends = frontends
	return
}

// Get returns a pointer to a new `Yaml` object for `key` in its `map` representation
//
// Example:
//      y.Get("xx").Get("yy").Int()
func (y *Yaml) Get(key interface{}) *Yaml {
	m, err := y.Map()
	if err == nil {
		if val, ok := m[key]; ok {
			return &Yaml{val}
		}
	}
	return &Yaml{nil}
}

// GetPath searches for the item as specified by the branch
//
// Example:
//      y.GetPath("bb", "cc").Int()
func (y *Yaml) GetPath(branch ...interface{}) *Yaml {
	yin := y
	for _, p := range branch {
		yin = yin.Get(p)
	}
	return yin
}

// Array type asserts to an `array`
func (y *Yaml) Array() ([]interface{}, error) {
	if a, ok := (y.data).([]interface{}); ok {
		return a, nil
	}
	return nil, errors.New("type assertion to []interface{} failed")
}

func (y *Yaml) IsArray() bool {
	_, err := y.Array()

	return err == nil
}

// return the size of array
func (y *Yaml) GetArraySize() (int, error) {
	a, err := y.Array()
	if err != nil {
		return 0, err
	}
	return len(a), nil
}

// GetIndex returns a pointer to a new `Yaml` object.
// for `index` in its `array` representation
//
// Example:
//      y.Get("xx").GetIndex(1).String()
func (y *Yaml) GetIndex(index int) *Yaml {
	a, err := y.Array()
	if err == nil {
		if len(a) > index {
			return &Yaml{a[index]}
		}
	}
	return &Yaml{nil}
}

// Int type asserts to `int`
func (y *Yaml) Int() (int, error) {
	if v, ok := (y.data).(int); ok {
		return v, nil
	}
	return 0, errors.New("type assertion to int failed")
}

// Bool type asserts to `bool`
func (y *Yaml) Bool() (bool, error) {
	if v, ok := (y.data).(bool); ok {
		return v, nil
	}
	return false, errors.New("type assertion to bool failed")
}

// String type asserts to `string`
func (y *Yaml) String() (string, error) {
	if v, ok := (y.data).(string); ok {
		return v, nil
	}
	return "", errors.New("type assertion to string failed")
}

func (y *Yaml) Float() (float64, error) {
	if v, ok := (y.data).(float64); ok {
		return v, nil
	}
	return 0, errors.New("type assertion to float64 failed")
}

// Map type asserts to `map`
func (y *Yaml) Map() (map[interface{}]interface{}, error) {
	if m, ok := (y.data).(map[interface{}]interface{}); ok {
		return m, nil
	}
	return nil, errors.New("type assertion to map[interface]interface{} failed")
}

// Check if it is a map
func (y *Yaml) IsMap() bool {
	_, err := y.Map()
	return err == nil
}

// Get all the keys of the map
func (y *Yaml) GetMapKeys() ([]string, error) {
	m, err := y.Map()

	if err != nil {
		return nil, err
	}
	keys := make([]string, 0)
	for k, _ := range m {
		if s, ok := k.(string); ok {
			keys = append(keys, s)
		}
	}
	return keys, nil
}
