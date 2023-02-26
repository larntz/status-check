// Package checks defines our checks structs
package checks

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

// StatusCheckMetadata models our timeseries metadata
type StatusCheckMetadata struct {
	Region  string `bson:"region"`
	CheckID string `bson:"check_id"`
}

// StatusCheckResult is the result of a StatusCheck
type StatusCheckResult struct {
	Metadata      StatusCheckMetadata `bson:"metadata"`
	Timestamp     time.Time           `bson:"timestamp"`
	ResponseID    string              `bson:"-"`
	ResponseCode  int                 `bson:"response_code"`
	TTFB          int64               `bson:"firstbyte_ms"`
	ConnectTiming int64               `bson:"connect_ms"`
	TLSTiming     int64               `bson:"tls_ms"`
	DNSTiming     int64               `bson:"dns_ms"`
	ResponseInfo  string              `bson:"response_info"`
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
