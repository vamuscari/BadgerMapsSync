# Environment Configuration

## Overview

The BadgerMaps CLI uses environment variables for configuration. These can be set through a `.env` file or exported directly in your shell. The CLI automatically loads environment variables from a `.env` file in the current directory.

## Quick Setup

1. Copy the example environment file:
```bash
cp env.example .env
```

2. Edit the `.env` file with your configuration:
```bash
# Database configuration
DB_TYPE=sqlite3
DB_NAME=badgersync.db

# API configuration
BADGERMAPS_API_KEY=your_api_key_here

# Logging configuration
LOG_LEVEL=info
LOG_FILE=./badgersync.log
```

## Environment Variables

### Database Configuration

#### DB_TYPE
**Required:** Yes  
**Default:** `sqlite3`  
**Values:** `sqlite3`, `postgres`, `mssql`  
**Description:** Specifies the database type to use.

```bash
DB_TYPE=sqlite3
```

#### DB_HOST
**Required:** For postgres/mssql  
**Default:** `localhost`  
**Description:** Database server hostname or IP address.

```bash
DB_HOST=localhost
DB_HOST=192.168.1.100
DB_HOST=db.example.com
```

#### DB_PORT
**Required:** For postgres/mssql  
**Default:** `5432` (postgres), `1433` (mssql)  
**Description:** Database server port number.

```bash
DB_PORT=5432    # PostgreSQL
DB_PORT=1433    # SQL Server
DB_PORT=5433    # Custom PostgreSQL port
```

#### DB_NAME
**Required:** Yes  
**Default:** `badgersync.db`  
**Description:** Database name or file path.

```bash
# SQLite3 (file path)
DB_NAME=./badgersync.db
DB_NAME=/path/to/database.db

# PostgreSQL/SQL Server (database name)
DB_NAME=badgermaps
DB_NAME=production_db
```

#### DB_USER
**Required:** For postgres/mssql  
**Default:** None  
**Description:** Database username for authentication.

```bash
DB_USER=badger_user
DB_USER=sa
DB_USER=postgres
```

#### DB_PASSWORD
**Required:** For postgres/mssql  
**Default:** None  
**Description:** Database password for authentication.

```bash
DB_PASSWORD=your_secure_password
DB_PASSWORD=mysecretpass123
```

### API Configuration

#### BADGERMAPS_API_KEY
**Required:** For API operations  
**Default:** None  
**Description:** Your BadgerMaps API key for authentication.

```bash
BADGERMAPS_API_KEY=your_api_key_here
BADGERMAPS_API_KEY=sk_1234567890abcdef
```

#### BADGERMAPS_API_URL
**Required:** No  
**Default:** `https://api.badgermapping.com/v1`  
**Description:** Custom API endpoint URL.

```bash
BADGERMAPS_API_URL=https://api.badgermapping.com/v1
BADGERMAPS_API_URL=https://custom-api.example.com/v2
BADGERMAPS_API_URL=http://localhost:8080/api
```

### Logging Configuration

#### LOG_LEVEL
**Required:** No  
**Default:** `info`  
**Values:** `debug`, `info`, `warn`, `error`  
**Description:** Logging verbosity level.

```bash
LOG_LEVEL=debug    # Most verbose
LOG_LEVEL=info     # Standard logging
LOG_LEVEL=warn     # Warnings and errors only
LOG_LEVEL=error    # Errors only
```

#### LOG_FILE
**Required:** No  
**Default:** `./badgersync.log`  
**Description:** Log file path. Leave empty for console-only logging.

```bash
LOG_FILE=./badgersync.log
LOG_FILE=/var/log/badgersync.log
LOG_FILE=           # Console-only logging
```

## Configuration Examples

### Development Setup (SQLite3)

```bash
# .env file
DB_TYPE=sqlite3
DB_NAME=./dev_database.db
BADGERMAPS_API_KEY=your_dev_api_key
LOG_LEVEL=debug
LOG_FILE=./dev.log
```

### Production Setup (PostgreSQL)

```bash
# .env file
DB_TYPE=postgres
DB_HOST=db.production.com
DB_PORT=5432
DB_NAME=badgermaps_prod
DB_USER=badger_user
DB_PASSWORD=secure_production_password
BADGERMAPS_API_KEY=your_production_api_key
LOG_LEVEL=info
LOG_FILE=/var/log/badgersync.log
```

### Enterprise Setup (SQL Server)

```bash
# .env file
DB_TYPE=mssql
DB_HOST=sqlserver.company.com
DB_PORT=1433
DB_NAME=BadgerMapsDB
DB_USER=badger_service
DB_PASSWORD=enterprise_password_123
BADGERMAPS_API_KEY=your_enterprise_api_key
LOG_LEVEL=warn
LOG_FILE=C:\Logs\badgersync.log
```

### Testing Setup

```bash
# .env file
DB_TYPE=sqlite3
DB_NAME=./test_database.db
BADGERMAPS_API_KEY=test_api_key
LOG_LEVEL=debug
LOG_FILE=./test.log
```

## Environment File Loading

The CLI automatically loads environment variables from a `.env` file in the current directory. The loading order is:

1. System environment variables
2. `.env` file in current directory
3. Default values

### File Format

The `.env` file uses a simple key-value format:

```bash
# Database configuration
DB_TYPE=sqlite3
DB_NAME=badgersync.db

# API configuration
BADGERMAPS_API_KEY=your_api_key_here
BADGERMAPS_API_URL=https://api.badgermapping.com/v1

# Logging configuration
LOG_LEVEL=info
LOG_FILE=./badgersync.log

# Comments start with #
# Empty lines are ignored
```

### File Location

The CLI looks for the `.env` file in the current working directory:

```bash
# Current directory
./badgersync pull accounts

# With custom .env file location
BADGERMAPS_CONFIG=/path/to/config.env ./badgersync pull accounts
```

## Security Considerations

### API Key Security

- Never commit API keys to version control
- Use environment variables or `.env` files
- Rotate API keys regularly
- Use different keys for development and production

```bash
# Good: Use .env file (not committed to git)
BADGERMAPS_API_KEY=your_api_key_here

# Bad: Hardcoded in scripts
./badgersync --api-key=your_api_key_here
```

### Database Security

- Use strong passwords for database connections
- Limit database user permissions
- Use SSL connections when available
- Regularly backup database credentials

```bash
# Good: Strong password
DB_PASSWORD=complex_password_with_special_chars_123!

# Bad: Weak password
DB_PASSWORD=password
```

### File Permissions

Ensure proper file permissions for configuration files:

```bash
# Set restrictive permissions on .env file
chmod 600 .env

# Set appropriate permissions on log files
chmod 644 badgersync.log
```

## Validation

The CLI validates environment variables on startup:

```bash
./badgersync test
```

This command will:
- Verify database connection parameters
- Test API key validity
- Check file permissions
- Display configuration summary

## Troubleshooting

### Common Issues

1. **Missing .env File**
   ```bash
   # Error: Configuration not found
   # Solution: Copy example file
   cp env.example .env
   ```

2. **Invalid Database Type**
   ```bash
   # Error: Unsupported database type
   # Solution: Use supported type
   DB_TYPE=sqlite3  # or postgres, mssql
   ```

3. **Missing API Key**
   ```bash
   # Error: API key required
   # Solution: Add API key to .env
   BADGERMAPS_API_KEY=your_api_key_here
   ```

4. **Database Connection Failed**
   ```bash
   # Error: Connection refused
   # Solution: Check database server
   # - Verify server is running
   # - Check host/port configuration
   # - Verify credentials
   ```

### Debug Mode

Enable debug logging to troubleshoot configuration issues:

```bash
# .env file
LOG_LEVEL=debug
```

This will log:
- Environment variable loading
- Configuration validation
- Connection attempts
- API request details

### Configuration Testing

Test your configuration without making changes:

```bash
# Test database connection
./badgersync utils test-db

# Test API connection
./badgersync utils test-api

# Test full configuration
./badgersync test
```

## Best Practices

1. **Use .env Files**: Prefer `.env` files over export commands
2. **Version Control**: Never commit `.env` files to version control
3. **Documentation**: Document required environment variables
4. **Validation**: Always test configuration before deployment
5. **Security**: Use strong passwords and secure API keys
6. **Backup**: Keep backups of configuration files
7. **Monitoring**: Monitor log files for configuration issues 