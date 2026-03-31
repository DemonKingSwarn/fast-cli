# AGENTS.md

This file provides guidance for AI agents working on this project.

## Project Overview

**fast-cli** is a Go CLI tool that tests internet speed using Netflix's fast.com infrastructure. It measures download speed, upload speed, and latency.

## Project Structure

```
.
├── main.go          # CLI entry point (cobra command)
├── core/fast.go     # Core speed test logic
├── go.mod           # Go module definition
├── builds/          # Compiled binaries for various platforms
└── justfile         # Build scripts
```

## Key Commands

### Build
```bash
go build -o fast-cli .
```

### Run
```bash
./fast-cli
./fast-cli --json    # JSON output
./fast-cli --simple  # No spinner
```

### Run Tests
```bash
go test ./...
```

### Lint/Format
```bash
go fmt ./...
go vet ./...
```

## Code Conventions

- **CLI framework**: spf13/cobra
- **HTTP client**: standard library `net/http`
- **Output**: Custom spinner for progress, JSON or text format
- **Error handling**: Return errors with `fmt.Errorf` using `%w` wrapping

## Key Functions

| Function | Purpose |
|----------|---------|
| `RunTest()` | Main test orchestrator |
| `getFastToken()` | Fetches authentication token from fast.com |
| `getDownloadURLs()` | Gets server URLs for testing |
| `measureDownload()` | Measures download speed |
| `measureUpload()` | Measures upload speed |
| `measureLatency()` | Measures latency (5 requests, returns median) |

## Development Notes

- Latency is measured before download, then used as initial buffer bloat value
- Download test runs before upload test
- The spinner only runs in TTY mode (checks `TERM` env var)
- Latency calculation divides by 2 (round-trip to one-way)
- Upload uses 2MB of random data
