package checks

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/murr/check-and-ping/internal/check"
	"github.com/murr/check-and-ping/internal/claude"
)

// All returns all registered checks
// Modify this function to add or remove checks
func All() []check.Check {
	return []check.Check{
		// Example: Court case PDF check (Claude-powered)
		// Example: CourtCaseCheck("CASE-123456", "https://court.example.gov/cases.pdf"),

		// Example: Simple HTTP check
		// WebsiteUpCheck("https://example.com"),

		// Example: Bitcoin price check
		// BitcoinPriceCheck(100000),
	}
}

// CourtCaseCheck monitors a court PDF for case status
func CourtCaseCheck(caseNumber, pdfURL string) check.Check {
	return check.Check{
		Name:     "court-case-" + strings.ReplaceAll(caseNumber, " ", "-"),
		Interval: 5 * time.Minute,
		Run: func(ctx context.Context, c *claude.Client) (check.CheckResult, error) {
			if c == nil {
				return check.CheckResult{}, fmt.Errorf("claude client required for this check")
			}

			// Fetch PDF
			pdf, err := fetchURL(ctx, pdfURL)
			if err != nil {
				return check.CheckResult{}, fmt.Errorf("fetch PDF: %w", err)
			}

			// Ask Claude to analyze
			prompt := fmt.Sprintf(
				"Find case %s in this PDF. Look at the 'Ready for Pickup' column. "+
					"If there is an X in that column for this case, respond with exactly: Ready. "+
					"If there is no X, or the case is not listed, respond with exactly: Not Ready. "+
					"Say nothing else.",
				caseNumber,
			)

			response, err := c.Analyze(ctx, prompt, pdf)
			if err != nil {
				return check.CheckResult{}, fmt.Errorf("claude analysis: %w", err)
			}

			response = strings.TrimSpace(response)

			if strings.HasPrefix(response, "Ready") {
				return check.CheckResult{
					ShouldAlert: true,
					Title:       "Case Ready!",
					Message:     fmt.Sprintf("%s is ready for pickup", caseNumber),
					Priority:    check.PriorityHigh,
					Tags:        []string{"court", "urgent"},
				}, nil
			}

			return check.CheckResult{ShouldAlert: false}, nil
		},
	}
}

// WebsiteUpCheck monitors website availability
func WebsiteUpCheck(url string) check.Check {
	// Create a safe name from URL
	name := strings.ReplaceAll(url, "https://", "")
	name = strings.ReplaceAll(name, "http://", "")
	name = strings.ReplaceAll(name, "/", "-")

	return check.Check{
		Name:     "website-" + name,
		Interval: 1 * time.Minute,
		Run: func(ctx context.Context, _ *claude.Client) (check.CheckResult, error) {
			client := &http.Client{Timeout: 10 * time.Second}

			req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return check.CheckResult{}, err
			}

			resp, err := client.Do(req)
			if err != nil {
				return check.CheckResult{
					ShouldAlert: true,
					Title:       "Site Down",
					Message:     fmt.Sprintf("%s is not responding: %v", url, err),
					Priority:    check.PriorityHigh,
				}, nil
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 500 {
				return check.CheckResult{
					ShouldAlert: true,
					Title:       "Site Error",
					Message:     fmt.Sprintf("%s returned status %d", url, resp.StatusCode),
					Priority:    check.PriorityHigh,
				}, nil
			}

			return check.CheckResult{ShouldAlert: false}, nil
		},
	}
}

// BitcoinPriceCheck monitors Bitcoin price against a threshold
func BitcoinPriceCheck(threshold float64) check.Check {
	return check.Check{
		Name:     "btc-price",
		Interval: 10 * time.Minute,
		Run: func(ctx context.Context, _ *claude.Client) (check.CheckResult, error) {
			price, err := fetchBTCPrice(ctx)
			if err != nil {
				return check.CheckResult{}, err
			}

			if price > threshold {
				return check.CheckResult{
					ShouldAlert: true,
					Title:       "BTC Alert",
					Message:     fmt.Sprintf("Bitcoin is at $%.2f (above $%.2f)", price, threshold),
					Priority:    check.PriorityNormal,
					Tags:        []string{"crypto", "btc"},
					Metadata: map[string]string{
						"price":     fmt.Sprintf("%.2f", price),
						"threshold": fmt.Sprintf("%.2f", threshold),
					},
				}, nil
			}

			return check.CheckResult{ShouldAlert: false}, nil
		},
	}
}

// fetchURL fetches content from a URL
func fetchURL(ctx context.Context, url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// fetchBTCPrice fetches current BTC price from CoinGecko
func fetchBTCPrice(ctx context.Context) (float64, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd", nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Simple parsing - in production you'd use proper JSON decoding
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// Parse {"bitcoin":{"usd":12345.67}}
	var price float64
	_, err = fmt.Sscanf(string(body), `{"bitcoin":{"usd":%f}}`, &price)
	if err != nil {
		return 0, fmt.Errorf("parse price: %w", err)
	}

	return price, nil
}
