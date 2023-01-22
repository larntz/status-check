// Package datastructures defines our structs
package datastructures

import "time"

// Checks is a list of checks
type Checks struct {
	StatusChecks []StatusCheck
	SSLChecks    []SSLCheck
}

// StatusCheck defines an up/down status checks
type StatusCheck struct {
	ID       string // uuid
	URL      string
	Interval int // seconds
	Regions  []string
}

// StatusCheckResult is the result of a StatusCheck
type StatusCheckResult struct {
	ID           string // uuid
	ResponseID   string // uuid (this specific check)
	ResponseCode int
	ResponseTime int // milliseconds?
}

// SSLCheck defines an SSL check
type SSLCheck struct {
	ID       string // uuid
	URL      string
	Interval int // seconds
}

// SSLCheckResult is the result of an SSLCheck
type SSLCheckResult struct {
	ID            string // uuid
	ResponseID    string // uuid
	SSLExpiration time.Time
	Valid         bool
}
