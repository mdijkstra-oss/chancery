package quota

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError string
	}{
		{name: "disabled"},
		{name: "enabled", config: Config{ReserveURL: "https://quota.example/reserve", SettleURL: "https://quota.example/settle", Timeout: time.Second}},
		{name: "missing reserve", config: Config{SettleURL: "https://quota.example/settle", Timeout: time.Second}, wantError: "QUOTA_RESERVE_URL"},
		{name: "missing settle", config: Config{ReserveURL: "https://quota.example/reserve", Timeout: time.Second}, wantError: "QUOTA_SETTLE_URL"},
		{name: "invalid reserve", config: Config{ReserveURL: "not-a-url", SettleURL: "https://quota.example/settle", Timeout: time.Second}, wantError: "absolute HTTP or HTTPS URL"},
		{name: "zero timeout", config: Config{ReserveURL: "https://quota.example/reserve", SettleURL: "https://quota.example/settle"}, wantError: "QUOTA_TIMEOUT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()
			if test.wantError == "" && err != nil {
				t.Fatalf("Validate(): %v", err)
			}
			if test.wantError != "" && (err == nil || !strings.Contains(err.Error(), test.wantError)) {
				t.Fatalf("Validate() error = %v, want containing %q", err, test.wantError)
			}
		})
	}
}

func TestClientReserveAndSettle(t *testing.T) {
	var gotReserve ReserveRequest
	var gotSettlement Settlement
	var gotAuthorization []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = append(gotAuthorization, r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/reserve":
			if err := json.NewDecoder(r.Body).Decode(&gotReserve); err != nil {
				t.Errorf("decode reserve: %v", err)
			}
			json.NewEncoder(w).Encode(Reservation{Allowed: true, ID: "res-123"})
		case "/settle":
			if err := json.NewDecoder(r.Body).Decode(&gotSettlement); err != nil {
				t.Errorf("decode settlement: %v", err)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(Config{
		ReserveURL: server.URL + "/reserve",
		SettleURL:  server.URL + "/settle",
		AuthToken:  "quota-secret",
		Timeout:    time.Second,
	})
	request := ReserveRequest{RequestID: "req-123", Subject: "user-456", Operation: "chat", Model: "model-a"}
	reservation, err := client.Reserve(context.Background(), request)
	if err != nil {
		t.Fatalf("Reserve(): %v", err)
	}
	settlement := Settlement{ReservationID: reservation.ID, Outcome: OutcomeCompleted, Usage: &Usage{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}}
	if err := client.Settle(context.Background(), settlement); err != nil {
		t.Fatalf("Settle(): %v", err)
	}

	if gotReserve != request {
		t.Errorf("reserve = %#v, want %#v", gotReserve, request)
	}
	if gotSettlement.ReservationID != settlement.ReservationID || gotSettlement.Outcome != settlement.Outcome || gotSettlement.Usage == nil || *gotSettlement.Usage != *settlement.Usage {
		t.Errorf("settlement = %#v, want %#v", gotSettlement, settlement)
	}
	for _, authorization := range gotAuthorization {
		if authorization != "Bearer quota-secret" {
			t.Errorf("Authorization = %q", authorization)
		}
	}
}

func TestClientReserveResponses(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		want      Reservation
		wantError string
	}{
		{name: "denied", status: http.StatusOK, body: `{"allowed":false,"reason":"daily_limit"}`, want: Reservation{Allowed: false, Reason: "daily_limit"}},
		{name: "allowed without id", status: http.StatusOK, body: `{"allowed":true}`, wantError: "no reservation_id"},
		{name: "service error", status: http.StatusInternalServerError, body: `failed`, wantError: "HTTP 500"},
		{name: "invalid response", status: http.StatusOK, body: `{`, wantError: "decode response"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				w.Write([]byte(test.body))
			}))
			defer server.Close()
			client := NewClient(Config{ReserveURL: server.URL, SettleURL: server.URL, Timeout: time.Second})
			got, err := client.Reserve(context.Background(), ReserveRequest{})
			if test.wantError == "" && err != nil {
				t.Fatalf("Reserve(): %v", err)
			}
			if test.wantError != "" && (err == nil || !strings.Contains(err.Error(), test.wantError)) {
				t.Fatalf("Reserve() error = %v, want containing %q", err, test.wantError)
			}
			if test.wantError == "" && got != test.want {
				t.Errorf("Reserve() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestDisabledClient(t *testing.T) {
	client := NewClient(Config{})
	reservation, err := client.Reserve(context.Background(), ReserveRequest{})
	if err != nil {
		t.Fatalf("Reserve(): %v", err)
	}
	if !reservation.Allowed || reservation.ID != "" {
		t.Errorf("reservation = %#v", reservation)
	}
	if err := client.Settle(context.Background(), Settlement{}); err != nil {
		t.Fatalf("Settle(): %v", err)
	}
}
