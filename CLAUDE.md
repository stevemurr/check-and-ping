# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/claude-code) when working on this repository.

## Project Overview

check-and-ping is a Go-based periodic check and alert system. Users define checks as Go functions that return whether to send an alert. The system handles scheduling, state tracking (to avoid duplicate alerts), and notification delivery.

## Architecture

```
cmd/checkandping/main.go     # Entry point, wires components together
checks/example.go            # User-defined checks live here
internal/
  check/types.go             # Core types: Check, CheckResult, CheckFunc, Alert
  claude/client.go           # Wraps "claude -p" CLI for AI analysis
  config/config.go           # YAML config loading with ${ENV_VAR} expansion
  notifier/                  # Notification implementations
    notifier.go              # Interface
    stdout.go, ntfy.go       # Implementations
    twilio.go, sendgrid.go
    multi.go                 # Fan-out to multiple notifiers
  scheduler/scheduler.go     # Runs checks on intervals with backoff
  state/                     # Tracks alert state to prevent duplicates
    state.go                 # Interface + in-memory implementation
    sqlite.go                # Persistent implementation
```

## Key Design Decisions

1. **Checks are compiled Go code** - Adding a check requires editing `checks/example.go` and rebuilding. This keeps things type-safe and simple.

2. **Claude via CLI, not API** - Uses `claude -p` shell command instead of API. Simpler for users with Claude Max subscriptions.

3. **Claude is optional** - The `*claude.Client` parameter can be ignored for checks that don't need AI analysis.

4. **Exponential backoff on failures** - Failed checks wait longer before retrying (up to 1 hour max).

5. **State prevents duplicate alerts** - Same alert won't fire twice until the condition clears.

## Common Tasks

### Adding a new check

Edit `checks/example.go`:

```go
func MyCheck() check.Check {
    return check.Check{
        Name:     "my-check",
        Interval: 5 * time.Minute,
        Run: func(ctx context.Context, c *claude.Client) (check.CheckResult, error) {
            // Your logic here
            if somethingWrong {
                return check.CheckResult{
                    ShouldAlert: true,
                    Title:       "Alert Title",
                    Message:     "Details here",
                }, nil
            }
            return check.CheckResult{ShouldAlert: false}, nil
        },
    }
}
```

Then register it in `All()`.

### Adding a new notifier

1. Create `internal/notifier/mynotifier.go`
2. Implement the `Notifier` interface (Name, Send methods)
3. Add config struct fields to `internal/config/config.go`
4. Wire it up in `cmd/checkandping/main.go` in `buildNotifiers()`

### Building and running

```bash
go build -o checkandping ./cmd/checkandping
./checkandping --config config.yaml
```

## Dependencies

- `gopkg.in/yaml.v3` - YAML config parsing
- `github.com/mattn/go-sqlite3` - SQLite state persistence (requires CGO)
- Claude CLI (`claude`) - For AI-powered checks
