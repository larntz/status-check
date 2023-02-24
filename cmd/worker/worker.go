// Package worker runs checks assigned by the controller
package worker

import (
	"context"
	"math/rand"
	"net/http"
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

func check(ctx context.Context, delay int, app *application.State, check *checks.StatusCheck) {
	time.Sleep(time.Duration(delay) * time.Second)

	metadata := checks.StatusCheckMetadata{
		Region:  app.Region,
		CheckID: check.ID,
	}
	for {
		result := checks.StatusCheckResult{
			ID:        check.ID,
			Timestamp: time.Now().UTC(),
			Region:    app.Region,
			Metadata:  metadata,
		}

		client := http.Client{
			Timeout: 15 * time.Second,
		}
		resp, err := client.Get(check.URL)
		if err != nil {
			result.ResponseInfo = err.Error()
			go sendCheckResult(app, &result)
			log.Errorf("%+v", result)
			return
		}
		defer resp.Body.Close()

		result.ResponseCode = resp.StatusCode
		result.ResponseInfo = resp.Status

		result.ResponseTime = 5
		go sendCheckResult(app, &result)

		log.Infof("%+v", result)
		time.Sleep(time.Duration(check.Interval) * time.Second)
	}
}

func startChecks(runID int64, app *application.State, workerChecks *checks.Checks) {
	log.Infof("Starting checks region: %s, runID: %d, %d StatusChecks, %d SSLChecks", app.Region, runID, len(workerChecks.StatusChecks), len(workerChecks.SSLChecks))
	app.ChecksMutex.Lock()

	rand.Seed(time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, statusCheck := range workerChecks.StatusChecks {
		delay := rand.Int() % statusCheck.Interval
		go check(ctx, delay, app, &statusCheck)
	}
	app.ChecksMutex.Unlock()
}

func sendCheckResult(app *application.State, result *checks.StatusCheckResult) {
	coll := app.DbClient.Database("status").Collection("check_results")
	iResult, err := coll.InsertOne(context.TODO(), result)
	if err != nil {
		log.Errorf("Failed to InsertMany: %s", err)
		return
	}
	log.Infof("Successfully inserted document %s, checkID: %s", iResult.InsertedID, result.ID)
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
