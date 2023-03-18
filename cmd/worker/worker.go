// Package worker runs checks assigned by the controller
package worker

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/larntz/status/internal/checks"
	"github.com/larntz/status/internal/data"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

func sendStatusCheckResult(dbClient *mongo.Client, log *zap.Logger, result *checks.StatusCheckResult) {
	coll := dbClient.Database("status").Collection("check_results")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(1)*time.Minute)
	defer cancel()
	iResult, err := coll.InsertOne(ctx, result)
	if err != nil {
		log.Error("InsertOne Failed", zap.String("err", err.Error()))
		return
	}
	log.Debug("InsertOne Successful", zap.Any("id", iResult.InsertedID), zap.String("request_id", result.Metadata.CheckID))
}

// State holds all state for the worker:
// A map containing with keys being the CheckID and
// values are channels allowing check updates to be sent to the thread
type State struct {
	Region        string
	DBClient      *mongo.Client
	Log           *zap.Logger
	statusChecks  map[string]*checks.StatusCheck
	statusThreads map[string](chan *checks.StatusCheck)
	wg            sync.WaitGroup
}

// NewState creates a new empty State struct
func NewState() *State {
	return &State{
		statusChecks:  make(map[string]*checks.StatusCheck),
		statusThreads: make(map[string](chan *checks.StatusCheck)),
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

	updateChecksTicker := time.NewTicker(time.Duration(1) * time.Minute)
	statusTicker := time.NewTicker(time.Duration(1) * time.Minute)

	for {
		select {
		case <-updateChecksTicker.C:
			state.Log.Info("Refreshing Status Checks Start")
			go state.UpdateChecks()
		case <-statusTicker.C:
			state.Log.Info("Status Ticker", zap.Int("num_goroutines", runtime.NumGoroutine()))
		}
	}

	// TODO graceful shutdowns
	// state.wg.Wait()
}

// UpdateChecks fetches checks from DB and updates threads and state.checks.StatusChecks
func (state *State) UpdateChecks() {
	// Fetch checks and populate statusChecks map.
	checkList := data.GetChecks(state.DBClient,
		state.Region, state.Log)

	for i, update := range checkList.StatusChecks {
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
