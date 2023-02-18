// Package controller get's the party started
package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/larntz/status/internal/data"
)

// StartController runs the controller
func StartController() {

	// handle route using handler function
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		region := "us-east-1"
		checks := data.GetChecks(region)
		log.Printf("Loaded %d checks for region '%s'", len(checks.StatusChecks), region)
		response, err := json.Marshal(checks)
		if err != nil {
			fmt.Println("error marshalling check")
		}
		fmt.Fprintf(w, string(response))
	})

	// listen to port
	http.ListenAndServe("127.0.0.1:4242", nil)
}
