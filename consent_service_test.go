package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorizePaymentNeedsConsent(t *testing.T) {
	cases := []struct {
		name    string
		granted bool
		want    string
	}{{"granted", true, "payment.authorized"}, {"missing", false, "payment.consent_requested"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/v1/auth/consent/check/u-1/essential" {
					if tc.granted {
						_, _ = w.Write([]byte(`{"ok":true,"data":{"granted":true}}`))
					} else {
						_, _ = w.Write([]byte(`{"ok":true,"data":{"granted":false}}`))
					}
					return
				}
				if r.URL.Path == "/v1/auth/consent/grant/u-1" && r.Method == http.MethodPost {
					var body map[string]string
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["user_id"] != "u-1" || body["category"] != "essential" {
						t.Fatalf("invalid grant body: %#v", body)
					}
					_, _ = w.Write([]byte(`{"ok":true,"data":{}}`))
					return
				}
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}))
			defer server.Close()
			client := &ConsentClient{BaseURL: server.URL, Key: "test-key", HTTP: server.Client()}
			notice, err := authorizePayment(context.Background(), client, PaymentEvent{UserID: "u-1", Amount: 1250, Currency: "USD"})
			if err != nil {
				t.Fatal(err)
			}
			if notice.Event != tc.want {
				t.Fatalf("event = %q, want %q", notice.Event, tc.want)
			}
		})
	}
}
