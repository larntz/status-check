package worker

import (
	"sync"
	"time"

	"github.com/larntz/status/internal/checks"
	"github.com/larntz/status/internal/data"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// Scheduler manages job execution
type Scheduler struct {
	statusChecks map[string]*checks.StatusCheck
	sslChecks    map[string]*checks.SSLCheck
	mutex        sync.Mutex
	stop         chan bool
	stopped      chan bool
	wg           sync.WaitGroup
	Region       string
	DBClient     *mongo.Client
	Log          *zap.Logger
}

// NewScheduler returns a new JobScheduler
func NewScheduler() *Scheduler {
	return &Scheduler{
		statusChecks: make(map[string]*checks.StatusCheck),
		sslChecks:    make(map[string]*checks.SSLCheck),
		stop:         make(chan bool),
		stopped:      make(chan bool),
	}
}

// Start scheduler
func (scheduler *Scheduler) Start() {
	startChecks := data.GetChecks(scheduler.DBClient,
		scheduler.Region, scheduler.Log)

	for i, chk := range startChecks.StatusChecks {
		if chk.Active {
			scheduler.Log.Info("Adding StatusCheck", zap.String("CheckID", chk.ID), zap.Int("Interval", chk.Interval))
			scheduler.statusChecks[chk.ID] = &startChecks.StatusChecks[i]
			scheduler.wg.Add(1)
			go scheduler.statusChecker(scheduler.statusChecks[chk.ID])
		}
	}

	for _, chk := range startChecks.SSLChecks {
		scheduler.Log.Info("Adding SSLCheck", zap.String("CheckID", chk.ID))
		// scheduler.wg.Add(1)
		// go scheduler.runSSLCheck(chk)
	}

	updateChecksTicker := time.NewTicker(time.Duration(1) * time.Minute)

	for {
		select {
		case <-scheduler.stop:
			scheduler.stopped <- true
			return
		case <-updateChecksTicker.C:
			scheduler.Log.Info("Refreshing Status Checks Start")
			scheduler.UpdateChecks()
			scheduler.Log.Info("Refreshing Status Checks Complete")
		}
	}
}

// Stop job scheduler
func (scheduler *Scheduler) Stop() {
	close(scheduler.stop)
	scheduler.wg.Wait()
}

// AddStatusCheck to the scheduler
func (scheduler *Scheduler) AddStatusCheck(check *checks.StatusCheck) {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	scheduler.statusChecks[check.ID] = check
	go scheduler.statusChecker(check)
	scheduler.wg.Add(1)
}

// UpdateChecks updates and/or adds checks
func (scheduler *Scheduler) UpdateChecks() {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	updatedChecks := data.GetChecks(scheduler.DBClient,
		scheduler.Region, scheduler.Log)

	var found bool
	for i, update := range updatedChecks.StatusChecks {
		found = false
		for j, chk := range scheduler.statusChecks {
			if update.ID == chk.ID && chk.Active {
				found = true
				scheduler.Log.Info("Updating StatusCheck", zap.String("CheckID", update.ID), zap.Int("Interval", update.Interval))
				*scheduler.statusChecks[j] = updatedChecks.StatusChecks[i]
			}
		}
		if !found && update.Active {
			scheduler.Log.Info("Adding StatusCheck", zap.String("CheckID", update.ID), zap.Int("Interval", update.Interval))
			scheduler.wg.Add(1)
			scheduler.statusChecks[update.ID] = &updatedChecks.StatusChecks[i]
			go scheduler.statusChecker(scheduler.statusChecks[update.ID])
		}
	}

	// TODO
	//for _, chk := range updatedChecks.SSLChecks {
	// scheduler.Log.Info("Adding SSLCheck", zap.String("CheckID", chk.ID))
	// scheduler.wg.Add(1)
	// go scheduler.runSSLCheck(chk)
	//}
}

// UpdateJob already scheduled
func (scheduler *Scheduler) UpdateJob(ID string, checkUpdate *checks.StatusCheck) {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	if check, ok := scheduler.statusChecks[ID]; ok {
		check.Interval = checkUpdate.Interval
		check.URL = checkUpdate.URL
	}
}

// RemoveJob from scheduler
func (scheduler *Scheduler) RemoveJob(ID string) {
	scheduler.mutex.Lock()
	defer scheduler.mutex.Unlock()

	if _, ok := scheduler.statusChecks[ID]; ok {
		scheduler.statusChecks[ID].Active = false
	}
}
