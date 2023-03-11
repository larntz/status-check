// Package checks defines our checks structs
package checks

import "time"

// Checks is a list of checks
type Checks struct {
	StatusChecks []StatusCheck
	SSLChecks    []SSLCheck
	Region       string
}

// StatusCheck defines an up/down status checks
type StatusCheck struct {
	ID          string // uuid
	URL         string
	Interval    int // seconds
	HTTPTimeout int // seconds
	Regions     []string
	Modified    time.Time
	Serial      uint64
	Active      bool
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
	ResponseCode  int                 `bson:"response_code,omitempty"`
	TTFB          int64               `bson:"firstbyte_ms,omitempty"`
	ConnectTiming int64               `bson:"connect_ms,omitempty"`
	TLSTiming     int64               `bson:"tls_ms,omitempty"`
	DNSTiming     int64               `bson:"dns_ms,omitempty"`
	ResponseInfo  string              `bson:"response_info"`
}

// SSLCheck defines an SSL check
type SSLCheck struct {
	ID       string // uuid
	URL      string
	Interval int // seconds
	Active   bool
}

// SSLCheckResult is the result of an SSLCheck
type SSLCheckResult struct {
	ID            string // uuid
	ResponseID    string // uuid
	SSLExpiration time.Time
	Valid         bool
}
