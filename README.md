# fast-cli

A command-line tool to test your internet speed using fast.com.

## Installation

### From source

```bash
git clone https://github.com/DemonKingSwarn/fast-cli.git
cd fast-cli
go build -o fast .
mv fast ~/.local/bin/
```

### Using Go install

```bash
go install github.com/DemonKingSwarn/fast-cli@latest
```

## Usage

Run a speed test:

```bash
fast
```

The test will display download and upload speeds along with latency measurements.

### Options

- `--json, -j` - Output results as JSON
- `--simple, -s` - Simple output without spinner

### Examples

Basic speed test:

```bash
$ fast
download: 350.5 Mbps
upload: 280.2 Mbps

Latency: 12 ms (unloaded) / 15 ms (loaded)
```

JSON output:

```bash
$ fast --json
{
  "downloadSpeed": 350.5,
  "uploadSpeed": 280.2,
  "downloadUnit": "Mbps",
  "uploadUnit": "Mbps",
  "downloaded": 26214400,
  "uploaded": 2097152,
  "latency": 12,
  "bufferBloat": 15
}
```

## How It Works

fast-cli uses Netflix's fast.com infrastructure to measure internet speed. It performs:

1. **Latency test** - Measures round-trip time to the nearest Netflix server using small range requests
2. **Download test** - Downloads data from Netflix's content delivery servers to measure download speed
3. **Upload test** - Uploads data to measure upload speed

The test runs automatically in sequence: latency, then download, then upload.

## Requirements

- Go 1.23 or later
- Internet connection

## License

GNU General Public License v3 (GPLv3)
