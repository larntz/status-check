package test

import "net/http"

// HTTPTransport used for mocking http.Transport.RoundTrip
type HTTPTransport struct {
	Response *http.Response
}

// RoundTrip mocks http.Transport.RoundTrip
func (h *HTTPTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return h.Response, nil
}

// Body mocks a response body
type Body struct{}

// Close the body
func (b *Body) Close() error {
	return nil
}

// Read something
func (b *Body) Read(_ []byte) (n int, err error) {
	return 0, nil
}
