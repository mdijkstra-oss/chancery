package quota

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	ReserveURL string
	SettleURL  string
	AuthToken  string
	Timeout    time.Duration
}

type ReserveRequest struct {
	RequestID            string `json:"request_id"`
	Subject              string `json:"subject,omitempty"`
	Operation            string `json:"operation"`
	Endpoint             string `json:"endpoint"`
	Provider             string `json:"provider"`
	Model                string `json:"model"`
	ServiceTier          string `json:"service_tier,omitempty"`
	ReasoningEffort      string `json:"reasoning_effort,omitempty"`
	EstimatedInputTokens int    `json:"estimated_input_tokens,omitempty"`
	MaximumOutputTokens  int    `json:"maximum_output_tokens,omitempty"`
}

type Reservation struct {
	Allowed           bool   `json:"allowed"`
	ID                string `json:"reservation_id,omitempty"`
	Reason            string `json:"reason,omitempty"`
	RetryAfterSeconds int    `json:"retry_after_seconds,omitempty"`
}

type Usage struct {
	InputTokens       int `json:"input_tokens"`
	CachedInputTokens int `json:"cached_input_tokens,omitempty"`
	CacheWriteTokens  int `json:"cache_write_tokens,omitempty"`
	OutputTokens      int `json:"output_tokens"`
	ReasoningTokens   int `json:"reasoning_tokens,omitempty"`
	TotalTokens       int `json:"total_tokens"`
}

type Outcome string

type Settlement struct {
	ReservationID string  `json:"reservation_id"`
	Outcome       Outcome `json:"outcome"`
	Usage         *Usage  `json:"usage,omitempty"`
}

type Client struct {
	config Config
	client *http.Client
}

const (
	OutcomeCompleted Outcome = "completed"
	OutcomeFailed    Outcome = "failed"
	OutcomeCancelled Outcome = "cancelled"
)

func (c Config) Enabled() bool {
	return c.ReserveURL != "" || c.SettleURL != ""
}

func (c Config) Validate() error {
	if !c.Enabled() {
		return nil
	}
	if c.ReserveURL == "" {
		return errors.New("QUOTA_RESERVE_URL is required when quota integration is enabled")
	}
	if c.SettleURL == "" {
		return errors.New("QUOTA_SETTLE_URL is required when quota integration is enabled")
	}
	if err := validateEndpoint(c.ReserveURL); err != nil {
		return fmt.Errorf("QUOTA_RESERVE_URL: %w", err)
	}
	if err := validateEndpoint(c.SettleURL); err != nil {
		return fmt.Errorf("QUOTA_SETTLE_URL: %w", err)
	}
	if c.Timeout <= 0 {
		return errors.New("QUOTA_TIMEOUT must be greater than zero")
	}
	return nil
}

func NewClient(config Config) *Client {
	return &Client{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.config.Enabled()
}

func (c *Client) Reserve(ctx context.Context, request ReserveRequest) (Reservation, error) {
	if !c.Enabled() {
		return Reservation{Allowed: true}, nil
	}
	var reservation Reservation
	if err := c.postJSON(ctx, c.config.ReserveURL, request, &reservation); err != nil {
		return Reservation{}, fmt.Errorf("reserve quota: %w", err)
	}
	if reservation.Allowed && reservation.ID == "" {
		return Reservation{}, errors.New("reserve quota: allowed response has no reservation_id")
	}
	return reservation, nil
}

func (c *Client) Settle(ctx context.Context, settlement Settlement) error {
	if !c.Enabled() || settlement.ReservationID == "" {
		return nil
	}
	settleCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.config.Timeout)
	defer cancel()
	if err := c.postJSON(settleCtx, c.config.SettleURL, settlement, nil); err != nil {
		return fmt.Errorf("settle quota: %w", err)
	}
	return nil
}

func (c *Client) postJSON(ctx context.Context, endpoint string, payload, response any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if c.config.AuthToken != "" {
		request.Header.Set("Authorization", "Bearer "+c.config.AuthToken)
	}
	result, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer result.Body.Close()
	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(result.Body, 4096))
		return fmt.Errorf("HTTP %d: %s", result.StatusCode, strings.TrimSpace(string(message)))
	}
	if response == nil || result.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(result.Body, 64*1024)).Decode(response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func validateEndpoint(value string) error {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("must be an absolute HTTP or HTTPS URL")
	}
	if parsed.User != nil || parsed.Fragment != "" {
		return errors.New("must not contain credentials or a fragment")
	}
	return nil
}
