// Package worker runs checks assigned by the controller
package worker

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/larntz/status/internal/application"
	"github.com/larntz/status/internal/checks"
	"github.com/larntz/status/internal/data"
	log "github.com/sirupsen/logrus"
)

// TODO spawn go routines for each allowed interval 60,120,300,600,900
// and assign checks to threads
// OR
// spawn a goroutine to handle groups of checks with the same interval
// 100 checks x60s could be split and assigned to 5 goroutines with 20 checks each

// TODO get ttfb or some kind of request timing
// https://stackoverflow.com/questions/48077098/getting-ttfb-time-to-first-byte-value-in-golang/48077762#48077762

func check(wg *sync.WaitGroup, app *application.State, checkID string, runID int64, region string, url string, out chan checks.StatusCheckResult) {
	defer wg.Done()
	metadata := checks.StatusCheckMetadata{
		Region:  region,
		CheckID: checkID,
	}
	result := checks.StatusCheckResult{
		ID:        checkID,
		Timestamp: time.Unix(runID, 0),
		Region:    region,
		Metadata:  metadata,
	}

	// This will time out the http client stuff. The entire run should take close to this amount of time, any
	// check not finished by the Timeout will be canceled.
	client := http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		//log.Errorf("Check result runID: %d, error: %s", runID, err)
		result.ResponseInfo = err.Error()
		out <- result
		//go sendCheckResult(app, &result)
		log.Errorf("%+v", result)
		return
	}
	defer resp.Body.Close()

	result.ResponseCode = resp.StatusCode
	result.ResponseInfo = resp.Status

	result.ResponseTime = 5
	out <- result
	//go sendCheckResult(app, &result)

	//log.Infof("Check result runID: %d, url: %s, status_code: %d", runID, url, resp.StatusCode)
	log.Infof("%+v", result)
}

func startChecks(runID int64, app *application.State, workerChecks *checks.Checks) {
	log.Infof("Starting checks region: %s, runID: %d, %d StatusChecks, %d SSLChecks", app.Region, runID, len(workerChecks.StatusChecks), len(workerChecks.SSLChecks))
	start := time.Now()
	checkWg := new(sync.WaitGroup)
	app.ChecksMutex.Lock()
	out := make(chan checks.StatusCheckResult, len(workerChecks.StatusChecks))
	for _, statusCheck := range workerChecks.StatusChecks {
		checkWg.Add(1)
		go check(checkWg, app, statusCheck.ID, runID, app.Region, statusCheck.URL, out)
		time.Sleep(500 * time.Nanosecond)
	}
	app.ChecksMutex.Unlock()
	checkWg.Wait()
	close(out)
	log.Infof("Completed checks region: %s, runID: %d, %d StatusChecks, %d SSLChecks (elapsed: %s)", app.Region, runID, len(workerChecks.StatusChecks), len(workerChecks.SSLChecks), time.Since(start))
	log.Infof("Sending check results to mongo.")
	sendCheckResults(app, out)
}

func sendCheckResults(app *application.State, out chan checks.StatusCheckResult) {
	coll := app.DbClient.Database("status").Collection("check_results")
	var results []interface{}
	for result := range out {
		results = append(results, result)
	}

	iResult, err := coll.InsertMany(context.TODO(), results)
	if err != nil {
		log.Errorf("Failed to InsertMany: %s", err)
		return
	}
	log.Infof("Successfully inserted %d documents", len(iResult.InsertedIDs))
}

// StartWorker starts the worker process
func StartWorker(app *application.State) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	defer cancel()
	data.CreateResultCollection(ctx, app.DbClient, "check_results")

	workerChecks := data.GetChecks(app.DbClient, app.Region)
	for {
		runID := time.Now().Unix()
		log.Infof("running %d checks", len(workerChecks.StatusChecks))
		go startChecks(runID, app, &workerChecks)

		time.Sleep(time.Second * 300)
		app.ChecksMutex.Lock()
		workerChecks = data.GetChecks(app.DbClient, app.Region)
		app.ChecksMutex.Unlock()
	}
}
