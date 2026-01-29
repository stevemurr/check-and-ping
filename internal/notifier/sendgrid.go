package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/murr/check-and-ping/internal/check"
)

// SendGrid sends email notifications via SendGrid
type SendGrid struct {
	apiKey     string
	from       string
	fromName   string
	to         string
	httpClient *http.Client
}

// SendGridOption configures the SendGrid notifier
type SendGridOption func(*SendGrid)

// WithSendGridFromName sets the sender display name
func WithSendGridFromName(name string) SendGridOption {
	return func(s *SendGrid) {
		s.fromName = name
	}
}

// WithSendGridHTTPClient sets a custom HTTP client
func WithSendGridHTTPClient(client *http.Client) SendGridOption {
	return func(s *SendGrid) {
		s.httpClient = client
	}
}

// NewSendGrid creates a new SendGrid email notifier
func NewSendGrid(apiKey, from, to string, opts ...SendGridOption) *SendGrid {
	s := &SendGrid{
		apiKey:   apiKey,
		from:     from,
		fromName: "Check-and-Ping Alerts",
		to:       to,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Name returns the notifier name
func (s *SendGrid) Name() string {
	return "sendgrid"
}

// Send sends an email via SendGrid
func (s *SendGrid) Send(ctx context.Context, alert check.Alert) error {
	payload := map[string]any{
		"personalizations": []map[string]any{
			{
				"to": []map[string]string{
					{"email": s.to},
				},
			},
		},
		"from": map[string]string{
			"email": s.from,
			"name":  s.fromName,
		},
		"subject": fmt.Sprintf("[%s] %s", alert.CheckName, alert.Title),
		"content": []map[string]string{
			{
				"type":  "text/plain",
				"value": formatEmailBody(alert),
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	defer resp.Body.Close()

	// SendGrid returns 202 Accepted on success
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sendgrid returned status %d", resp.StatusCode)
	}

	return nil
}

func formatEmailBody(alert check.Alert) string {
	body := fmt.Sprintf("Check: %s\nPriority: %s\nTime: %s\n\n%s",
		alert.CheckName,
		alert.Priority.String(),
		alert.Timestamp.Format(time.RFC1123),
		alert.Message,
	)

	if len(alert.Metadata) > 0 {
		body += "\n\nMetadata:\n"
		for k, v := range alert.Metadata {
			body += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}

	return body
}
