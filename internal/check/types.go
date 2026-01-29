package check

import (
	"context"
	"time"

	"github.com/murr/check-and-ping/internal/claude"
)

// Priority indicates the urgency level of an alert
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityNormal:
		return "normal"
	case PriorityHigh:
		return "high"
	case PriorityUrgent:
		return "urgent"
	default:
		return "normal"
	}
}

// CheckResult represents the outcome of a check
type CheckResult struct {
	ShouldAlert bool
	Title       string
	Message     string
	Priority    Priority
	Tags        []string
	Metadata    map[string]string
}

// CheckFunc is the signature for user-defined checks.
// The claude parameter is optional - checks that don't need AI analysis can ignore it.
type CheckFunc func(ctx context.Context, claude *claude.Client) (CheckResult, error)

// Check wraps a CheckFunc with scheduling metadata
type Check struct {
	Name     string
	Interval time.Duration
	Run      CheckFunc
}

// Alert represents a notification to be sent
type Alert struct {
	CheckName string
	Title     string
	Message   string
	Priority  Priority
	Tags      []string
	Metadata  map[string]string
	Timestamp time.Time
}

// NewAlertFromResult creates an Alert from a CheckResult
func NewAlertFromResult(checkName string, result CheckResult) Alert {
	return Alert{
		CheckName: checkName,
		Title:     result.Title,
		Message:   result.Message,
		Priority:  result.Priority,
		Tags:      result.Tags,
		Metadata:  result.Metadata,
		Timestamp: time.Now(),
	}
}
