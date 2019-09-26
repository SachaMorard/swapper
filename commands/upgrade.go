package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/docopt/docopt-go"
	"github.com/dustin/go-humanize"
	"github.com/sachamorard/swapper/response"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var(
	upgradeUsage = `
swapper upgrade [OPTIONS].

Upgrade swapper version

Usage:
 swapper upgrade [--force]
 swapper upgrade (-h|--help)

Options:
 -h --help               Show this screen.
 -f --force              Force upgrade (without prompt)

Run 'swapper upgrade --help' for more information on a command.

`
)

const (
	repo = "SachaMorard/swapper"
)

type LastestRelease struct {
	Tag    string          `json:"tag_name"`
}

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer
// interface and we can pass this into io.TeeReader() which will report progress on each
// write cycle.
type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory. We pass an io.TeeReader
// into Copy() to report progress on the download.
func DownloadFile(filepath string, url string) error {

	// Create the file, but give it a tmp file extension, this means we won't overwrite a
	// file until it's downloaded, but we'll remove the tmp extension once downloaded.
	out, err := os.Create(filepath + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create our progress reporter and pass it to be used alongside our writer
	counter := &WriteCounter{}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Print("\n")

	err = os.Rename(filepath+".tmp", filepath)
	if err != nil {
		return err
	}

	return nil
}

func UpgradeArgs(argv []string) docopt.Opts {
	arguments, _ := docopt.ParseArgs(upgradeUsage, argv, "")
	return arguments
}

func Upgrade(argv []string) response.Response {
	arguments := UpgradeArgs(argv)

	resp, err := http.Get("https://api.github.com/repos/" + repo + "/releases/latest")
	if err != nil {
		return response.Fail(err.Error())
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var val LastestRelease
	if resp.Status != "200 OK" {
		return response.Fail("err")
	} else {
		_ = json.Unmarshal(body, &val)

		if version != val.Tag {

			cmdString := "Y"
			if arguments["--force"] == false {
				fmt.Println("Swapper is actually at version " + version)
				fmt.Println("[WARNING] Upgrading could be sensitive, you would have to read the changelog (https://github.com/SachaMorard/swapper/blob/master/CHANGELOG.md) to check side effects are expected.")
				fmt.Print("\nDo you really want to upgrade swapper to version " + val.Tag + "? [Y/n] ")
				reader := bufio.NewReader(os.Stdin)
				cmdString, err = reader.ReadString('\n')
				if err != nil {
					return response.Fail(err.Error())
				}
			}

			if strings.Trim(cmdString, "\n") == "Y" {
				cmdOs := exec.Command("uname", "-s")
				currentos, err := cmdOs.Output()
				if err != nil {
					return response.Fail(err.Error())
				}
				OS := strings.Trim(string(currentos), "\n")

				cmdArch := exec.Command("uname", "-m")
				currentarch, err := cmdArch.Output()
				if err != nil {
					return response.Fail(err.Error())
				}
				ARCH := strings.Trim(string(currentarch), "\n")


				ex, err := os.Executable()
				if err != nil {
					return response.Fail(err.Error())
				}

				fmt.Print("\n")
				err = DownloadFile(ex, "https://github.com/"+repo+"/releases/download/"+val.Tag+"/swapper-" + OS + "-" + ARCH)
				if err != nil {
					return response.Fail(err.Error())
				}

				err = os.Chmod(ex, 0755)
				if err != nil {
					return response.Fail(err.Error())
				}
			}
		} else {
			fmt.Println("Swapper is already up to date (version " + version + ")")
		}
		return response.Success("")
	}

}
