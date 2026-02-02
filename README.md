# Bader IoT CLI

Command-line interface for the Bader IoT Platform.

## Installation

### Quick Install (Linux/macOS)

```bash
curl -fsSL https://api.iot.bader.solutions/api/releases/cli/install.sh | bash
```

### Homebrew (macOS/Linux)

```bash
brew install bader-solutions/tap/iot
```

### Manual Download

Download the latest binary from [Releases](https://github.com/bader-solutions/iot-cli/releases).

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux | x64 | `iot-linux-amd64` |
| Linux | ARM64 | `iot-linux-arm64` |
| macOS | Intel | `iot-darwin-amd64` |
| macOS | Apple Silicon | `iot-darwin-arm64` |
| Windows | x64 | `iot-windows-amd64.exe` |

### From Source

```bash
go install github.com/bader-solutions/iot-cli@latest
```

## Quick Start

```bash
# Authenticate (opens browser)
iot auth login

# List your devices
iot device list

# View device details
iot device get <device-id>

# Check auth status
iot auth status
```

## Commands

```
iot auth login      Authenticate with the platform
iot auth logout     Log out and clear credentials
iot auth status     Show authentication status

iot device list     List all devices
iot device get      Get device details

iot version         Show version information
```

## Configuration

Config file location:
- Linux: `~/.config/iot/config.yaml`
- macOS: `~/Library/Application Support/iot/config.yaml`
- Windows: `%APPDATA%\iot\config.yaml`

## Development

```bash
# Build
make build

# Build for all platforms
make build-all

# Run tests
make test

# Install locally
make install
```

## License

Copyright (c) Bader Solutions. All rights reserved.
