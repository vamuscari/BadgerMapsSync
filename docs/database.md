# Database Documentation

## Overview

The BadgerMaps CLI supports multiple database systems for storing and syncing data. This document covers database configuration, schema, and usage patterns.

## Supported Databases

### SQLite3 (Default)
- **Driver:** `github.com/mattn/go-sqlite3`
- **File-based:** No server required
- **Best for:** Development, testing, small deployments
- **Features:** Full SQL support, transactions, indexes

### PostgreSQL
- **Driver:** `github.com/lib/pq`
- **Server-based:** Requires PostgreSQL server
- **Best for:** Production, large datasets, concurrent access
- **Features:** JSONB support, advanced indexing, foreign keys

### SQL Server
- **Driver:** `github.com/microsoft/go-mssqldb`
- **Server-based:** Requires SQL Server instance
- **Best for:** Enterprise environments, Windows integration
- **Features:** Full enterprise features, clustering support

## Database Configuration

### Environment Variables

Configure your database connection in the `.env` file:

```bash
# Database Type (sqlite3, postgres, mssql)
DB_TYPE=sqlite3

# Database Host (for postgres/mssql)
DB_HOST=localhost

# Database Port (5432 for postgres, 1433 for mssql)
DB_PORT=5432

# Database Name
DB_NAME=badgersync.db

# Database Username (for postgres/mssql)
DB_USER=

# Database Password (for postgres/mssql)
DB_PASSWORD=
```

### Configuration Examples

#### SQLite3
```bash
DB_TYPE=sqlite3
DB_NAME=./badgersync.db
```

#### PostgreSQL
```bash
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_NAME=badgermaps
DB_USER=badger_user
DB_PASSWORD=your_password
```

#### SQL Server
```bash
DB_TYPE=mssql
DB_HOST=localhost
DB_PORT=1433
DB_NAME=badgermaps
DB_USER=sa
DB_PASSWORD=your_password
```

## Database Schema

### Tables

#### accounts
Stores customer/account information.

```sql
CREATE TABLE accounts (
    id INTEGER PRIMARY KEY,
    first_name TEXT,
    last_name TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### routes
Stores route information.

```sql
CREATE TABLE routes (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    account_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### account_checkins
Stores check-in records.

```sql
CREATE TABLE account_checkins (
    id INTEGER PRIMARY KEY,
    account_id INTEGER,
    route_id INTEGER,
    checkin_time TIMESTAMP,
    latitude REAL,
    longitude REAL,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### user_profiles
Stores user profile information.

```sql
CREATE TABLE user_profiles (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    first_name TEXT,
    last_name TEXT,
    email TEXT,
    role TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### account_locations
Stores account location data.

```sql
CREATE TABLE account_locations (
    id INTEGER PRIMARY KEY,
    account_id INTEGER,
    latitude REAL,
    longitude REAL,
    address TEXT,
    city TEXT,
    state TEXT,
    zip TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### data_sets
Stores data set definitions.

```sql
CREATE TABLE data_sets (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### data_set_values
Stores data set values.

```sql
CREATE TABLE data_set_values (
    id INTEGER PRIMARY KEY,
    data_set_id INTEGER,
    account_id INTEGER,
    value TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### route_waypoints
Stores route waypoint information.

```sql
CREATE TABLE route_waypoints (
    id INTEGER PRIMARY KEY,
    route_id INTEGER,
    sequence_order INTEGER,
    latitude REAL,
    longitude REAL,
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Indexes

The CLI automatically creates indexes for optimal performance:

```sql
-- Primary indexes
CREATE INDEX IF NOT EXISTS idx_accounts_id ON accounts(id);
CREATE INDEX IF NOT EXISTS idx_routes_id ON routes(id);
CREATE INDEX IF NOT EXISTS idx_checkins_id ON account_checkins(id);
CREATE INDEX IF NOT EXISTS idx_profiles_id ON user_profiles(id);

-- Foreign key indexes
CREATE INDEX IF NOT EXISTS idx_routes_account_id ON routes(account_id);
CREATE INDEX IF NOT EXISTS idx_checkins_account_id ON account_checkins(account_id);
CREATE INDEX IF NOT EXISTS idx_checkins_route_id ON account_checkins(route_id);
CREATE INDEX IF NOT EXISTS idx_locations_account_id ON account_locations(account_id);
CREATE INDEX IF NOT EXISTS idx_waypoints_route_id ON route_waypoints(route_id);

-- Timestamp indexes
CREATE INDEX IF NOT EXISTS idx_accounts_updated_at ON accounts(updated_at);
CREATE INDEX IF NOT EXISTS idx_routes_updated_at ON routes(updated_at);
CREATE INDEX IF NOT EXISTS idx_profiles_updated_at ON user_profiles(updated_at);
```

## Database Operations

### Schema Management

#### Create Tables
```bash
./badgersync utils create-tables
```

This command:
- Creates all required tables
- Sets up indexes for performance
- Handles database-specific syntax differences

#### Drop Tables
```bash
./badgersync utils drop-tables
```

**Warning:** This will permanently delete all data in the database.

### Data Operations

#### Pull Data
```bash
# Pull all data types
./badgersync pull

# Pull specific data types
./badgersync pull accounts
./badgersync pull routes
./badgersync pull checkins
./badgersync pull profiles
```

#### Push Data
```bash
# Push all data types
./badgersync push

# Push specific data types
./badgersync push accounts
./badgersync push routes
./badgersync push checkins
```

## SQL Files

The CLI uses SQL files for database operations. These are located in `database/{database_type}/`:

### SQLite3 Files
- `database/sqlite3/create_*.sql` - Table creation scripts
- `database/sqlite3/merge_*.sql` - Data merge operations
- `database/sqlite3/insert_*.sql` - Data insertion scripts
- `database/sqlite3/delete_*.sql` - Data deletion scripts

### PostgreSQL Files
- `database/postgres/create_*.sql` - Table creation scripts
- `database/postgres/merge_*.sql` - Data merge operations
- `database/postgres/insert_*.sql` - Data insertion scripts
- `database/postgres/delete_*.sql` - Data deletion scripts

### SQL Server Files
- `database/mssql/create_*.sql` - Table creation scripts
- `database/mssql/merge_*.sql` - Data merge operations
- `database/mssql/insert_*.sql` - Data insertion scripts
- `database/mssql/delete_*.sql` - Data deletion scripts

## Database Client Implementation

The database client is implemented in `database/client.go` and provides:

- Connection management
- Schema initialization
- Transaction support
- SQL file loading
- Cross-database compatibility

### Key Methods

```go
// Connect establishes a database connection
func (c *Client) Connect() error

// InitializeSchema creates all required tables and indexes
func (c *Client) InitializeSchema() error

// StoreAccounts stores accounts using merge_accounts_basic
func (c *Client) StoreAccounts(accounts []api.Account) error

// GetAccountIDs retrieves all account IDs from the database
func (c *Client) GetAccountIDs() ([]int, error)

// Close closes the database connection
func (c *Client) Close() error
```

## Performance Considerations

### SQLite3
- Use WAL mode for better concurrency
- Regular VACUUM operations for maintenance
- Monitor file size growth

### PostgreSQL
- Configure connection pooling
- Use appropriate indexes
- Monitor query performance

### SQL Server
- Configure connection timeouts
- Use appropriate isolation levels
- Monitor lock contention

## Backup and Recovery

### SQLite3
```bash
# Backup
cp badgersync.db badgersync.db.backup

# Restore
cp badgersync.db.backup badgersync.db
```

### PostgreSQL
```bash
# Backup
pg_dump -h localhost -U badger_user badgermaps > backup.sql

# Restore
psql -h localhost -U badger_user badgermaps < backup.sql
```

### SQL Server
```bash
# Backup
sqlcmd -S localhost -U sa -P password -Q "BACKUP DATABASE badgermaps TO DISK = 'backup.bak'"

# Restore
sqlcmd -S localhost -U sa -P password -Q "RESTORE DATABASE badgermaps FROM DISK = 'backup.bak'"
```

## Troubleshooting

### Common Issues

1. **Connection Errors**
   - Verify database server is running
   - Check connection parameters
   - Ensure proper permissions
   - Review error messages which include specific database type and connection details

2. **Schema Errors**
   - Run `./badgersync utils create-tables`
   - Check SQL file syntax
   - Verify database type configuration
   - Look for specific file paths in error messages to locate problematic SQL files

3. **Performance Issues**
   - Check index creation
   - Monitor query execution plans
   - Consider database-specific optimizations

4. **Data Corruption**
   - Restore from backup
   - Check disk space
   - Verify file permissions

5. **SQL File Errors**
   - Check if SQL files exist in the correct directory (error messages will indicate the expected path)
   - Verify file permissions
   - Review SQL syntax for database-specific compatibility

### Understanding Error Messages

The database package provides detailed error messages to help diagnose issues:

1. **Database Connection Errors**
   ```
   failed to open postgres database connection: [original error] (check connection parameters)
   ```
   This indicates a problem with your database connection settings.

2. **SQL File Loading Errors**
   ```
   failed to read SQL file create_accounts_table.sql: [original error] (check if file exists and has correct permissions in database/postgres directory)
   ```
   This shows which SQL file is missing and where it should be located.

3. **SQL Execution Errors**
   ```
   failed to execute SQL statement #2 in create_indexes.sql: [original error] (check SQL syntax and database permissions)
   ```
   This indicates which statement in which file has a syntax error.

### Debug Mode

Enable debug logging for database operations:

```bash
# .env file
LOG_LEVEL=debug
```

This will log all SQL operations and connection details. 