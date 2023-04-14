package worker

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
	"time"
)

// RequestTrace is used for http connection tracing
type RequestTrace struct {
	start             time.Time
	connStart         time.Time
	ConnDur           time.Duration
	DNSInfo           httptrace.DNSDoneInfo
	dnsStart          time.Time
	DNSDur            time.Duration
	tlsHandshakeStart time.Time
	TLSHandshakeDur   time.Duration
	TTFB              time.Duration
	ConnInfo          httptrace.GotConnInfo
	Trace             *httptrace.ClientTrace
}

// TraceRequest performs a request and saves tracing data
func (a *RequestTrace) TraceRequest(ctx context.Context,
	client http.RoundTripper, req *http.Request) (*http.Response, error) {
	a.start = time.Now().UTC()
	req = req.WithContext(httptrace.WithClientTrace(ctx, a.Trace))
	resp, err := client.RoundTrip(req)
	return resp, err
}

// Reset to zero values
func (a *RequestTrace) Reset() {
}

// NewRequestTrace returns a nice new trace
func NewRequestTrace() *RequestTrace {
	r := RequestTrace{}
	r.Trace = &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) { r.dnsStart = time.Now() },
		DNSDone: func(d httptrace.DNSDoneInfo) {
			r.DNSInfo = d
			r.DNSDur = time.Since(r.dnsStart)
		},
		TLSHandshakeStart: func() { r.tlsHandshakeStart = time.Now() },
		TLSHandshakeDone: func(c tls.ConnectionState, err error) {
			r.TLSHandshakeDur = time.Since(r.tlsHandshakeStart)
		},
		ConnectStart: func(network, addr string) { r.connStart = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			r.ConnDur = time.Since(r.connStart)
		},
		GotFirstResponseByte: func() {
			r.TTFB = time.Since(r.start)
		},
		GotConn: func(c httptrace.GotConnInfo) {
			r.ConnInfo = c
		},
	}
	return &r
}
