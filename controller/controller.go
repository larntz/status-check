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

func getChecks() datastructures.Checks {
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
	return checks
}

// StartController runs the controller
func StartController() {

	// handle route using handler function
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		checks := getChecks()
		response, err := json.Marshal(checks)
		if err != nil {
			fmt.Println("error marshalling check")
		}
		fmt.Fprintf(w, string(response))
	})

	// listen to port
	http.ListenAndServe("127.0.0.1:4242", nil)
}
