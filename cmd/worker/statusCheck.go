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

func (state *State) statusCheck(ch chan *checks.StatusCheck) {
	defer state.wg.Done()
	check := <-ch

	// delay to distribute checks over time
	delay := rand.Intn(check.Interval)
	state.Log.Debug("Check Delay", zap.String("CheckID", check.ID), zap.Int("Seconds", delay))
	time.Sleep(time.Duration(delay) * time.Second)

	// Setup HTTP client once before we start the thread loop
	req, err := http.NewRequest("GET", check.URL, nil)
	if err != nil {
		state.Log.Error("failed to create NewRequest", zap.String("err", err.Error()))
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
			Region:  state.Region,
			CheckID: check.ID,
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	ticker := time.NewTicker(time.Duration(check.Interval) * time.Second)

	for {
		select {
		case update, ok := <-ch:
			if !update.Active {
				state.Log.Info("Check no longer active. Exiting.", zap.String("CheckID", check.ID))
				return
			} else if !ok {
				state.Log.Info("Check channel closed. Exiting.\n", zap.String("CheckID", check.ID))
				return
			}
			check = update

		case <-ticker.C:
			state.Log.Debug("Starting Check", zap.String("CheckID", check.ID), zap.Bool("Active", check.Active))
			http.DefaultClient.Timeout = time.Duration(check.HTTPTimeout) * time.Second
			start = time.Now()
			resp, err := http.DefaultTransport.RoundTrip(req)
			if err != nil {
				result.ResponseInfo = err.Error()
				state.Log.Error("httpClient.Get() error",
					zap.String("check_id", result.Metadata.CheckID),
					zap.String("region", result.Metadata.Region),
					zap.Int("response_code", result.ResponseCode),
					zap.String("response_info", result.ResponseInfo),
				)
				go sendStatusCheckResult(state.DBClient, state.Log, &result)
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

			go sendStatusCheckResult(state.DBClient, state.Log, &result)

			state.Log.Info("check_result",
				zap.String("check_id", result.Metadata.CheckID),
				zap.String("region", result.Metadata.Region),
				zap.Int("response_code", result.ResponseCode),
				zap.String("response_info", result.ResponseInfo),
				zap.Int("interval", check.Interval))
		}
	}
}
