package commands

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/sachamorard/swapper/response"
	"github.com/sachamorard/swapper/utils"
	"github.com/sachamorard/swapper/yaml"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	haproxyBaseConf = `
global
    log 127.0.0.1 local5 debug

defaults
    log     global
    option  dontlognull
    timeout connect 5000
    timeout client  50000
    timeout server  50000
`
	currentHash = ""
	firstYaml = `
version: "1"

services:
  hello:
    ports:
      - 80:80
    containers:
      - image: nginx
        tag: latest`
)

const (
	PidDirectory = "/tmp/swapper"
	YamlDirectory = "/tmp/swapper"
)

func CreateHaproxyConf(yamlConf yaml.YamlConf) (conf string, err error) {

	var haproxyConf []string
	// create frontend haproxy conf
	haproxyConf = append(haproxyConf, haproxyBaseConf)
	for _, frontend := range yamlConf.Frontends  {
		haproxyConf = append(haproxyConf, "frontend "+frontend.Name)
		haproxyConf = append(haproxyConf, "    option forwardfor")
		haproxyConf = append(haproxyConf, "    mode tcp")
		haproxyConf = append(haproxyConf, "    option tcplog")
		haproxyConf = append(haproxyConf, "    maxconn 800")
		haproxyConf = append(haproxyConf, "    bind 0.0.0.0:"+strconv.Itoa(frontend.Listen))
		haproxyConf = append(haproxyConf, "    default_backend "+frontend.BackendName)
		haproxyConf = append(haproxyConf, "")
	}


	for _, frontend := range yamlConf.Frontends {
		haproxyConf = append(haproxyConf, "backend "+frontend.BackendName)
		haproxyConf = append(haproxyConf, "    balance roundrobin")

		for _, container := range frontend.Containers {
			containerName := "swapper-container." + yamlConf.Hash + "." + frontend.ServiceName + "." + strconv.Itoa(container.Index)
			Id, err := utils.Command("docker ps --format {{.ID}} --filter name=" + containerName)
			if err != nil || Id == "" {
				return conf, errors.New(fmt.Sprintf(response.ErrorMessages["container_failed"], containerName))
			}

			ipCommand := "docker inspect -f {{.NetworkSettings.IPAddress}} " + containerName
			outIp, err := utils.Command(ipCommand)
			if err != nil {
				return conf, errors.New(fmt.Sprintf(response.ErrorMessages["container_ip_failed"], containerName))
			}
			ip := strings.TrimSpace(outIp)

			haproxyConf = append(haproxyConf, "    server container_"+strconv.Itoa(container.Index)+" "+ip+":"+strconv.Itoa(frontend.Bind)+" check observe layer4 weight "+strconv.Itoa(container.Weight))
		}
	}

	if len(haproxyConf) == 0 {
		return conf, errors.New(response.ErrorMessages["haproxy_conf_empty"])
	}

	return strings.Join(haproxyConf, "\n"), err
}

func getYamlConfFromMasters(masters []string) (yamlConf yaml.YamlConf, err error) {

	// shuffle masters array
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(masters), func(i, j int) { masters[i], masters[j] = masters[j], masters[i] })

	// Get swapper.yml from master(s)
	for _, master := range masters {
		swapperYaml := CurlSwapperYaml(master)
		if swapperYaml != "" {
			yamlConf, err := yaml.ParseSwapperYaml(swapperYaml)
			return yamlConf, err
		}
	}
	return yamlConf, errors.New(response.ErrorMessages["cannot_contact_master"])
}

func CurlSwapperYaml(hostname string) string {
	resp, err := http.Get("http://"+hostname+"/swapper.yml")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.Status != "200 OK" {
		return ""
	} else {
		return string(body)
	}
}

func unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func WriteSwapperYaml(swapperYaml string, currentPort string, masters []string, forceTime int64) error {
	newYamlConf, err := yaml.ParseSwapperYaml(swapperYaml)
	if err != nil {
		return err
	}

	hasher := md5.New()
	hasher.Write([]byte(swapperYaml))
	newYamlConf.Hash = hex.EncodeToString(hasher.Sum(nil))
	if forceTime == int64(0) {
		newYamlConf.Time = time.Now().UnixNano()
	} else {
		newYamlConf.Time = forceTime
	}

	sourceFile := YamlDirectory+"/swapper-"+currentPort+".yml"
	if _, err := os.Stat(sourceFile); err == nil {
		oldYaml, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			return err
		}
		oldYamlConf, err := yaml.ParseSwapperYaml(string(oldYaml))
		if err != nil {
			return err
		}
		masters = append(oldYamlConf.Masters, newYamlConf.Masters...)
	}
	hostname, _ := utils.GetHostname()
	addr := hostname+":"+currentPort
	masters = append(masters, addr)

	allMasters := unique(masters)
	sort.Strings(allMasters)
	newYamlConf.Masters = allMasters

	swapperYaml = swapperYaml + "\nhash: "+newYamlConf.Hash
	swapperYaml = swapperYaml + "\ntime: "+strconv.FormatInt(newYamlConf.Time, 10)
	swapperYaml = swapperYaml + "\nmasters: "
	for _, master := range newYamlConf.Masters {
		swapperYaml = swapperYaml + "\n  - "+master
	}

	err = ioutil.WriteFile(sourceFile, []byte(swapperYaml), 0644)
	if err != nil {
		return err
	}

	return nil
}

func AddMaster(masters []string, currentPort string) bool {
	sourceFile := YamlDirectory+"/swapper-"+currentPort+".yml"
	if _, err := os.Stat(sourceFile); err == nil {
		swapperYaml, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			return false
		}
		localYamlConf, err := yaml.ParseSwapperYaml(string(swapperYaml))
		if err != nil {
			return false
		}

		masters = append(masters, localYamlConf.Masters...)
		allMasters := unique(masters)
		sort.Strings(allMasters)
		swapperYamlStr := string(swapperYaml)
		swapperYamlSplit := strings.Split(swapperYamlStr, "\nmasters:")
		swapperYamlFinal := swapperYamlSplit[0] + "\nmasters: "
		for _, master := range allMasters {
			swapperYamlFinal = swapperYamlFinal + "\n  - "+master
		}
		err = ioutil.WriteFile(sourceFile, []byte(swapperYamlFinal), 0644)
		if err == nil {
			return true
		}
	}
	return false
}

func RefreshMaster(currentPort string) (quorum []string) {
	sourceFile := YamlDirectory+"/swapper-"+currentPort+".yml"
	if _, err := os.Stat(sourceFile); err == nil {
		swapperYaml, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			return quorum
		}
		localYamlConf, err := yaml.ParseSwapperYaml(string(swapperYaml))
		if err != nil {
			return quorum
		}

		quorum := GetQuorum(localYamlConf.Masters, currentPort)
		for _, master := range quorum {
			distantYamlStr := CurlSwapperYaml(master)
			if distantYamlStr != "" {
				distantYamlConf, err := yaml.ParseSwapperYaml(distantYamlStr)
				if err == nil {
					if distantYamlConf.Time > localYamlConf.Time {
						err = ioutil.WriteFile(sourceFile, []byte(distantYamlStr), 0644)
						if err != nil {
							break
						}
					}
				}
			}
		}
		return quorum
	}
	return quorum
}

func PingMasters(currentPort string) (quorum []string) {
	sourceFile := YamlDirectory+"/swapper-"+currentPort+".yml"
	if _, err := os.Stat(sourceFile); err == nil {
		swapperYaml, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			return quorum
		}
		localYamlConf, err := yaml.ParseSwapperYaml(string(swapperYaml))
		if err != nil {
			return quorum
		}

		quorum := GetQuorum(localYamlConf.Masters, currentPort)
		for _, master := range quorum {
			hostname, _ := utils.GetHostname()
			resp, err := http.Get("http://"+master+"/ping?mynameis="+hostname+":"+currentPort)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			if resp.Status != "200 OK" {
				continue
			}
		}
		return quorum
	}
	return quorum
}

func FirstPing(currentPort string) {
	time.Sleep(3000 * time.Millisecond)
	sourceFile := YamlDirectory+"/swapper-"+currentPort+".yml"
	if _, err := os.Stat(sourceFile); err == nil {
		swapperYaml, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			return
		}
		localYamlConf, err := yaml.ParseSwapperYaml(string(swapperYaml))
		if err != nil {
			return
		}

		if len(localYamlConf.Masters) <= 1 {
			return
		}

		for _, master := range localYamlConf.Masters {
			hostname, _ := utils.GetHostname()
			resp, _ := http.Get("http://"+master+"/ping?mynameis="+hostname+":"+currentPort)
			if err != nil {
				continue
			}
			defer resp.Body.Close()
		}
	}
	return
}

func GetQuorum(masters []string, currentPort string) (quorum []string) {
	countMaster := len(masters)
	if countMaster == 1 {
		return
	}

	var quorumNb int
	if countMaster%2 == 0 {
		quorumNb = (countMaster/2) + 1
	} else {
		floatCountMaster := (float64(countMaster) / 2) + 0.5
		quorumNb = int(floatCountMaster)
	}

	// shuffle masters
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(masters), func(i, j int) { masters[i], masters[j] = masters[j], masters[i] })

	hostname, _ := utils.GetHostname()
	for _, master := range masters {
		if master != hostname+":"+currentPort {
			if len(quorum) < quorumNb {
				quorum = append(quorum, master)
			}
		}
	}
	return quorum
}

func masterRequestHandler(ctx *fasthttp.RequestCtx) {

	ctx.SetContentType("text/plain; charset=utf8")

	if string(ctx.Method()) == "GET" {
		switch string(ctx.Path()) {
		case "/swapper.yml":
			sourceFile := YamlDirectory+"/swapper-"+masterPort+".yml"
			yaml, ioErr := ioutil.ReadFile(sourceFile)
			if ioErr != nil {
				ctx.Response.Reset()
				ctx.SetStatusCode(500)
				return
			}
			fmt.Fprintf(ctx, string(yaml))
			return
		case "/ping":
			mynameis := ctx.QueryArgs().Peek("mynameis")
			hostname := string(mynameis)
			AddMaster([]string{hostname}, masterPort)
			fmt.Fprintf(ctx, "Pong\n\n")
			return
		}
	}

	if string(ctx.Method()) == "POST" {
		switch string(ctx.Path()) {
		case "/swapper.yml":
			swapperYamlStr := string(ctx.PostBody())
			if strings.Contains(swapperYamlStr, "hash: ") {
				ctx.Response.Reset()
				ctx.Response.SetBody([]byte("Your yaml is invalid. You cannot use 'hash' field"))
				ctx.SetStatusCode(403)
				return
			}
			if strings.Contains(swapperYamlStr, "masters: ") {
				ctx.Response.Reset()
				ctx.Response.SetBody([]byte("Your yaml is invalid. You cannot use 'masters' field"))
				ctx.SetStatusCode(403)
				return
			}
			if strings.Contains(swapperYamlStr, "time: ") {
				ctx.Response.Reset()
				ctx.Response.SetBody([]byte("Your yaml is invalid. You cannot use 'time' field"))
				ctx.SetStatusCode(403)
				return
			}
			WriteSwapperYaml(swapperYamlStr, masterPort, []string{}, 0)
			fmt.Fprintf(ctx, "Successful deployment\n")
			return
		}
	}

	ctx.Response.Reset()
	ctx.SetStatusCode(404)
	return
}
