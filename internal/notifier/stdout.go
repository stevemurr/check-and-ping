package notifier

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/murr/check-and-ping/internal/check"
)

// Stdout writes alerts to stdout (useful for containers/logging)
type Stdout struct {
	writer io.Writer
}

// StdoutOption configures the Stdout notifier
type StdoutOption func(*Stdout)

// WithWriter sets a custom writer (useful for testing)
func WithWriter(w io.Writer) StdoutOption {
	return func(s *Stdout) {
		s.writer = w
	}
}

// NewStdout creates a new stdout notifier
func NewStdout(opts ...StdoutOption) *Stdout {
	s := &Stdout{
		writer: os.Stdout,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Name returns the notifier name
func (s *Stdout) Name() string {
	return "stdout"
}

// Send writes the alert to stdout
func (s *Stdout) Send(ctx context.Context, alert check.Alert) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] ", alert.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("[%s] ", strings.ToUpper(alert.Priority.String())))
	sb.WriteString(fmt.Sprintf("[%s] ", alert.CheckName))
	sb.WriteString(alert.Title)

	if alert.Message != "" {
		sb.WriteString(": ")
		sb.WriteString(alert.Message)
	}

	if len(alert.Tags) > 0 {
		sb.WriteString(fmt.Sprintf(" [tags: %s]", strings.Join(alert.Tags, ", ")))
	}

	sb.WriteString("\n")

	_, err := s.writer.Write([]byte(sb.String()))
	return err
}
