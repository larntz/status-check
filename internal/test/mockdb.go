// Package test is used for unit tests
package test

import (
	"fmt"

	"github.com/larntz/status/internal/checks"
)

// MockDB is a mock database used for testing
type MockDB struct {
	Checks       checks.Checks
	StatusResult []checks.StatusCheckResult
}

// Connect to the MockDB
func (db *MockDB) Connect() error {
	return nil
}

// GetRegionChecks gets mock region checks
func (db MockDB) GetRegionChecks(_ string) (checks.Checks, error) {
	return db.Checks, nil
}

// SendResults to the MockDB
func (db *MockDB) SendResults(results []interface{}) (string, error) {
	added := 0
	for _, result := range results {
		r, ok := result.(checks.StatusCheckResult)
		if ok {
			db.StatusResult = append(db.StatusResult, r)
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
	db.Checks.StatusChecks = append(db.Checks.StatusChecks, check)
}
