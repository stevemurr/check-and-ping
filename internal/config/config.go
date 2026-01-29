package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure
type Config struct {
	Claude        ClaudeConfig        `yaml:"claude"`
	Notifications []NotificationConfig `yaml:"notifications"`
	State         StateConfig          `yaml:"state"`
}

// ClaudeConfig configures the Claude CLI client
type ClaudeConfig struct {
	Disabled bool   `yaml:"disabled"` // Set to true to disable Claude entirely
	CLIPath  string `yaml:"cli_path"` // Path to claude CLI (defaults to "claude" in PATH)
}

// NotificationConfig configures a notification channel
type NotificationConfig struct {
	Type string `yaml:"type"`

	// ntfy.sh options
	Topic  string `yaml:"topic,omitempty"`
	Server string `yaml:"server,omitempty"`

	// Twilio options
	AccountSID string `yaml:"account_sid,omitempty"`
	AuthToken  string `yaml:"auth_token,omitempty"`
	From       string `yaml:"from,omitempty"`
	To         string `yaml:"to,omitempty"`

	// SendGrid options
	APIKey   string `yaml:"api_key,omitempty"`
	FromName string `yaml:"from_name,omitempty"`
}

// StateConfig configures state persistence
type StateConfig struct {
	Type   string `yaml:"type"` // "memory" or "sqlite"
	DBPath string `yaml:"db_path,omitempty"`
}

// Load reads and parses a config file, expanding environment variables
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Expand environment variables
	expanded := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Set defaults
	if cfg.State.Type == "" {
		cfg.State.Type = "memory"
	}

	return &cfg, nil
}

// envVarPattern matches ${VAR_NAME} patterns
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// expandEnvVars replaces ${VAR_NAME} with environment variable values
func expandEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR_NAME}
		varName := strings.TrimPrefix(strings.TrimSuffix(match, "}"), "${")
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match // Keep original if not found
	})
}

// Validate checks the config for required fields
func (c *Config) Validate() error {
	// Claude API key is optional (checks might not use Claude)
	// Notifications are optional (might just use stdout)

	// Validate state config
	switch c.State.Type {
	case "memory", "sqlite":
		// OK
	case "":
		c.State.Type = "memory"
	default:
		return fmt.Errorf("unknown state type: %s", c.State.Type)
	}

	if c.State.Type == "sqlite" && c.State.DBPath == "" {
		return fmt.Errorf("sqlite state requires db_path")
	}

	// Validate notification configs
	for i, n := range c.Notifications {
		switch n.Type {
		case "stdout":
			// No validation needed
		case "ntfy":
			if n.Topic == "" {
				return fmt.Errorf("notification[%d]: ntfy requires topic", i)
			}
		case "twilio":
			if n.AccountSID == "" || n.AuthToken == "" || n.From == "" || n.To == "" {
				return fmt.Errorf("notification[%d]: twilio requires account_sid, auth_token, from, and to", i)
			}
		case "sendgrid":
			if n.APIKey == "" || n.From == "" || n.To == "" {
				return fmt.Errorf("notification[%d]: sendgrid requires api_key, from, and to", i)
			}
		default:
			return fmt.Errorf("notification[%d]: unknown type: %s", i, n.Type)
		}
	}

	return nil
}
