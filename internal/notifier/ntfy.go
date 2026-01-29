package notifier

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/murr/check-and-ping/internal/check"
)

const defaultNtfyServer = "https://ntfy.sh"

// Ntfy sends notifications via ntfy.sh
type Ntfy struct {
	server     string
	topic      string
	httpClient *http.Client
}

// NtfyOption configures the Ntfy notifier
type NtfyOption func(*Ntfy)

// WithNtfyServer sets the ntfy server URL
func WithNtfyServer(server string) NtfyOption {
	return func(n *Ntfy) {
		n.server = strings.TrimSuffix(server, "/")
	}
}

// WithNtfyHTTPClient sets a custom HTTP client
func WithNtfyHTTPClient(client *http.Client) NtfyOption {
	return func(n *Ntfy) {
		n.httpClient = client
	}
}

// NewNtfy creates a new ntfy.sh notifier
func NewNtfy(topic string, opts ...NtfyOption) *Ntfy {
	n := &Ntfy{
		server: defaultNtfyServer,
		topic:  topic,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(n)
	}

	return n
}

// Name returns the notifier name
func (n *Ntfy) Name() string {
	return "ntfy"
}

// Send sends an alert via ntfy.sh
func (n *Ntfy) Send(ctx context.Context, alert check.Alert) error {
	url := fmt.Sprintf("%s/%s", n.server, n.topic)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(alert.Message))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Title", alert.Title)
	req.Header.Set("Priority", ntfyPriority(alert.Priority))

	if len(alert.Tags) > 0 {
		req.Header.Set("Tags", strings.Join(alert.Tags, ","))
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy returned status %d", resp.StatusCode)
	}

	return nil
}

func ntfyPriority(p check.Priority) string {
	switch p {
	case check.PriorityLow:
		return "2"
	case check.PriorityNormal:
		return "3"
	case check.PriorityHigh:
		return "4"
	case check.PriorityUrgent:
		return "5"
	default:
		return "3"
	}
}
