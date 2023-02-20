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
	"go.mongodb.org/mongo-driver/mongo"
)

// TODO spawn go routines for each allowed interval 60,120,300,600,900
// and assign checks to threads
// OR
// spawn a goroutine to handle groups of checks with the same interval
// 100 checks x60s could be split and assigned to 5 goroutines with 20 checks each

// TODO get ttfb or some kind of request timing
// https://stackoverflow.com/questions/48077098/getting-ttfb-time-to-first-byte-value-in-golang/48077762#48077762

func check(wg *sync.WaitGroup, dbClient *mongo.Client, checkID string, runID int64, region string, url string) {
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

	client := http.Client{
		Timeout: 20 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		//log.Errorf("Check result runID: %d, error: %s", runID, err)
		result.ResponseInfo = err.Error()
		sendCheckResult(dbClient, &result)
		log.Errorf("%+v", result)
		return
	}
	defer resp.Body.Close()

	result.ResponseCode = resp.StatusCode
	result.ResponseInfo = resp.Status
	result.ResponseTime = 5
	sendCheckResult(dbClient, &result)

	//log.Infof("Check result runID: %d, url: %s, status_code: %d", runID, url, resp.StatusCode)
	log.Infof("%+v", result)
}

func sendCheckResult(dbClient *mongo.Client, result *checks.StatusCheckResult) {
	coll := dbClient.Database("status").Collection("check_results")
	r, err := coll.InsertOne(context.TODO(), result)
	if err != nil {
		log.Errorf("Failed to insert document: %s", err)
	}
	log.Infof("Inserted document with _id: %v\n", r.InsertedID)
}

// StartWorker starts the worker process
func StartWorker(app *application.State) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	defer cancel()
	data.CreateResultCollection(ctx, app.DbClient, "check_results")

	checks := data.GetChecks(app.DbClient, app.Region)
	for {
		runID := time.Now().Unix()
		log.Infof("Starting checks region: %s, runID: %d, %d StatusChecks, %d SSLChecks", app.Region, runID, len(checks.StatusChecks), len(checks.SSLChecks))
		wg := new(sync.WaitGroup)
		for _, statusCheck := range checks.StatusChecks {
			wg.Add(1)
			go check(wg, app.DbClient, statusCheck.ID, runID, app.Region, statusCheck.URL)
		}
		wg.Wait()
		log.Infof("Completed checks region: %s, runID: %d, %d StatusChecks, %d SSLChecks", app.Region, runID, len(checks.StatusChecks), len(checks.SSLChecks))
		time.Sleep(time.Second * 300)

		app.ChecksMutex.Lock()
		checks = data.GetChecks(app.DbClient, app.Region)
		app.ChecksMutex.Unlock()
	}
}
