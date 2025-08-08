# Configuration

This document describes how to configure the BadgerMaps CLI.

## Configuration Sources

The BadgerMaps CLI supports multiple configuration sources with the following precedence (highest to lowest):

1. Command-line flags
2. Environment variables
3. Configuration files (config.yaml or .env)
4. Default values

When a configuration value is set in multiple places, the highest priority source is used.

## Configuration Files

The CLI looks for configuration in the following locations (in order of preference):

1. `.env` file in the current directory
2. OS-specific configuration directory:
   - Linux/macOS: `~/.config/badgermaps/config.yaml` (default)
   - Windows: `%LocalAppData%\badgermaps\config.yaml`
3. `~/.badgermaps/config.yaml` (for backward compatibility on Linux/macOS)

When you run the `setup` command, you'll be prompted for your API URL, API key, database type, and database-specific configurations. By default, your configuration is saved to `~/.config/badgermaps/config.yaml`. If the directory doesn't exist, it will be created automatically. You can also optionally save to the `.env` file by using the `--env` flag, but you must first create the `.env` file using the `badgermaps utils create-env` command.

Example `.env` file:

```
BADGERMAPS_API_URL=https://badgerapis.badgermapping.com/api/2
BADGERMAPS_API_TOKEN=your_api_token
BADGERMAPS_RATE_LIMIT_REQUESTS=100
BADGERMAPS_RATE_LIMIT_PERIOD=60
BADGERMAPS_MAX_PARALLEL_PROCESSES=5
BADGERMAPS_SERVER_HOST=localhost
BADGERMAPS_SERVER_PORT=8080
BADGERMAPS_SERVER_TLS_ENABLED=false
```

Example `config.yaml` file:

```yaml
# BadgerMaps CLI Configuration
API_KEY: your_api_key
API_URL: https://badgerapis.badgermapping.com/api/2
RATE_LIMIT_REQUESTS: 100
RATE_LIMIT_PERIOD: 60
MAX_PARALLEL_PROCESSES: 5
SERVER_HOST: localhost
SERVER_PORT: 8080
SERVER_TLS_ENABLED: false
```

## Environment Variables

All configuration options can be set using environment variables with the `BADGERMAPS_` prefix:

```bash
export BADGERMAPS_API_TOKEN=your_api_token
export BADGERMAPS_MAX_PARALLEL_PROCESSES=5
```

## Command-line Flags

Global flags that apply to all commands:

- `--verbose` or `-v`: Enable verbose output with additional details
- `--quiet` or `-q`: Suppress all non-essential output
- `--debug`: Enable debug mode with maximum verbosity
- `--no-color`: Disable colored output
- `--config`: Specify a config file (default is $HOME/.badgermaps.yaml)

## Directory Structure

The BadgerMaps CLI uses the following directory structure:

### Configuration Directories
- Linux/macOS: `~/.config/badgermaps/`
- Windows: `%LocalAppData%\badgermaps\`

### Cache Directories
- Linux/macOS: `~/.cache/badgermaps/`
- Windows: `%TEMP%\badgermaps\`

## Configuration Options

### API Configuration

- `API_URL`: The base URL for the BadgerMaps API (default: https://badgerapis.badgermapping.com/api/2)
- `API_TOKEN`: Your BadgerMaps API token

### Rate Limiting

- `RATE_LIMIT_REQUESTS`: Maximum number of API requests per time period (default: 100)
- `RATE_LIMIT_PERIOD`: Time period for rate limiting in seconds (default: 60)

### Parallel Processing

- `MAX_PARALLEL_PROCESSES`: Maximum number of concurrent operations (default: 5)

### Server Configuration

- `SERVER_HOST`: Host address for server mode (default: localhost)
- `SERVER_PORT`: Port number for server mode (default: 8080)
- `SERVER_TLS_ENABLED`: Enable/disable TLS for server (default: false)
- `SERVER_TLS_CERT`: Path to TLS certificate file
- `SERVER_TLS_KEY`: Path to TLS key file

### Database Configuration

- `DB_TYPE`: Database type (sqlite3, postgres, mssql) (default: sqlite3)
- `DB_PATH`: Path to SQLite database file (default: ./config/badgermaps/badgermaps.db)
- `DB_HOST`: Database host for PostgreSQL or MSSQL
- `DB_PORT`: Database port for PostgreSQL or MSSQL
- `DB_NAME`: Database name for PostgreSQL or MSSQL
- `DB_USER`: Database user for PostgreSQL or MSSQL
- `DB_PASSWORD`: Database password for PostgreSQL or MSSQL

## Viewing Current Configuration

To view the current effective configuration:

```bash
badgermaps config show
```

This will display all configuration values and their sources.

## Updating Configuration

To update a configuration value:

```bash
badgermaps config set API_TOKEN your_new_token
```

## Secrets Management

Sensitive information like API tokens is stored securely using one of these methods:

1. Environment variables (recommended for CI/CD environments)
2. OS keychain/credential store (recommended for desktop use)
3. Configuration file with restricted permissions (fallback)

The CLI automatically redacts secrets in logs and error messages, replacing them with `[REDACTED]`.

## Configuration Validation

The CLI validates configuration values when starting up and provides clear error messages for invalid configurations. For example:

- Invalid API URL format
- Missing required configuration
- Invalid port number
- Incompatible settings