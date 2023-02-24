// Package worker runs checks assigned by the controller
package worker

import (
	"context"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/larntz/status/internal/application"
	"github.com/larntz/status/internal/checks"
	"github.com/larntz/status/internal/data"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
)

// TODO get ttfb or some kind of request timing
// https://stackoverflow.com/questions/48077098/getting-ttfb-time-to-first-byte-value-in-golang/48077762#48077762

// StartWorker starts the worker process
func StartWorker(app *application.State) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	defer cancel()
	data.CreateResultCollection(ctx, app.DbClient, "check_results")

	workerChecks := data.GetChecks(app.DbClient, app.Region)
	var wg sync.WaitGroup
	app.ChecksMutex.Lock()
	rand.Seed(time.Now().Unix())
	for _, statusCheck := range workerChecks.StatusChecks {
		delay := rand.Int() % statusCheck.Interval
		wg.Add(1)
		go check(&wg, delay, app, statusCheck)

		// TODO check for new checks and create new goroutines for new checks
	}
	app.ChecksMutex.Unlock()
	wg.Wait()
}

func check(wg *sync.WaitGroup, delay int, app *application.State, check checks.StatusCheck) {
	defer wg.Done()
	log.Infof("Preparing check %+v", check)
	time.Sleep(time.Duration(delay) * time.Second)

	result := checks.StatusCheckResult{
		Metadata: checks.StatusCheckMetadata{
			Region:  app.Region,
			CheckID: check.ID,
		},
	}

	client := http.Client{
		Timeout: 15 * time.Second,
	}

	for {
		result.Timestamp = time.Now().UTC()
		resp, err := client.Get(check.URL)
		if err != nil {
			result.ResponseInfo = err.Error()
			go sendCheckResult(app.DbClient, &result)
			log.Errorf("%+v", result)
			return
		}

		result.ResponseCode = resp.StatusCode
		result.ResponseInfo = resp.Status

		// done with resp
		resp.Body.Close()

		result.ResponseTime = 5
		go sendCheckResult(app.DbClient, &result)

		log.Infof("Check Result: %+v", result)
		log.Infof("Check %s sleeping for %d seconds", check.ID, check.Interval)
		time.Sleep(time.Duration(check.Interval) * time.Second)
	}
}

func sendCheckResult(client *mongo.Client, result *checks.StatusCheckResult) {
	coll := client.Database("status").Collection("check_results")
	iResult, err := coll.InsertOne(context.TODO(), result)
	if err != nil {
		log.Errorf("Failed to InsertMany: %s", err)
		return
	}
	log.Infof("Successfully inserted document %s, checkID: %s", iResult.InsertedID, result.Metadata.CheckID)
}
