// Package test is used for unit tests
package test

import (
	"time"

	"github.com/larntz/status/internal/checks"
)

// MockDB is a mock database used for testing
type MockDB struct {
	checks checks.Checks
}

// Connect to the MockDB
func (db *MockDB) Connect() error {
	// add some checks
	db.checks.StatusChecks = append(db.checks.StatusChecks, checks.StatusCheck{
		ID:          "test-check-1",
		URL:         "https://test-check-1.local",
		Interval:    300,
		HTTPTimeout: 5,
		Regions:     []string{"test-region-1", "test-region-2"},
		Modified:    time.Now().UTC(),
		Serial:      0,
		Active:      true,
	})
	db.checks.StatusChecks = append(db.checks.StatusChecks, checks.StatusCheck{
		ID:          "test-check-2",
		URL:         "https://test-check-2.local",
		Interval:    900,
		HTTPTimeout: 60,
		Regions:     []string{"test-region-1", "test-region-2"},
		Modified:    time.Now().UTC(),
		Serial:      0,
		Active:      false,
	})
	return nil
}

// GetRegionChecks gets mock region checks
func (db MockDB) GetRegionChecks() (checks.Checks, error) {
	return checks.Checks{}, nil
}

// SendResults to the MockDB
func (db MockDB) SendResults([]interface{}) (string, error) {
	return "", nil
}

// Disconnect from the MockDB
func (db MockDB) Disconnect() {}

// AddCheck to MockDB
func (db *MockDB) AddCheck(check checks.StatusCheck) {
	db.checks.StatusChecks = append(db.checks.StatusChecks, check)
}
