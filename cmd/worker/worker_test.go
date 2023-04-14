package worker

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/larntz/status/internal/checks"
	"github.com/larntz/status/internal/test"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

var testChecks = []checks.StatusCheck{
	{
		ID:          "test-check-1",
		URL:         "https://gitea.chacarntz.net",
		Interval:    1,
		HTTPTimeout: 5,
		Regions:     []string{"test-region-1", "test-region-2"},
		Modified:    time.Now().UTC(),
		Serial:      0,
		Active:      true,
	},
	{
		ID:          "test-check-2",
		URL:         "https://blue42.net",
		Interval:    1,
		HTTPTimeout: 15,
		Regions:     []string{"test-region-1", "test-region-2"},
		Modified:    time.Now().UTC(),
		Serial:      0,
		Active:      true,
	},
}

// setupState returns a State struct without a DBClient to be used in testing
func setupState() *State {
	state := NewState()
	state.Region = "us-test-1"
	log, _ := observer.New(zap.DebugLevel)
	state.Log = zap.New(log)
	return state
}

func TestStatusChecks(t *testing.T) {
	workerState := setupState()
	mockDB := test.MockDB{}
	workerState.DBClient = &mockDB

	trans := test.HTTPTransport{
		Response: &http.Response{
			StatusCode: 200,
			Status:     "test code 200",
		},
	}
	trans.Response.Body = &test.Body{}
	workerState.HTTPTransport = &trans

	ch := make(chan *checks.StatusCheck)
	workerState.wg.Add(1)
	go workerState.statusCheck(ch, 0)
	ch <- &testChecks[0]

	result := <-workerState.statusCheckResultCh

	// response status code
	if trans.Response.StatusCode != result.ResponseCode {
		t.Fatalf("response code fail. Want: %d. Got: %d", trans.Response.StatusCode,
			result.ResponseCode)
	}
	// response status
	if trans.Response.Status != result.ResponseInfo {
		t.Fatalf("response code fail. Want: %s. Got: %s", trans.Response.Status,
			result.ResponseInfo)
	}

	/* TODO
	- test timings
	- test check changes
	  * url
	  * interval
	  *
	*/

}

func TestUpdateChecks(t *testing.T) {
	workerState := setupState()
	defer workerState.Log.Sync()
	mockDB := test.MockDB{}
	workerState.DBClient = &mockDB

	if len(workerState.statusChecks) != 0 {
		t.Fatalf("Wanted 0 checks, got %d", len(workerState.statusChecks))
	}

	// add testChecks
	for _, check := range testChecks {
		mockDB.AddCheck(check)
	}

	t.Log("Testing UpdateChecks on emtpy state.")
	newChecks := workerState.UpdateChecks()
	if len(workerState.statusChecks) != 2 {
		t.Fatal("statusChecks != 2")
	}

	if !reflect.DeepEqual(mockDB.Checks.StatusChecks, newChecks.StatusChecks) {
		t.Fatalf("mockDB checks != workerState. Got: %+v, Want: %+v", newChecks.StatusChecks, mockDB.Checks.StatusChecks)
	}

	// next test
	/// emtpy channel buffers
	for _, c := range mockDB.Checks.StatusChecks {
		if len(workerState.statusThreads[c.ID]) > 0 {
			<-workerState.statusThreads[c.ID]
		}
	}
	t.Log("Testing UpdateChecks with active -> inactive")
	mockDB.Checks.StatusChecks[0].Active = false
	_ = workerState.UpdateChecks()
	for i, c := range mockDB.Checks.StatusChecks {
		v, ok := workerState.statusChecks[c.ID]
		if !ok {
			t.Fatalf("mockDB checks != workerState. \nGot: %+v, \nWant: %+v\n", v, mockDB.Checks.StatusChecks[i])
		}
	}

	// next test
	/// emtpy channel buffers
	for _, c := range mockDB.Checks.StatusChecks {
		if len(workerState.statusThreads[c.ID]) > 0 {
			<-workerState.statusThreads[c.ID]
		}
	}
	t.Log("Testing UpdateChecks with inactive -> active")
	mockDB.Checks.StatusChecks[0].Active = true
	_ = workerState.UpdateChecks()
	for i, c := range mockDB.Checks.StatusChecks {
		v, ok := workerState.statusChecks[c.ID]
		if !ok {
			t.Fatalf("mockDB checks != workerState. \nGot: %+v, \nWant: %+v\n", v, mockDB.Checks.StatusChecks[i])
		}
	}

	// next test
	/// emtpy channel buffers
	for _, c := range mockDB.Checks.StatusChecks {
		if len(workerState.statusThreads[c.ID]) > 0 {
			<-workerState.statusThreads[c.ID]
		}
	}
	t.Log("Testing UpdateChecks by adding a new check")
	mockDB.Checks.StatusChecks = append(mockDB.Checks.StatusChecks,
		checks.StatusCheck{
			ID:          "test-check-3",
			URL:         "https://ha.chacarntz.net",
			Interval:    1,
			HTTPTimeout: 5,
			Regions:     []string{"test-region-1"},
			Modified:    time.Now().UTC(),
			Serial:      0,
			Active:      true,
		})
	_ = workerState.UpdateChecks()
	for i, c := range mockDB.Checks.StatusChecks {
		v, ok := workerState.statusChecks[c.ID]
		if !ok {
			t.Fatalf("mockDB checks != workerState. \nGot: %+v, \nWant: %+v\n", v, mockDB.Checks.StatusChecks[i])
		}
	}
}

func TestSendResultsWorker(t *testing.T) {
	workerState := setupState()
	defer workerState.Log.Sync()
	mockDB := test.MockDB{}
	workerState.DBClient = &mockDB

	go workerState.sendResultsWorker(1)

	timestamp := time.Now().UTC()
	sent := &checks.StatusCheckResult{
		Metadata:      checks.StatusCheckMetadata{Region: "us-test-1", CheckID: "test-check-1"},
		Timestamp:     timestamp,
		ResponseID:    "test response id",
		ResponseCode:  200,
		TTFB:          5,
		ConnectTiming: 10,
		TLSTiming:     15,
		DNSTiming:     20,
		ResponseInfo:  "test response",
	}

	workerState.statusCheckResultCh <- sent
	time.Sleep(10 * time.Millisecond)
	rCount := len(mockDB.StatusResult)

	if rCount != 1 {
		t.Fatalf("Check results. Want: 1 Got: %d", rCount)
	}
}
