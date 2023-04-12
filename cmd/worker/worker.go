// Package worker runs checks assigned by the controller
package worker

import (
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
	// Fetch checks and populate statusChecks map.
	checkList, err := state.DBClient.GetRegionChecks(state.Region)
	if err != nil {
		state.Log.Error("GetRegionChecks failed.", zap.String("error", err.Error()))
	}
	for i, check := range checkList.StatusChecks {
		state.statusChecks[check.ID] = &checkList.StatusChecks[i]
	}

	go state.sendResultsWorker()
	// Start checks
	for _, chk := range state.statusChecks {
		if chk.Active {
			state.Log.Info("Adding StatusCheck", zap.String("CheckID", chk.ID), zap.Int("Interval", chk.Interval))
			ch := make(chan *checks.StatusCheck)
			state.statusThreads[chk.ID] = ch
			state.wg.Add(1)
			go state.statusCheck(state.statusThreads[chk.ID])
			state.statusThreads[chk.ID] <- state.statusChecks[chk.ID]
		}
	}

	updateChecksTicker := time.NewTicker(time.Duration(3) * time.Minute)
	statusTicker := time.NewTicker(time.Duration(1) * time.Minute)

	for {
		select {
		case <-updateChecksTicker.C:
			state.Log.Info("Update Status Checks Start")
			// TODO How to handle this so I can still run tests?
			// ----------------
			// newChecks := state.UpdateChecks()
			// for _, c := range newChecks.StatusChecks {
			// 	state.statusThreads[c.ID] = make(chan *checks.StatusCheck)
			// 	go state.statusCheck(state.statusThreads[c.ID])
			// 	state.statusThreads[c.ID] <- state.statusChecks[c.ID]
			// }
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
		    - [ ] Add new check
			  - [ ] Change interval; close existing goroutine and start another.
			  - [ ] Change url; close existing goroutine and start another.
	*/

	newChecks := checks.Checks{}
	// Fetch checks and populate statusChecks map.
	checkList, err := state.DBClient.GetRegionChecks(state.Region)
	if err != nil {
		state.Log.Error("GetRegionChecks failed.", zap.String("error", err.Error()))
	}

	// TODO - see above near `updateChecksTicker`
	// this needs work. I want to test to verify these are being done properly,
	// but won't have channels or any of that setup when running these tests...
	for i, update := range checkList.StatusChecks {
		_, containsKey := state.statusChecks[update.ID]
		if containsKey && state.statusChecks[update.ID].Active {
			// check url or interval change; if yes close thread and start another.
			// better way to do that than a bunch of nested ifs?
			state.statusChecks[update.ID] = &checkList.StatusChecks[i]
			state.statusThreads[update.ID] <- state.statusChecks[update.ID]
		}
		if containsKey && !state.statusChecks[update.ID].Active { // original check is not active, update is active
			state.statusChecks[update.ID] = &checkList.StatusChecks[i]
			newChecks.StatusChecks = append(newChecks.StatusChecks, checkList.StatusChecks[i])
			// state.statusThreads[update.ID] = make(chan *checks.StatusCheck)
			// go state.statusCheck(state.statusThreads[update.ID])
			// state.statusThreads[update.ID] <- state.statusChecks[update.ID]
		}
		if containsKey && !update.Active { // original check is active, update is not active
			state.statusChecks[update.ID] = &checkList.StatusChecks[i]
			state.statusThreads[update.ID] <- state.statusChecks[update.ID]
		}
		if !containsKey && update.Active { // add new active check and start goroutine
			state.statusChecks[update.ID] = &checkList.StatusChecks[i]
			newChecks.StatusChecks = append(newChecks.StatusChecks, checkList.StatusChecks[i])
			// state.statusThreads[update.ID] = make(chan *checks.StatusCheck)
			// go state.statusCheck(state.statusThreads[update.ID])
			// state.statusThreads[update.ID] <- state.statusChecks[update.ID]
		}
	}

	// TODO
	// update ssl checks
}

func (state *State) sendResultsWorker() {
	sendTicker := time.NewTicker(30 * time.Second)
	var results []interface{}
	for {
		select {
		case <-sendTicker.C:
			if len(results) > 1 {
				insertResult, err := state.DBClient.SendResults(results)
				if err != nil {
					state.Log.Error("SendResults Error", zap.String("err", err.Error()))
					continue
				}
				state.Log.Info("SendResults Successful", zap.String("result", insertResult))
				results = results[:0] // empty results
			} else {
				state.Log.Info("InsertMany - no results to insert")
			}

		case result := <-state.statusCheckResultCh:
			results = append(results, result)
		}
	}
}
