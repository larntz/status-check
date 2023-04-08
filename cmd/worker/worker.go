// Package worker runs checks assigned by the controller
package worker

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/larntz/status/internal/checks"
	"github.com/larntz/status/internal/data"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// State holds all state for the worker:
// A map containing with keys being the CheckID and
// values are channels allowing check updates to be sent to the thread
type State struct {
	Region              string
	DBClient            *mongo.Client
	Log                 *zap.Logger
	statusChecks        map[string]*checks.StatusCheck
	statusThreads       map[string](chan *checks.StatusCheck)
	wg                  sync.WaitGroup
	statusCheckResultCh chan *checks.StatusCheckResult
}

// NewState creates a new empty State struct
func NewState() *State {
	return &State{
		statusChecks:        make(map[string]*checks.StatusCheck),
		statusThreads:       make(map[string](chan *checks.StatusCheck)),
		statusCheckResultCh: make(chan *checks.StatusCheckResult, 20000),
	}
}

// RunWorker runs the worker
func (state *State) RunWorker() {
	// Fetch checks and populate statusChecks map.
	checkList := data.GetChecks(state.DBClient,
		state.Region, state.Log)
	for i, check := range checkList.StatusChecks {
		state.statusChecks[check.ID] = &checkList.StatusChecks[i]
	}

	go state.sendStatusCheckResult()
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
			state.UpdateChecks()
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
func (state *State) UpdateChecks() {
	//TODO Updates are not working properly
	// updated a check url but it kept using the old url

	/*

			  - [x] Change active from true to false. Shuts down goroutine.
			  - [x] Change active from false to true. Starts new goroutine.
			  - [ ] Change interval changes check interval and ticker. Should this exit the goroutine and start a new one?
		    - [ ] Change url.

	*/

	// Fetch checks and populate statusChecks map.
	checkList := data.GetChecks(state.DBClient,
		state.Region, state.Log)

	for i, update := range checkList.StatusChecks {
		fmt.Printf("update: %+v\n", update)
		_, containsKey := state.statusChecks[update.ID]
		if containsKey {
			if state.statusChecks[update.ID].Active { // both checks active
				state.statusChecks[update.ID] = &checkList.StatusChecks[i]
				state.statusThreads[update.ID] <- state.statusChecks[update.ID]
			} else if !state.statusChecks[update.ID].Active { // original check is not active, update is active
				state.statusChecks[update.ID] = &checkList.StatusChecks[i]
				state.statusThreads[update.ID] = make(chan *checks.StatusCheck)
				go state.statusCheck(state.statusThreads[update.ID])
				state.statusThreads[update.ID] <- state.statusChecks[update.ID]
			} else if !update.Active { // original check is active, update is not active
				state.statusChecks[update.ID] = &checkList.StatusChecks[i]
				state.statusThreads[update.ID] <- state.statusChecks[update.ID]
			}
		} else if !containsKey && update.Active { // add new active check and start goroutine
			state.statusChecks[update.ID] = &checkList.StatusChecks[i]
			state.statusThreads[update.ID] = make(chan *checks.StatusCheck)
			go state.statusCheck(state.statusThreads[update.ID])
			state.statusThreads[update.ID] <- state.statusChecks[update.ID]
		}
	}

	// TODO
	// update ssl checks
}

func (state *State) sendStatusCheckResult() {
	coll := state.DBClient.Database("status").Collection("check_results")

	sendTicker := time.NewTicker(30 * time.Second)
	var results []interface{}
	for {
		select {
		case <-sendTicker.C:
			if len(results) > 1 {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				insertResult, err := coll.InsertMany(ctx, results)
				if err != nil {
					state.Log.Error("InsertMany Failed", zap.String("err", err.Error()))
					continue
				}
				state.Log.Info("InsertMany Successful", zap.Int("Document Count", len(insertResult.InsertedIDs)))
				results = results[:0]
				cancel()
			} else {
				state.Log.Info("InsertMany - no results to insert")
			}

		case result := <-state.statusCheckResultCh:
			results = append(results, result)

		}
	}
}
