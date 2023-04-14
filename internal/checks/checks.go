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
	Metadata      StatusCheckMetadata `json:"metadata" bson:"metadata"`
	Timestamp     time.Time           `json:"timestamp" bson:"timestamp"`
	ResponseID    string              `json:"-" bson:"-"`
	ResponseCode  int                 `json:"response_code" bson:"response_code,omitempty"`
	TTFB          int64               `json:"firstbyte_ms" bson:"firstbyte_ms,omitempty"`
	ConnectTiming int64               `json:"connect_ms" bson:"connect_ms,omitempty"`
	TLSTiming     int64               `json:"tls_ms" bson:"tls_ms,omitempty"`
	DNSTiming     int64               `json:"dns_ms" bson:"dns_ms,omitempty"`
	ResponseInfo  string              `json:"response_info" bson:"response_info"`
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
