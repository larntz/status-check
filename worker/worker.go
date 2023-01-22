// Package worker runs checks assigned by the controller
package worker

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/larntz/status/datastructures"
)

// TODO spawn go routines for each allowed interval 60,120,300,600,900
// and assign checks to threads
// OR
// spawn a goroutine to handle groups of checks with the same interval
// 100 checks x60s could be split and assigned to 5 goroutines with 20 checks each

// TODO get ttfb or some kind of request timing
// https://stackoverflow.com/questions/48077098/getting-ttfb-time-to-first-byte-value-in-golang/48077762#48077762

func getChecks() datastructures.Checks {
	// get checks
	var checks datastructures.Checks
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

// StartWoker starts the worker process
func StartWoker() {
	checks := getChecks()
	for {
		for _, statusCheck := range checks.StatusChecks {
			resp, err := http.Get(statusCheck.URL)
			if err != nil {
				fmt.Printf("http.Get error: %s\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			fmt.Println("Checking ", statusCheck.URL)
			fmt.Println("CheckTime:", time.Now())
			fmt.Println("StatusCode:", resp.StatusCode)
			// not an ssl check
			// fmt.Printf("CertificateExpriation: %s\n\n", resp.TLS.PeerCertificates[0].NotAfter)
		}
		time.Sleep(time.Second * 20)
		checks = getChecks()
	}
}
