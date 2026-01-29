package notifier

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/murr/check-and-ping/internal/check"
)

// Twilio sends SMS notifications via Twilio
type Twilio struct {
	accountSID string
	authToken  string
	from       string
	to         string
	httpClient *http.Client
}

// TwilioOption configures the Twilio notifier
type TwilioOption func(*Twilio)

// WithTwilioHTTPClient sets a custom HTTP client
func WithTwilioHTTPClient(client *http.Client) TwilioOption {
	return func(t *Twilio) {
		t.httpClient = client
	}
}

// NewTwilio creates a new Twilio SMS notifier
func NewTwilio(accountSID, authToken, from, to string, opts ...TwilioOption) *Twilio {
	t := &Twilio{
		accountSID: accountSID,
		authToken:  authToken,
		from:       from,
		to:         to,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Name returns the notifier name
func (t *Twilio) Name() string {
	return "twilio"
}

// Send sends an SMS via Twilio
func (t *Twilio) Send(ctx context.Context, alert check.Alert) error {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)

	// Build message body
	body := alert.Title
	if alert.Message != "" {
		body += ": " + alert.Message
	}

	// Truncate if too long for SMS
	if len(body) > 1600 {
		body = body[:1597] + "..."
	}

	data := url.Values{}
	data.Set("To", t.to)
	data.Set("From", t.from)
	data.Set("Body", body)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.SetBasicAuth(t.accountSID, t.authToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("twilio returned status %d", resp.StatusCode)
	}

	return nil
}
