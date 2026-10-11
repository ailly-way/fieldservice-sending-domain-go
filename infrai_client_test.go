package main

import (
	"io"
	"net/http"
	"strings"
)

type staticTransport struct{}

func (*staticTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 400, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"ok":false,"error":{"code":"INVALID_ARGUMENT","message":"invalid domain"}}`)), Request: req}, nil
}
func (s *staticTransport) client() *http.Client { return &http.Client{Transport: s} }
