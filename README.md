# BadgerMaps CLI

A command line interface for interacting with the BadgerMaps API.

## Overview

BadgerMaps CLI allows you to push and pull data, run in server mode, and perform various utility operations with the BadgerMaps API. It provides a powerful interface for managing customer accounts, routes, check-ins, and user profiles.

## Installation

```bash
# Clone the repository
git clone https://github.com/vamuscari/BadgerMaps_CLI.git

# Build the application
cd BadgerMaps_CLI
go build -o badgermaps
```

## Main Commands

- `push`: Send data to BadgerMaps API
- `pull`: Retrieve data from BadgerMaps API
- `server`: Run in server mode
- `test`: Run tests and diagnostics
- `version`: Display version information

## Getting Started

1. Authenticate with the API:
   ```bash
   badgermaps auth
   ```

2. Pull data from the API:
   ```bash
   badgermaps pull accounts
   ```

3. Push data to the API:
   ```bash
   badgermaps push account 123456
   ```

## Configuration

Configuration can be managed through:
- Command-line flags
- Environment variables
- Configuration file (.env)

For more details on configuration options, see the [Configuration Documentation](docs/configuration.md).

## Documentation

For more detailed information, please refer to the documentation in the `docs` folder:

- [API Integration](docs/api-integration.md)
- [Command Reference](docs/command-reference.md)
- [Configuration](docs/configuration.md)
- [Data Types](docs/data-types.md)
- [Server Mode](docs/server-mode.md)

## License

This project is licensed under the terms of the license included in the repository.
