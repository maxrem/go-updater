package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const downloadFolder = "download"

// Download is a struct to hold information about the downloaded file
type Download struct {
	URL          string
	TargetPath   string
	SectionCount int
}

// Section is a struct that holds information about a section of the download
type Section struct {
	ID    int
	Start int
	End   int
}

// GetTempFileName returns a formatted temporary filename
func (s *Section) GetTempFileName() string {
	return fmt.Sprintf("section-%v.tmp", s.ID)
}

// Do checks the size of downloaded file, initializes download of the sections and merges the downloaded sections
func (d Download) Do() (err error) {
	log.Println("setting up connection")
	req, err := d.getNewRequest("HEAD")
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	log.Println("HEAD request response code", resp.StatusCode)

	if resp.StatusCode > 299 {
		return fmt.Errorf("HEAD request response code is %v", resp.StatusCode)
	}
	sizeBytes, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return
	}
	log.Println("Content-Length is", sizeBytes, "bytes")

	sectionList := d.initSectionList(sizeBytes)
	log.Println(sectionList)

	var wg sync.WaitGroup
	for _, s := range sectionList {
		wg.Add(1)
		go func(s Section) {
			defer wg.Done()
			err = d.downloadSection(s)
			if err != nil {
				panic(err)
			}
		}(s)
	}
	wg.Wait()

	return d.mergeFiles(sectionList)
}

func (d Download) getNewRequest(method string) (req *http.Request, err error) {
	req, err = http.NewRequest(method, d.URL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "My Download Manager v"+version())

	return
}

func (d Download) initSectionList(sizeBytes int) (sectionList []Section) {
	sectionSize := sizeBytes / d.SectionCount
	sectionList = make([]Section, d.SectionCount)
	for i := range sectionList {
		section := Section{ID: i}
		if i == 0 {
			section.Start = 0
		} else {
			section.Start = sectionList[i-1].End + 1
		}

		if i < d.SectionCount-1 {
			section.End = section.Start + sectionSize
		} else {
			section.End = sizeBytes - 1
		}
		sectionList[i] = section
	}

	return
}

func (d Download) downloadSection(s Section) (err error) {
	req, err := d.getNewRequest("GET")
	if err != nil {
		return
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", s.Start, s.End))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	log.Println("downloaded", resp.Header.Get("Content-Length"), "bytes for section", s.ID)
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = ioutil.WriteFile(filepath.Join(downloadFolder, s.GetTempFileName()), b, os.ModePerm)
	if err != nil {
		return err
	}

	return
}

func (d Download) mergeFiles(sectionList []Section) (err error) {
	f, err := os.OpenFile(filepath.Join(downloadFolder, d.TargetPath), os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return
	}
	defer func() {
		err = f.Close()
		if err != nil {
			log.Println("error during closing of file", err)
		}
	}()

	for _, s := range sectionList {
		b, err := ioutil.ReadFile(filepath.Join(downloadFolder, s.GetTempFileName()))
		if err != nil {
			return err
		}
		n, err := f.Write(b)
		if err != nil {
			return err
		}
		err = os.Remove(filepath.Join(downloadFolder, s.GetTempFileName()))
		if err != nil {
			return err
		}
		log.Println(n, "bytes merged")
	}

	return
}

func main() {
	var err error

	goVersion := flag.String("version", "1.14.6", "Go version")
	skipDownload := flag.Bool("skip-download", false, "Skip download")
	flag.Parse()

	log.Println("Go version to be installed", *goVersion)
	log.Println("Skip download", *skipDownload)

	startTime := time.Now()

	filename := fmt.Sprintf("go%v.linux-amd64.tar.gz", *goVersion)

	if !*skipDownload {
		d := Download{
			URL:          fmt.Sprintf("https://golang.org/dl/" + filename),
			TargetPath:   filename,
			SectionCount: 10,
		}
		err = d.Do()
		if err != nil {
			log.Fatal("an error occurred while downloading the file", err)
		}
		log.Println("download completed in", time.Now().Sub(startTime))
	} else {
		log.Println("skipping download")
	}

	err = Install(filename)
	if err != nil {
		log.Println("error installing go", err)
	}
}

// Install will untar download file in /usr/local folder
func Install(filename string) (err error) {
	env := os.Environ()

	versionFilename := "/usr/local/go/VERSION"
	versionBefore, err := GetStringFromText(versionFilename)
	if err != nil {
		return
	}
	log.Println("version before install", versionBefore)

	binTar, err := exec.LookPath("tar")
	if err != nil {
		return
	}
	log.Println("tar found at", binTar)

	argsTar := []string{"tar", "-C", "/usr/local", "-xzf", filepath.Join(downloadFolder, filename)}

	// TODO this will exit
	err = syscall.Exec(binTar, argsTar, env)
	if err != nil {
		return
	}

	versionAfter, err := GetStringFromText(versionFilename)
	if err != nil {
		return
	}
	log.Println("version after install", versionAfter)

	return
}

// GetStringFromText will read file and return its contents
func GetStringFromText(filename string) (result string, err error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	result = string(b)

	return
}

func version() string {
	return "0.0.1"
}
