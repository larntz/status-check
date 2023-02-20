// Package controller get's the party started
package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/larntz/status/internal/application"
	"github.com/larntz/status/internal/data"
	log "github.com/sirupsen/logrus"
)

// StartController runs the controller
func StartController(app *application.State) {

	// handle route using handler function
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		region := "us-east-1"
		checks := data.GetChecks(app.DbClient, region)
		log.Infof("Loaded %d checks for region '%s'", len(checks.StatusChecks), region)
		response, err := json.Marshal(checks)
		if err != nil {
			log.Error("error marshalling check")
		}
		fmt.Fprintf(w, string(response))
	})

	// listen to port
	http.ListenAndServe("127.0.0.1:4242", nil)
}
