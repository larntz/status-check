package data

import (
	"github.com/larntz/status/internal/checks"
)

// Database interface abstracts database access.
type Database interface {
	Connect() error
	GetRegionChecks(region string) (checks.Checks, error)
	SendResults(results []interface{}) (string, error)
	Disconnect()
}
