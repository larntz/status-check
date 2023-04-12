package worker

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/larntz/status/internal/checks"
	"github.com/larntz/status/internal/test"
	"go.uber.org/zap"
)

func TestUpdatechecks(t *testing.T) {
	region := "us-test-1"
	workerState := NewState()
	workerState.wg.Add(1)
	workerState.Region = region
	mockDB := test.MockDB{}
	workerState.DBClient = &mockDB

	log, err := zap.NewProduction()
	if err != nil {
		fmt.Println("Unable to setup logger. Exiting...")
		os.Exit(1)
	}
	defer log.Sync()

	if len(workerState.statusChecks) != 0 {
		t.Fatalf("Wanted 0 checks, got %d", len(workerState.statusChecks))
	}

	// add two checks
	chk1, _ := mockDB.GetRegionChecks(region)
	fmt.Printf("mockDB has %d checks\n", len(chk1.StatusChecks))
	mockDB.AddCheck(checks.StatusCheck{
		ID:          "test-check-1",
		URL:         "https://gitea.chacarntz.net",
		Interval:    10,
		HTTPTimeout: 5,
		Regions:     []string{"test-region-1", "test-region-2"},
		Modified:    time.Now().UTC(),
		Serial:      0,
		Active:      true,
	})
	mockDB.AddCheck(checks.StatusCheck{
		ID:          "test-check-2",
		URL:         "https://blue42.net",
		Interval:    5,
		HTTPTimeout: 15,
		Regions:     []string{"test-region-1", "test-region-2"},
		Modified:    time.Now().UTC(),
		Serial:      0,
		Active:      true,
	})
	chk2, _ := mockDB.GetRegionChecks(region)
	fmt.Printf("mockDB has %d checks\n", len(chk2.StatusChecks))

	fmt.Printf("workerState has %d checks\n", len(workerState.statusChecks))
	workerState.UpdateChecks()
	fmt.Printf("workerState has %d checks\n", len(workerState.statusChecks))

	if len(workerState.statusChecks) != 2 {
		t.Fatal("statusChecks != 2")
	}
}
