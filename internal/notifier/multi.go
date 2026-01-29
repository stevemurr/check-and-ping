package notifier

import (
	"context"
	"fmt"
	"strings"

	"github.com/murr/check-and-ping/internal/check"
)

// Multi fans out alerts to multiple notifiers
type Multi struct {
	notifiers []Notifier
}

// NewMulti creates a notifier that sends to all provided notifiers
func NewMulti(notifiers ...Notifier) *Multi {
	return &Multi{
		notifiers: notifiers,
	}
}

// Name returns the names of all notifiers
func (m *Multi) Name() string {
	names := make([]string, len(m.notifiers))
	for i, n := range m.notifiers {
		names[i] = n.Name()
	}
	return "multi[" + strings.Join(names, ", ") + "]"
}

// Send sends the alert to all notifiers, collecting any errors
func (m *Multi) Send(ctx context.Context, alert check.Alert) error {
	var errs []error

	for _, n := range m.notifiers {
		if err := n.Send(ctx, alert); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", n.Name(), err))
		}
	}

	if len(errs) > 0 {
		return &MultiError{Errors: errs}
	}

	return nil
}

// Add adds a notifier to the multi-notifier
func (m *Multi) Add(n Notifier) {
	m.notifiers = append(m.notifiers, n)
}

// MultiError contains errors from multiple notifiers
type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	var sb strings.Builder
	sb.WriteString("multiple notification errors: ")
	for i, err := range e.Errors {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Error())
	}
	return sb.String()
}
