// Package controller get's the party started
package controller

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/larntz/status/datastructures"
)

func getChecks() {
	// read checks.csv and get requested checks
	file, err := os.Open("checks.csv")
	if err != nil {
		fmt.Printf("error opening checks.csv: %s\n", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var checks datastructures.Checks
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
		columns := strings.Split(line, ",")
		interval, err := strconv.Atoi(columns[2])
		if err != nil {
			fmt.Printf("unable to convert interval: %s\n", err)
		}
		checks.StatusChecks = append(checks.StatusChecks, datastructures.StatusCheck{
			ID:       columns[0],
			URL:      columns[1],
			Interval: interval,
		})
	}

	if err := scanner.Err(); err != nil {
		fmt.Println(err)
	}

}

// StartController runs the controller
func StartController() {
	var checks datastructures.Checks
	checks.StatusChecks = append(checks.StatusChecks, datastructures.StatusCheck{
		ID:       "check1",
		URL:      "https://blue42.net/404",
		Interval: 60,
		Regions:  []string{"SOFLO"},
	})
	checks.StatusChecks = append(checks.StatusChecks, datastructures.StatusCheck{
		ID:       "check2",
		URL:      "http://lek.net",
		Interval: 60,
		Regions:  []string{"SOFLO"},
	})
	checks.StatusChecks = append(checks.StatusChecks, datastructures.StatusCheck{
		ID:       "check3",
		URL:      "http://slashdot.org",
		Interval: 60,
		Regions:  []string{"SOFLO"},
	})

	checks.SSLChecks = append(checks.SSLChecks, datastructures.SSLCheck{
		ID:       "check1",
		URL:      "https://blue42.net",
		Interval: 60,
	})

	// handle route using handler function
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response, err := json.Marshal(checks)
		if err != nil {
			fmt.Println("error marshalling check")
		}
		fmt.Fprintf(w, string(response))
	})

	// listen to port
	http.ListenAndServe("127.0.0.1:4242", nil)
}
