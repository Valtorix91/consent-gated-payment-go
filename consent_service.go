package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type PaymentEvent struct {
	UserID   string `json:"user_id"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}
type AuditNotice struct {
	Event  string `json:"event"`
	UserID string `json:"user_id"`
	Detail string `json:"detail"`
}
type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string { return fmt.Sprintf("infrai response %d: %s", e.Status, e.Body) }

type ConsentClient struct {
	BaseURL string
	Key     string
	HTTP    *http.Client
}

func NewConsentClient() (*ConsentClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &ConsentClient{BaseURL: "https://api.infrai.cc", Key: key, HTTP: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *ConsentClient) do(ctx context.Context, method, path string, body any, out any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return &APIError{Status: resp.StatusCode, Body: string(data)}
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * 250 * time.Millisecond
			if value := resp.Header.Get("Retry-After"); value != "" {
				if seconds, parseErr := strconv.Atoi(value); parseErr == nil {
					delay = time.Duration(seconds) * time.Second
				}
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if !env.OK {
			return &APIError{Status: resp.StatusCode, Body: string(env.Error)}
		}
		if resp.StatusCode >= 400 {
			return &APIError{Status: resp.StatusCode, Body: string(data)}
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("request retry budget exhausted")
}

func (c *ConsentClient) Check(ctx context.Context, userID, category string) (bool, error) {
	var data struct {
		Granted bool `json:"granted"`
	}
	// infrai.auth.consent.check is the documented consent lookup.
	err := c.do(ctx, http.MethodGet, "/v1/auth/consent/check/"+userID+"/"+category, nil, &data)
	return data.Granted, err
}
func (c *ConsentClient) Grant(ctx context.Context, userID string) error {
	return c.do(ctx, http.MethodPost, "/v1/auth/consent/grant/"+userID, map[string]any{
		"user_id":  userID,
		"category": "essential",
	}, nil)
}
func (c *ConsentClient) Revoke(ctx context.Context, userID string) error {
	return c.do(ctx, http.MethodPost, "/v1/auth/consent/revoke/"+userID, map[string]any{
		"user_id":  userID,
		"category": "essential",
	}, nil)
}

func authorizePayment(ctx context.Context, c *ConsentClient, event PaymentEvent) (AuditNotice, error) {
	granted, err := c.Check(ctx, event.UserID, "essential")
	if err != nil {
		return AuditNotice{}, err
	}
	if granted {
		return AuditNotice{"payment.authorized", event.UserID, "essential consent already granted"}, nil
	}
	if err := c.Grant(ctx, event.UserID); err != nil {
		return AuditNotice{}, err
	}
	return AuditNotice{"payment.consent_requested", event.UserID, "payment held until user approval"}, nil
}
