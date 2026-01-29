package notifier

import (
	"context"

	"github.com/murr/check-and-ping/internal/check"
)

// Notifier is the interface for sending alerts
type Notifier interface {
	Name() string
	Send(ctx context.Context, alert check.Alert) error
}
