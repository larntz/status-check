package worker

import (
	"crypto/tls"
	"math/rand"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/larntz/status/internal/checks"
	"go.uber.org/zap"
)

func (scheduler *Scheduler) statusChecker(check *checks.StatusCheck) {
	defer scheduler.wg.Done()
	scheduler.Log.Debug("Preparing check", zap.Any("check", check))
	delay := rand.Intn(check.Interval)
	scheduler.Log.Debug("Check Delay", zap.String("CheckID", check.ID), zap.Int("Seconds", delay))
	time.Sleep(time.Duration(delay) * time.Second)

	req, err := http.NewRequest("GET", check.URL, nil)
	if err != nil {
		scheduler.Log.Error("failed to create NewRequest", zap.String("err", err.Error()))
	}

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
			Region:  scheduler.Region,
			CheckID: check.ID,
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	ticker := time.NewTicker(time.Duration(check.Interval) * time.Second)

	for {
		select {
		case <-ticker.C:
			scheduler.Log.Debug("Starting Check", zap.String("CheckID", check.ID), zap.Bool("Active", check.Active))
			if check.Active {
				http.DefaultClient.Timeout = time.Duration(check.HTTPTimeout) * time.Second
				start = time.Now()
				resp, err := http.DefaultTransport.RoundTrip(req)
				if err != nil {
					result.ResponseInfo = err.Error()
					scheduler.Log.Error("client.Get() error",
						zap.String("check_id", result.Metadata.CheckID),
						zap.String("region", result.Metadata.Region),
						zap.Int("response_code", result.ResponseCode),
						zap.String("response_info", result.ResponseInfo),
					)
					go sendStatusCheckResult(scheduler.DBClient, scheduler.Log, &result)
					continue
				}

				result.Timestamp = start.UTC()
				result.ResponseCode = resp.StatusCode
				result.ResponseInfo = resp.Status
				result.TTFB = ttfb.Milliseconds()
				result.DNSTiming = dnsTime.Milliseconds()
				result.TLSTiming = tlsTime.Milliseconds()
				result.ConnectTiming = connectTime.Milliseconds()

				// done with resp
				resp.Body.Close()

				go sendStatusCheckResult(scheduler.DBClient, scheduler.Log, &result)

				scheduler.Log.Info("check_result",
					zap.String("check_id", result.Metadata.CheckID),
					zap.String("region", result.Metadata.Region),
					zap.Int("response_code", result.ResponseCode),
					zap.String("response_info", result.ResponseInfo),
					zap.Int("interval", check.Interval))
			} else {
				scheduler.Log.Info("Check Now Inactive", zap.String("CheckID", check.ID))
				return
			}
		case stop, ok := <-scheduler.stop:
			if !ok || stop {
				scheduler.Log.Info("statusChecker Stopping", zap.String("CheckID", check.ID))
				return
			}
		}
	}
}
