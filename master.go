package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

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
	newYamlConf, err := ParseSwapperYaml(swapperYaml)
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

	sourceFile := yamlDirectory+"/swapper-"+currentPort+".yml"
	if _, err := os.Stat(sourceFile); err == nil {
		oldYaml, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			return err
		}
		oldYamlConf, err := ParseSwapperYaml(string(oldYaml))
		if err != nil {
			return err
		}
		masters = append(oldYamlConf.Masters, newYamlConf.Masters...)
	}
	hostname, _ := GetHostname()
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
	sourceFile := yamlDirectory+"/swapper-"+currentPort+".yml"
	if _, err := os.Stat(sourceFile); err == nil {
		swapperYaml, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			return false
		}
		localYamlConf, err := ParseSwapperYaml(string(swapperYaml))
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
	sourceFile := yamlDirectory+"/swapper-"+currentPort+".yml"
	if _, err := os.Stat(sourceFile); err == nil {
		swapperYaml, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			return quorum
		}
		localYamlConf, err := ParseSwapperYaml(string(swapperYaml))
		if err != nil {
			return quorum
		}

		quorum := GetQuorum(localYamlConf.Masters, currentPort)
		for _, master := range quorum {
			distantYamlStr := CurlSwapperYaml(master)
			if distantYamlStr != "" {
				distantYamlConf, err := ParseSwapperYaml(distantYamlStr)
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
	sourceFile := yamlDirectory+"/swapper-"+currentPort+".yml"
	if _, err := os.Stat(sourceFile); err == nil {
		swapperYaml, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			return quorum
		}
		localYamlConf, err := ParseSwapperYaml(string(swapperYaml))
		if err != nil {
			return quorum
		}

		quorum := GetQuorum(localYamlConf.Masters, currentPort)
		for _, master := range quorum {
			hostname, _ := GetHostname()
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
	sourceFile := yamlDirectory+"/swapper-"+currentPort+".yml"
	if _, err := os.Stat(sourceFile); err == nil {
		swapperYaml, err := ioutil.ReadFile(sourceFile)
		if err != nil {
			return
		}
		localYamlConf, err := ParseSwapperYaml(string(swapperYaml))
		if err != nil {
			return
		}

		if len(localYamlConf.Masters) <= 1 {
			return
		}

		for _, master := range localYamlConf.Masters {
			hostname, _ := GetHostname()
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

	hostname, _ := GetHostname()
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
			sourceFile := yamlDirectory+"/swapper-"+masterPort+".yml"
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
