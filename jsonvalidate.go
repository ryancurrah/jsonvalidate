package main

// Example invocations...
//
// WITHOUT PROFILING:
// go run jsonvalidate.go -jsonfiles '/Users/ryan.currah/Downloads/JSON/*.json'
//
// WITH PROFILING:
// go run jsonvalidate.go -jsonfiles '/Users/ryan.currah/Downloads/JSON/*.json' -cpuprofile cpu.prof
//
// GENERATE PROFILE REPORT PNG:
// go tool pprof -png cpu.prof
//

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sync"
	"time"
)

var wg sync.WaitGroup
var exitCode int
var jsonfiles = flag.String("jsonfiles", "", "file glob path to json files")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")

func main() {
	timeStart := time.Now()

	defer func() {
		pprof.StopCPUProfile()
		os.Exit(exitCode)
	}()

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Println("could not create CPU profile: ", err)
			exitCode = 1
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Println("could not start CPU profile: ", err)
			exitCode = 1
		}
	}

	files, err := filepath.Glob(*jsonfiles)
	if err != nil {
		log.Println(err)
		exitCode = 1
	}

	failed := make(chan bool, len(files))
	for _, file := range files {
		wg.Add(1)
		go validateJSON(file, failed)
	}

	timeSpentWaitingStart := time.Now()
	wg.Wait()
	timeSpentWaitingEnd := time.Since(timeSpentWaitingStart).Seconds()
	close(failed)

	failedCount := 0
	for f := range failed {
		if f {
			failedCount++
			exitCode = 1
		}
	}

	log.Printf(
		"Successfully parsed '%v' files. "+
			"Failed to parse '%v' files. "+
			"All files parsed in '%v' seconds. "+
			"Spent '%v' seconds waiting for goroutines to finish\n",
		len(files)-failedCount,
		failedCount,
		time.Since(timeStart).Seconds(),
		timeSpentWaitingEnd,
	)
}

func validateJSON(f string, failed chan<- bool) {
	defer wg.Done()

	jsonFile, err := os.Open(f)
	if err != nil {
		failed <- true
		log.Println(err)
		return
	}
	defer jsonFile.Close()

	jsonBytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		failed <- true
		log.Println(err)
		return
	}

	var j interface{}
	err = json.Unmarshal(jsonBytes, &j)
	if err != nil {
		failed <- true
		log.Println(f, ":", err)
		return
	}
}
