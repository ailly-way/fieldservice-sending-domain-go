package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type recordedCall struct {
	method, path string
	body         any
}
type stubAPI struct {
	status string
	calls  []recordedCall
}

func (s *stubAPI) call(_ context.Context, method, path string, body any, out any, _ string) error {
	s.calls = append(s.calls, recordedCall{method, path, body})
	var data []byte
	switch path {
	case "/v1/dns/domain/get?domain=crew.example":
		data = []byte(`{"zone_id":"zone-17"}`)
	case "/v1/email/domain/verify":
		data, _ = json.Marshal(map[string]any{"verification": map[string]string{"status": s.status}})
	case "/v1/email/send":
		data = []byte(`{"message_id":"msg-42"}`)
	}
	if out != nil && data != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

func TestDispatchVerificationBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, status string
		wantCalls    int
		wantMessage  string
	}{
		{"verified sends", "verified", 5, "msg-42"},
		{"pending holds mail", "pending", 4, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &stubAPI{status: tc.status}
			in := dispatchRequest{Domain: "crew.example", From: "updates@crew.example", To: "owner@example.org", WorkOrderID: "WO-21", Status: "dispatched", TechnicianFollowUp: "Return Tuesday", PhotoURLs: []string{"https://crew.example/photo/21"}, SPF: "v=spf1 include:example.org ~all", DKIMName: "selector._domainkey.crew.example", DKIMContent: "v=DKIM1; p=abc"}
			got, err := dispatch(context.Background(), api, in)
			if err != nil {
				t.Fatal(err)
			}
			if got.ZoneID != "zone-17" || got.MessageID != tc.wantMessage || len(api.calls) != tc.wantCalls {
				t.Fatalf("result=%+v calls=%+v", got, api.calls)
			}
			for _, call := range api.calls[1:3] {
				fields := call.body.(map[string]any)
				if call.method != "PUT" || fields["zone_id"] != got.ZoneID || fields["record_type"] != "TXT" {
					t.Fatalf("DNS boundary: %+v", call)
				}
			}
		})
	}
}

func TestAPIEnvelopeBusinessRejection(t *testing.T) {
	// A 4xx envelope remains a typed business rejection for the HTTP handler.
	server := &staticTransport{}
	client := &apiClient{key: "test-key", base: "http://example.test", http: server.client()}
	err := client.call(context.Background(), "POST", "/v1/email/domain/verify", map[string]string{"domain": "crew.example"}, nil, "request-1")
	var remote *apiError
	if !errors.As(err, &remote) || remote.Code != "INVALID_ARGUMENT" {
		t.Fatalf("got %v", err)
	}
}
