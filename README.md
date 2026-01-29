# check-and-ping

A Go-based alert system that runs periodic checks and sends notifications. Checks are just Go functions—they can fetch URLs, parse PDFs, call APIs, or use Claude for intelligent analysis.

## Quick Start

```bash
# Build
go build -o checkandping ./cmd/checkandping

# Run (requires claude CLI in PATH for AI-powered checks)
./checkandping --config config.yaml
```

## Example Checks

### 1. Simple HTTP Health Check

```go
func WebsiteUpCheck(url string) check.Check {
    return check.Check{
        Name:     "website-up",
        Interval: 1 * time.Minute,
        Run: func(ctx context.Context, _ *claude.Client) (check.CheckResult, error) {
            resp, err := http.Get(url)
            if err != nil || resp.StatusCode >= 500 {
                return check.CheckResult{
                    ShouldAlert: true,
                    Title:       "Site Down",
                    Message:     fmt.Sprintf("%s is not responding", url),
                }, nil
            }
            return check.CheckResult{ShouldAlert: false}, nil
        },
    }
}
```

### 2. Price Threshold Alert

```go
func BitcoinPriceCheck(threshold float64) check.Check {
    return check.Check{
        Name:     "btc-price",
        Interval: 10 * time.Minute,
        Run: func(ctx context.Context, _ *claude.Client) (check.CheckResult, error) {
            resp, _ := http.Get("https://api.coingecko.com/api/v3/simple/price?ids=bitcoin&vs_currencies=usd")
            defer resp.Body.Close()

            var data struct { Bitcoin struct { USD float64 `json:"usd"` } `json:"bitcoin"` }
            json.NewDecoder(resp.Body).Decode(&data)

            if data.Bitcoin.USD > threshold {
                return check.CheckResult{
                    ShouldAlert: true,
                    Title:       "BTC Alert",
                    Message:     fmt.Sprintf("Bitcoin is at $%.0f", data.Bitcoin.USD),
                }, nil
            }
            return check.CheckResult{ShouldAlert: false}, nil
        },
    }
}
```

### 3. Claude-Powered PDF Analysis

```go
func CourtCaseCheck(caseNumber, pdfURL string) check.Check {
    return check.Check{
        Name:     "court-case",
        Interval: 5 * time.Minute,
        Run: func(ctx context.Context, c *claude.Client) (check.CheckResult, error) {
            // Fetch PDF
            resp, _ := http.Get(pdfURL)
            pdf, _ := io.ReadAll(resp.Body)
            resp.Body.Close()

            // Ask Claude to analyze (uses "claude -p" CLI)
            response, err := c.Analyze(ctx,
                fmt.Sprintf("Find case %s. Is 'Ready for Pickup' marked? Reply: Ready or Not Ready", caseNumber),
                pdf,
            )
            if err != nil {
                return check.CheckResult{}, err
            }

            if strings.HasPrefix(response, "Ready") {
                return check.CheckResult{
                    ShouldAlert: true,
                    Title:       "Case Ready!",
                    Message:     fmt.Sprintf("%s is ready for pickup", caseNumber),
                    Priority:    check.PriorityHigh,
                }, nil
            }
            return check.CheckResult{ShouldAlert: false}, nil
        },
    }
}
```

## Adding Checks

Edit `checks/example.go` and register your checks in `All()`:

```go
func All() []check.Check {
    return []check.Check{
        WebsiteUpCheck("https://mysite.com"),
        BitcoinPriceCheck(100000),
        CourtCaseCheck("CASE-123", "https://court.gov/cases.pdf"),
    }
}
```

## Configuration

```yaml
# config.yaml
claude:
  # cli_path: /path/to/claude  # optional

notifications:
  - type: stdout  # always logs to console

  - type: ntfy
    topic: my-alerts  # free push notifications

  - type: twilio
    account_sid: ${TWILIO_ACCOUNT_SID}
    auth_token: ${TWILIO_AUTH_TOKEN}
    from: "+1234567890"
    to: "+0987654321"

  - type: sendgrid
    api_key: ${SENDGRID_API_KEY}
    from: alerts@example.com
    to: me@example.com

state:
  type: memory  # or "sqlite" for persistence
```

## Docker

```bash
docker build -t checkandping .
docker run -v ./config.yaml:/config.yaml checkandping
```

## How It Works

- Checks run on their configured interval
- On failure, exponential backoff kicks in (up to 1 hour)
- State tracking prevents duplicate alerts for the same condition
- Claude is optional—simple checks don't need AI
