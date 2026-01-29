package claude

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Client wraps the Claude CLI for check analysis
type Client struct {
	cliPath string
}

// ClientOption configures the Client
type ClientOption func(*Client)

// WithCLIPath sets the path to the claude CLI binary
func WithCLIPath(path string) ClientOption {
	return func(c *Client) {
		c.cliPath = path
	}
}

// NewClient creates a new Claude CLI client
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		cliPath: "claude", // assume it's in PATH
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Analyze sends content to Claude CLI for analysis
// For binary content (PDF, images), it writes to a temp file and passes the path
func (c *Client) Analyze(ctx context.Context, prompt string, content []byte) (string, error) {
	// Detect if content is binary (PDF, image, etc.)
	if isBinary(content) {
		return c.analyzeFile(ctx, prompt, content)
	}

	// For text content, just include it in the prompt
	fullPrompt := string(content) + "\n\n" + prompt
	return c.runClaude(ctx, fullPrompt, "")
}

// AnalyzeText sends text content for analysis
func (c *Client) AnalyzeText(ctx context.Context, prompt, text string) (string, error) {
	fullPrompt := text + "\n\n" + prompt
	return c.runClaude(ctx, fullPrompt, "")
}

// AnalyzeFile sends a file to Claude for analysis
func (c *Client) AnalyzeFile(ctx context.Context, prompt, filePath string) (string, error) {
	return c.runClaude(ctx, prompt, filePath)
}

// analyzeFile writes binary content to a temp file and analyzes it
func (c *Client) analyzeFile(ctx context.Context, prompt string, content []byte) (string, error) {
	// Determine extension based on content
	ext := detectExtension(content)

	// Create temp file
	tmpFile, err := os.CreateTemp("", "claude-check-*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write content
	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	return c.runClaude(ctx, prompt, tmpPath)
}

// runClaude executes the claude CLI with the given prompt and optional file
func (c *Client) runClaude(ctx context.Context, prompt string, filePath string) (string, error) {
	args := []string{"-p", prompt}
	if filePath != "" {
		args = append(args, filePath)
	}

	cmd := exec.CommandContext(ctx, c.cliPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude CLI error: %w (stderr: %s)", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// AnalyzeURL fetches a URL and analyzes it (convenience method)
func (c *Client) AnalyzeURL(ctx context.Context, prompt, url string) (string, error) {
	// For URLs, we can pass the URL directly and let claude fetch it,
	// or we could fetch it ourselves. For now, assume caller fetches.
	return "", fmt.Errorf("AnalyzeURL not implemented - fetch URL first and use Analyze()")
}

// isBinary checks if content appears to be binary (not text)
func isBinary(content []byte) bool {
	if len(content) < 4 {
		return false
	}

	// Check for common binary file signatures
	// PDF
	if bytes.HasPrefix(content, []byte("%PDF")) {
		return true
	}
	// PNG
	if bytes.HasPrefix(content, []byte{0x89, 0x50, 0x4E, 0x47}) {
		return true
	}
	// JPEG
	if bytes.HasPrefix(content, []byte{0xFF, 0xD8, 0xFF}) {
		return true
	}
	// GIF
	if bytes.HasPrefix(content, []byte("GIF87a")) || bytes.HasPrefix(content, []byte("GIF89a")) {
		return true
	}
	// WebP
	if len(content) >= 12 && bytes.Equal(content[0:4], []byte("RIFF")) && bytes.Equal(content[8:12], []byte("WEBP")) {
		return true
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < min(512, len(content)); i++ {
		if content[i] == 0 {
			return true
		}
	}

	return false
}

// detectExtension returns a file extension based on content type
func detectExtension(content []byte) string {
	if len(content) < 4 {
		return ".bin"
	}

	if bytes.HasPrefix(content, []byte("%PDF")) {
		return ".pdf"
	}
	if bytes.HasPrefix(content, []byte{0x89, 0x50, 0x4E, 0x47}) {
		return ".png"
	}
	if bytes.HasPrefix(content, []byte{0xFF, 0xD8, 0xFF}) {
		return ".jpg"
	}
	if bytes.HasPrefix(content, []byte("GIF87a")) || bytes.HasPrefix(content, []byte("GIF89a")) {
		return ".gif"
	}
	if len(content) >= 12 && bytes.Equal(content[0:4], []byte("RIFF")) && bytes.Equal(content[8:12], []byte("WEBP")) {
		return ".webp"
	}

	return ".bin"
}

// min returns the smaller of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Ensure temp directory exists for file operations
func init() {
	// Ensure we can create temp files
	tmpDir := os.TempDir()
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		os.MkdirAll(tmpDir, 0755)
	}
}

// ValidateCLI checks if the claude CLI is available
func (c *Client) ValidateCLI() error {
	path, err := exec.LookPath(c.cliPath)
	if err != nil {
		return fmt.Errorf("claude CLI not found: %w", err)
	}

	// Verify it's executable
	absPath, _ := filepath.Abs(path)
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("cannot stat claude CLI: %w", err)
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("claude CLI is not executable: %s", absPath)
	}

	return nil
}
