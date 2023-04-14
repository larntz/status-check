package worker

import (
	"context"
	"net/http"
	"time"

	"github.com/larntz/status/internal/checks"
	"go.uber.org/zap"
)

func (state *State) statusCheck(ch chan *checks.StatusCheck, delay int) {
	defer state.wg.Done()
	check := <-ch

	// delay to distribute checks over time
	state.Log.Debug("Check Delay", zap.String("CheckID", check.ID), zap.Int("Seconds", delay))
	time.Sleep(time.Duration(delay) * time.Second)

	// Setup HTTP client once before we start the thread loop
	req, err := http.NewRequest("GET", check.URL, nil)
	if err != nil {
		state.Log.Error("failed to create NewRequest", zap.String("err", err.Error()))
	}

	reqTrace := NewRequestTrace()
	var result checks.StatusCheckResult

	// run the check [almost] immediately, then after the first
	// run Reset ticker to Interval. Helps with testing also.
	ticker := time.NewTicker(1 * time.Nanosecond)
	firstRun := true

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
			state.Log.Debug("Check updated.", zap.Any("check", update))
			check = update

		case <-ticker.C:
			if firstRun {
				firstRun = false
				ticker.Reset(time.Duration(check.Interval) * time.Second)
			}
			state.Log.Debug("Starting Check", zap.String("CheckID", check.ID), zap.Bool("Active", check.Active))
			state.Log.Debug("Check Details", zap.Any("check", check))

			result = checks.StatusCheckResult{
				Metadata: checks.StatusCheckMetadata{
					Region:  state.Region,
					CheckID: check.ID,
				},
			}

			timeout := time.Duration(check.HTTPTimeout) * time.Second
			ctx, cancelCTX := context.WithTimeout(context.Background(), timeout)

			// TODO every result is getting sent to the database twice for some reason.
			resp, err := reqTrace.TraceRequest(ctx, state.HTTPTransport, req)
			if err != nil {
				result.ResponseInfo = err.Error()
				state.Log.Error("httpClient.Get() error",
					zap.String("check_id", result.Metadata.CheckID),
					zap.String("region", result.Metadata.Region),
					zap.Int("response_code", result.ResponseCode),
					zap.String("response_info", result.ResponseInfo),
				)
				result.ResponseCode = 0
				state.statusCheckResultCh <- &result
				cancelCTX()
				continue
			}

			result.Timestamp = reqTrace.start
			result.ResponseCode = resp.StatusCode
			result.ResponseInfo = resp.Status
			result.TTFB = reqTrace.TTFB.Milliseconds()
			result.DNSTiming = reqTrace.DNSDur.Milliseconds()
			result.TLSTiming = reqTrace.TLSHandshakeDur.Milliseconds()
			result.ConnectTiming = reqTrace.ConnDur.Milliseconds()

			// done with resp
			resp.Body.Close()
			cancelCTX()

			state.statusCheckResultCh <- &result

			state.Log.Info("check_result",
				zap.String("check_id", result.Metadata.CheckID),
				zap.String("region", result.Metadata.Region),
				zap.Int("response_code", result.ResponseCode),
				zap.String("response_info", result.ResponseInfo),
				zap.Int("interval", check.Interval))
		}
	}
}
