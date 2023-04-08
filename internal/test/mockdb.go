// Package test is used for unit tests
package test

import (
	"fmt"
	"time"

	"github.com/larntz/status/internal/checks"
)

// MockDB is a mock database used for testing
type MockDB struct {
	checks       checks.Checks
	statusResult []checks.StatusCheckResult
}

// Connect to the MockDB
func (db *MockDB) Connect() error {
	// add some checks
	db.checks.StatusChecks = append(db.checks.StatusChecks, checks.StatusCheck{
		ID:          "test-check-1",
		URL:         "https://gitea.chacarntz.net",
		Interval:    10,
		HTTPTimeout: 5,
		Regions:     []string{"test-region-1", "test-region-2"},
		Modified:    time.Now().UTC(),
		Serial:      0,
		Active:      true,
	})
	db.checks.StatusChecks = append(db.checks.StatusChecks, checks.StatusCheck{
		ID:          "test-check-2",
		URL:         "https://blue42.net",
		Interval:    5,
		HTTPTimeout: 15,
		Regions:     []string{"test-region-1", "test-region-2"},
		Modified:    time.Now().UTC(),
		Serial:      0,
		Active:      true,
	})
	return nil
}

// GetRegionChecks gets mock region checks
func (db MockDB) GetRegionChecks(_ string) (checks.Checks, error) {
	return db.checks, nil
}

// SendResults to the MockDB
func (db *MockDB) SendResults(results []interface{}) (string, error) {
	added := 0
	for _, result := range results {
		r, ok := result.(checks.StatusCheckResult)
		if ok {
			db.statusResult = append(db.statusResult, r)
			added++
		}
	}
	summary := fmt.Sprintf("successfully inserted %d items", len(results))
	return summary, nil
}

// Disconnect from the MockDB to satisfy interface
func (db MockDB) Disconnect() {}

// AddCheck to MockDB
func (db *MockDB) AddCheck(check checks.StatusCheck) {
	db.checks.StatusChecks = append(db.checks.StatusChecks, check)
}
