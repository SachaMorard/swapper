package commands

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
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
	baseYaml = `
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
	PidDirectory = "/tmp/swapper-pid"
	YamlDirectory = "/tmp/swapper-yaml"
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

func getYamlConfFromMasters(filename string, masters []string) (yamlConf yaml.YamlConf, err error) {

	// shuffle masters array
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(masters), func(i, j int) { masters[i], masters[j] = masters[j], masters[i] })

	// Get yaml file from master(s)
	for _, master := range masters {
		swapperYaml := GetYaml(filename, master)
		if swapperYaml != "" {
			yamlConf, err := yaml.ParseSwapperYaml(swapperYaml)
			return yamlConf, err
		}
	}
	return yamlConf, errors.New(response.ErrorMessages["cannot_contact_master"])
}

func GetYaml(filename string, hostname string) string {

	if strings.Contains(hostname, "gs://") {
		return GetYamlFromGCS(filename, hostname)
	}

	resp, err := http.Get("http://"+hostname+"/"+filename)
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

func GetYamlFromGCS(filename string, hostname string) string {

	ctx := context.Background()

	// Creates a client.
	client, errClient := storage.NewClient(ctx)
	if errClient != nil {
		return ""
	}

	// Set name of the new bucket.
	hostname = strings.Replace(hostname, "gs://", "", -1)
	bucketNames := strings.Split(hostname, "/")
	if len(bucketNames) != 1 {
		return ""
	}

	bucketName := bucketNames[0]
	rc, err := client.Bucket(bucketName).Object(filename).NewReader(ctx)
	if err != nil {
		return ""
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return ""
	}
	return string(data)
}

func GetMastersConf(hostname string) (val Conf) {
	resp, err := http.Get("http://"+hostname+"/")
	if err != nil {
		return val
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.Status != "200 OK" {
		return val
	} else {
		_ = json.Unmarshal(body, &val)
		return val
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

func WriteSwapperYaml(fileName string, swapperYaml string, currentPort string, masters []string, forceTime int64) error {
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

	sourceFile := YamlDirectory+"/"+fileName+"_"+currentPort
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
	files, err := ioutil.ReadDir(YamlDirectory)
	if err != nil {
		return false
	}

	var valid = regexp.MustCompile(`\.yml_` + currentPort + `$`)
	for _, f := range files {
		if valid.MatchString(f.Name()) {
			sourceFile := YamlDirectory+"/"+f.Name()
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
			if err != nil {
				return false
			}
		}
	}

	return true
}

func RefreshMaster(currentPort string) (err error) {
	defaultYaml, err := ioutil.ReadFile(YamlDirectory+"/default.yml_"+currentPort)
	if err != nil {
		return err
	}
	defaultYamlConf, err := yaml.ParseSwapperYaml(string(defaultYaml))
	if err != nil {
		return err
	}

	quorum := GetQuorum(defaultYamlConf.Masters, currentPort)
	for _, master := range quorum {
		conf := GetMastersConf(master)
		if len(conf.Yamls) != 0 {
			for _, filename := range conf.Yamls {
				distantYamlStr := GetYaml(filename, master)
				if distantYamlStr != "" {
					sourceFile := YamlDirectory+"/"+filename+"_"+currentPort
					swapperYaml, err := ioutil.ReadFile(sourceFile)
					if err != nil {
						err = ioutil.WriteFile(sourceFile, []byte(distantYamlStr), 0644)
						if err != nil {
							continue
						}
					}
					localYamlConf, err := yaml.ParseSwapperYaml(string(swapperYaml))
					if err != nil {
						return err
					}
					if distantYamlStr != "" {
						distantYamlConf, err := yaml.ParseSwapperYaml(distantYamlStr)
						if err == nil {
							if distantYamlConf.Time > localYamlConf.Time {
								_ = ioutil.WriteFile(sourceFile, []byte(distantYamlStr), 0644)
							}
						}
					}
				}
			}
			break
		}
	}
	return nil
}

func PingMasters(currentPort string) (quorum []string) {
	defaultYaml, err := ioutil.ReadFile(YamlDirectory+"/default.yml_"+currentPort)
	if err != nil {
		return quorum
	}
	defaultYamlConf, err := yaml.ParseSwapperYaml(string(defaultYaml))
	if err != nil {
		return quorum
	}

	quorum = GetQuorum(defaultYamlConf.Masters, currentPort)
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

func FirstPing(currentPort string) {
	time.Sleep(3000 * time.Millisecond)

	defaultYaml, err := ioutil.ReadFile(YamlDirectory+"/default.yml_"+currentPort)
	if err != nil {
		return
	}
	defaultYamlConf, err := yaml.ParseSwapperYaml(string(defaultYaml))
	if err != nil {
		return
	}

	if len(defaultYamlConf.Masters) <= 1 {
		return
	}

	for _, master := range defaultYamlConf.Masters {
		hostname, _ := utils.GetHostname()
		resp, err := http.Get("http://"+master+"/ping?mynameis="+hostname+":"+currentPort)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
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

type Conf struct {
	Yamls     []string      `json:"yamls,omitempty"`
	Masters   []string      `json:"masters,omitempty"`
}

func masterRequestHandler(ctx *fasthttp.RequestCtx) {

	if string(ctx.Method()) == "GET" {
		var valid = regexp.MustCompile(`\.yml$`)
		if valid.MatchString(string(ctx.Path())) {
			sourceFile := YamlDirectory+string(ctx.Path())+"_"+masterPort
			yaml, ioErr := ioutil.ReadFile(sourceFile)
			if ioErr != nil {
				ctx.Response.Reset()
				ctx.SetStatusCode(404)
				return
			}
			ctx.SetContentType("text/plain; charset=utf8")
			fmt.Fprintf(ctx, string(yaml))
			return
		}

		if string(ctx.Path()) == "/" {
			files, err := ioutil.ReadDir(YamlDirectory)
			if err != nil {
				ctx.Response.Reset()
				ctx.SetStatusCode(404)
				return
			}
			ctx.SetContentType("application/json; charset=utf8")

			conf := Conf{}
			var valid = regexp.MustCompile(`\.yml_`+masterPort+`$`)
			for _, f := range files {
				if valid.MatchString(string(f.Name())) {
					filename := strings.Replace(f.Name(), "_"+masterPort, "", -1)
					conf.Yamls = append(conf.Yamls, filename)
				}
			}

			defaultYaml, ioErr := ioutil.ReadFile(YamlDirectory+"/default.yml_"+masterPort)
			if ioErr != nil {
				ctx.Response.Reset()
				ctx.SetStatusCode(404)
				return
			}
			defaultYamlConf, err := yaml.ParseSwapperYaml(string(defaultYaml))
			if err != nil {
				ctx.Response.Reset()
				ctx.SetStatusCode(404)
				return
			}
			conf.Masters = defaultYamlConf.Masters
			serialized, err := json.Marshal(conf)
			fmt.Fprintf(ctx,"%s\n", serialized)
			return
		}

		if string(ctx.Path()) == "/ping" {
			mynameis := ctx.QueryArgs().Peek("mynameis")
			hostname := string(mynameis)
			AddMaster([]string{hostname}, masterPort)
			ctx.SetContentType("text/plain; charset=utf8")
			fmt.Fprintf(ctx, "Pong\n\n")
			return
		}
	}

	if string(ctx.Method()) == "POST" {
		ctx.SetContentType("text/plain; charset=utf8")
		var valid = regexp.MustCompile(`\.yml$`)
		if valid.MatchString(string(ctx.Path())) {
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
			WriteSwapperYaml(string(ctx.Path()), swapperYamlStr, masterPort, []string{}, 0)
			fmt.Fprintf(ctx, "Successful deployment\n")
			return
		}
	}

	ctx.Response.Reset()
	ctx.SetStatusCode(404)
	return
}
