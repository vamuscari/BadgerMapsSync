# BadgerMaps CLI

A command-line interface for syncing BadgerMaps data with various databases including **PostgreSQL**, SQL Server, and SQLite.

## Quick Start

```bash
# Clone and build
git clone <repository-url>
cd badgermaps-cli
go build -o badgersync

# Configure your environment
cp env.example .env
# Edit .env with your database and API settings

# Test connectivity
./badgersync test

# Pull all data from BadgerMaps API
./badgersync pull

# Pull specific data types
./badgersync pull accounts
./badgersync pull routes
./badgersync pull checkins
./badgersync pull profiles

# Push data to BadgerMaps API
./badgersync push

# Database utilities
./badgersync utils create-tables
./badgersync utils drop-tables
```

## Features

- **Multi-Database Support** - PostgreSQL, SQL Server, SQLite
- **BadgerMaps API Integration** - Sync accounts, routes, checkins, and user profiles
- **Automatic Schema Creation** - Tables and indexes created automatically
- **Environment File Support** - Automatic loading of `.env` files
- **Comprehensive Logging** - Console and file logging with configurable levels
- **Modular Architecture** - Separated API and database packages

## Documentation

For detailed documentation, see the [docs/](docs/) folder:

- **[Overview](docs/overview.md)** - Project overview, architecture, and key concepts
- **[Environment Configuration](docs/environment.md)** - Complete guide to configuring the CLI
- **[API Documentation](docs/api.md)** - BadgerMaps API integration and endpoints
- **[Database Documentation](docs/database.md)** - Database setup, schema, and operations
- **[Database Error Handling](docs/database_errors.md)** - Detailed database error messages and troubleshooting
- **[Contributing Guidelines](docs/contributing.md)** - How to contribute to the project
- **[Troubleshooting Guide](docs/troubleshooting.md)** - Common issues and solutions

## Installation

### Prerequisites

- Go 1.21 or later
- PostgreSQL 12+ (for PostgreSQL support)
- BadgerMaps API key

### Build from Source

```bash
git clone <repository-url>
cd badgermaps-cli
go build -o badgersync
```

## Configuration

The BadgerMaps CLI uses environment variables for configuration. Copy `env.example` to `.env` and configure your settings:

```bash
cp env.example .env
# Edit .env with your database and API settings
```

For detailed configuration information, see [Environment Configuration](docs/environment.md).

## Usage

### Basic Commands

```bash
# Pull data from API
./badgersync pull [accounts|routes|checkins|profiles|all]

# Push data to API
./badgersync push [accounts|routes|checkins|profiles|all]

# Database utilities
./badgersync utils create-tables
./badgersync utils drop-tables

# Test connectivity
./badgersync test
```

### Available Data Types

- **accounts** - Customer account information
- **routes** - Route planning data  
- **checkins** - Sales visit tracking
- **profiles** - User profile data
- **all** - All data types (default when no arguments provided)

For detailed usage information, see [API Documentation](docs/api.md) and [Database Documentation](docs/database.md).

## Database Support

The CLI supports multiple database systems:

- **SQLite3** (default) - File-based, no server required
- **PostgreSQL** - Full-featured relational database
- **SQL Server** - Enterprise database system

For detailed database setup and configuration, see [Database Documentation](docs/database.md).

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_TYPE` | Database type (sqlite3, postgres, mssql) | sqlite3 |
| `DB_HOST` | Database host (for postgres/mssql) | localhost |
| `DB_PORT` | Database port (5432 for postgres, 1433 for mssql) | 5432 |
| `DB_NAME` | Database name or file path | badgersync.db |
| `DB_USER` | Database username (for postgres/mssql) | (empty) |
| `DB_PASSWORD` | Database password | (empty) |
| `BADGERMAPS_API_KEY` | BadgerMaps API key for data sync | (empty) |
| `BADGERMAPS_API_URL` | BadgerMaps API base URL | https://badgerapis.badgermapping.com/api/2 |
| `LOG_LEVEL` | Logging level (debug, info, warn, error) | info |
| `LOG_FILE` | Log file path (leave empty for console-only) | (empty) |

For detailed configuration examples, see [Environment Configuration](docs/environment.md).

## Troubleshooting

For common issues and solutions, see [Troubleshooting Guide](docs/troubleshooting.md).

## Contributing

For information on contributing to the project, see [Contributing Guidelines](docs/contributing.md).

## License

See LICENSE file for details.