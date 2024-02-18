// Package worker runs checks assigned by the controller
package worker

import (
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/larntz/status/internal/checks"
	"github.com/larntz/status/internal/data"
	"go.uber.org/zap"
)

// State holds all state for the worker:
// A map containing with keys being the CheckID and
// values are channels allowing check updates to be sent to the thread
type State struct {
	Region              string
	DBClient            data.Database
	HTTPTransport       http.RoundTripper
	Log                 *zap.Logger
	statusChecks        map[string]*checks.StatusCheck
	statusThreads       map[string](chan *checks.StatusCheck)
	wg                  sync.WaitGroup
	statusCheckResultCh chan *checks.StatusCheckResult
}

// NewState creates a new empty State struct
func NewState() *State {
	state := &State{
		statusChecks:        make(map[string]*checks.StatusCheck),
		statusThreads:       make(map[string](chan *checks.StatusCheck)),
		statusCheckResultCh: make(chan *checks.StatusCheckResult, 20000),
	}
	return state
}

// RunWorker runs the worker
func (state *State) RunWorker() {
	go state.sendResultsWorker(30000) // 30 seconds in ms

	firstRun := true
	updateChecksTicker := time.NewTicker(1 * time.Nanosecond)
	statusTicker := time.NewTicker(time.Duration(1) * time.Minute)

	for {
		select {
		case <-updateChecksTicker.C:
			if firstRun {
				updateChecksTicker.Reset(time.Duration(3) * time.Minute)
				firstRun = false
			}
			state.Log.Info("Update Status Checks Start")
			newChecks := state.UpdateChecks()
			for _, c := range newChecks.StatusChecks {
				go state.statusCheck(state.statusThreads[c.ID], rand.Intn(60))
			}
		case <-statusTicker.C:
			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)
			state.Log.Info("Status Ticker", zap.Int("num_goroutines", runtime.NumGoroutine()), zap.Uint64("HeapAlloc", mem.HeapAlloc))
		}
	}

	// TODO graceful shutdowns
	// state.wg.Wait()
}

// UpdateChecks fetches checks from DB and updates threads and state.checks.StatusChecks
func (state *State) UpdateChecks() checks.Checks {
	//TODO Updates are not working properly
	// updated a check url but it kept using the old url

	/*
			  - [x] Change active from true to false. Shuts down goroutine.
			  - [x] Change active from false to true. Starts new goroutine.
		    - [x] Add new check
			  - [ ] Change interval; close existing goroutine and start another.
			  - [ ] Change url; close existing goroutine and start another.
	*/

	newChecks := checks.Checks{}
	// Fetch checks and populate statusChecks map.
	checkList, err := state.DBClient.GetRegionChecks(state.Region)
	if err != nil {
		state.Log.Error("GetRegionChecks failed.", zap.String("error", err.Error()))
	}

	for i, update := range checkList.StatusChecks {
		_, containsKey := state.statusChecks[update.ID]
		if containsKey && state.statusChecks[update.ID].Active {
			// check url or interval change; if yes close thread and start another.
			// better way to do that than a bunch of nested ifs?
			state.statusChecks[update.ID] = &checkList.StatusChecks[i]
			state.statusThreads[update.ID] <- state.statusChecks[update.ID]
			continue
		}
		if containsKey && !state.statusChecks[update.ID].Active { // original check is not active, update is active
			state.statusChecks[update.ID] = &checkList.StatusChecks[i]
			newChecks.StatusChecks = append(newChecks.StatusChecks, checkList.StatusChecks[i])
			//state.statusThreads[update.ID] = make(chan *checks.StatusCheck, 1)
			//state.statusThreads[update.ID] <- state.statusChecks[update.ID]
			continue
		}
		if containsKey && !update.Active { // original check is active, update is not active
			state.statusChecks[update.ID] = &checkList.StatusChecks[i]
			state.statusThreads[update.ID] <- state.statusChecks[update.ID]
			continue
		}
		if !containsKey && update.Active { // add new active check and append to newChecks
			state.statusChecks[update.ID] = &checkList.StatusChecks[i]
			newChecks.StatusChecks = append(newChecks.StatusChecks, checkList.StatusChecks[i])
			state.statusThreads[update.ID] = make(chan *checks.StatusCheck, 1)
			state.statusThreads[update.ID] <- state.statusChecks[update.ID]
		}
	}

	return newChecks

	// TODO
	// update ssl checks
}

func (state *State) sendResultsWorker(intervalMS int) {
	sendTicker := time.NewTicker(time.Duration(intervalMS) * time.Millisecond)
	var results []interface{}
	for {
		select {
		case <-sendTicker.C:
			if len(results) > 0 {
				insertResult, err := state.DBClient.SendResults(results)
				if err != nil {
					state.Log.Error("SendResults", zap.String("error", err.Error()))
					continue
				}
				state.Log.Info("SendResults", zap.String("message", insertResult))
				results = results[:0] // empty results
			} else {
				state.Log.Info("InsertMany - no results to insert")
			}

		case result := <-state.statusCheckResultCh:
			results = append(results, *result)
		}
	}
}
