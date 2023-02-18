// Package worker runs checks assigned by the controller
package worker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/larntz/status/internal/checks"
)

// TODO spawn go routines for each allowed interval 60,120,300,600,900
// and assign checks to threads
// OR
// spawn a goroutine to handle groups of checks with the same interval
// 100 checks x60s could be split and assigned to 5 goroutines with 20 checks each

// TODO get ttfb or some kind of request timing
// https://stackoverflow.com/questions/48077098/getting-ttfb-time-to-first-byte-value-in-golang/48077762#48077762

func getChecks() checks.Checks {
	// get checks
	var checks checks.Checks
	// contoller url sould be configurable via env
	resp, err := http.Get("http://localhost:4242")
	if err != nil {
		fmt.Printf("Error getting checks from controller: %s\n", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("error reading body from resp: %s\n", err)
	}
	err = json.Unmarshal(body, &checks)
	if err != nil {
		fmt.Printf("error unmarshalling response from controller: %s\n", err)
	}
	return checks

}

func check(wg *sync.WaitGroup, url string) {
	defer wg.Done()
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("http.Get error: %s\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	time := time.Now().UTC()
	fmt.Printf("CHECK: %s,%d,%s\n", url, resp.StatusCode, time)
}

// StartWorker starts the worker process
func StartWorker() {
	checks := getChecks()
	for {
		fmt.Println("INFO: Starting checks")
		wg := new(sync.WaitGroup)
		for _, statusCheck := range checks.StatusChecks {
			wg.Add(1)
			go check(wg, statusCheck.URL)
		}
		wg.Wait()
		fmt.Printf("INFO: Completed checks\n\n")
		time.Sleep(time.Second * 20)
		checks = getChecks()
	}
}
