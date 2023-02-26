// Package worker runs checks assigned by the controller
package worker

import (
	"context"
	"crypto/tls"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/larntz/status/internal/application"
	"github.com/larntz/status/internal/checks"
	"github.com/larntz/status/internal/data"
	"go.uber.org/zap"
)

// TODO get ttfb or some kind of request timing
// https://stackoverflow.com/questions/48077098/getting-ttfb-time-to-first-byte-value-in-golang/48077762#48077762

// StartWorker starts the worker process
func StartWorker(app *application.State) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	defer cancel()
	data.CreateResultCollection(ctx, app.DbClient, "check_results")

	workerChecks := data.GetChecks(app.DbClient, app.Region, app.Log)
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
	app.Log.Debug("Preparing check", zap.Any("check", check))
	time.Sleep(time.Duration(delay) * time.Second)

	http.DefaultClient.Timeout = 15 * time.Second
	req, _ := http.NewRequest("GET", check.URL, nil)

	var start, dns, tlsHandshake, connect time.Time
	var ttfb, dnsTime, tlsTime, connectTime time.Duration

	trace := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			dnsTime = time.Since(dns)
		},

		TLSHandshakeStart: func() { tlsHandshake = time.Now() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			tlsTime = time.Since(tlsHandshake)
		},

		ConnectStart: func(network, addr string) { connect = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			connectTime = time.Since(connect)
		},

		GotFirstResponseByte: func() {
			ttfb = time.Since(start)
		},
	}

	result := checks.StatusCheckResult{
		Metadata: checks.StatusCheckMetadata{
			Region:  app.Region,
			CheckID: check.ID,
		},
	}

	for {
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		start = time.Now()
		resp, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			log.Fatal(err)
		}

		result.Timestamp = start.UTC()
		//resp, err := client.Get(check.URL)
		if err != nil {
			result.ResponseInfo = err.Error()
			go sendCheckResult(app, &result)
			app.Log.Error("client.Get() error",
				zap.String("check_id", result.Metadata.CheckID),
				zap.String("region", result.Metadata.Region),
				zap.Int("response_code", result.ResponseCode),
				zap.String("response_info", result.ResponseInfo),
			)
			return
		}

		result.ResponseCode = resp.StatusCode
		result.ResponseInfo = resp.Status
		result.TTFB = ttfb.Milliseconds()
		result.DNSTiming = dnsTime.Milliseconds()
		result.TLSTiming = tlsTime.Milliseconds()
		result.ConnectTiming = connectTime.Milliseconds()

		// done with resp
		resp.Body.Close()

		go sendCheckResult(app, &result)

		app.Log.Info("check_result",
			zap.String("check_id", result.Metadata.CheckID),
			zap.String("region", result.Metadata.Region),
			zap.Int("response_code", result.ResponseCode),
			zap.String("response_info", result.ResponseInfo),
			zap.Int("interval", check.Interval))

		time.Sleep(time.Duration(check.Interval) * time.Second)
	}
}

func sendCheckResult(app *application.State, result *checks.StatusCheckResult) {
	coll := app.DbClient.Database("status").Collection("check_results")
	iResult, err := coll.InsertOne(context.TODO(), result)
	if err != nil {
		app.Log.Error("Failed to InsertMany", zap.String("err", err.Error()))
		return
	}
	app.Log.Debug("Successfully inserted document", zap.Any("id", iResult.InsertedID), zap.String("request_id", result.Metadata.CheckID))
}
