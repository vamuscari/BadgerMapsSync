# BadgerMaps CLI Overview

## What is BadgerMaps CLI?

BadgerMaps CLI is a command-line interface tool designed to synchronize data between the BadgerMaps API and various database systems. It provides a robust, scalable solution for managing BadgerMaps data locally while maintaining synchronization with the cloud-based API.

## Key Features

### Multi-Database Support
- **SQLite3** - Lightweight, file-based database for development and small deployments
- **PostgreSQL** - Full-featured relational database for production environments
- **SQL Server** - Enterprise-grade database for Windows environments

### BadgerMaps API Integration
- **Accounts** - Customer and account information management
- **Routes** - Route planning and optimization data
- **Check-ins** - Location tracking and visit records
- **User Profiles** - User management and authentication data

### Automated Operations
- **Schema Management** - Automatic table creation and indexing
- **Data Synchronization** - Bidirectional sync between API and database
- **Error Handling** - Robust error recovery and logging
- **Configuration Management** - Environment-based configuration

## Architecture

### Core Components

```
BadgerMaps CLI
├── API Client (api/client.go)
│   ├── Authentication
│   ├── Request/Response handling
│   ├── Rate limiting
│   └── Error management
├── Database Client (database/client.go)
│   ├── Connection management
│   ├── Schema initialization
│   ├── Transaction support
│   └── Cross-database compatibility
├── CLI Interface (main.go)
│   ├── Command parsing
│   ├── Configuration loading
│   └── Logging setup
└── SQL Layer (database/*/)
    ├── Database-specific SQL files
    ├── Schema definitions
    └── Data operations
```

### Data Flow

1. **Configuration Loading**
   - Environment variables from `.env` file
   - Database connection parameters
   - API authentication credentials

2. **Database Operations**
   - Schema initialization and validation
   - Data storage using merge operations
   - Transaction management for data integrity

3. **API Integration**
   - Authentication with API keys
   - Request/response handling
   - Rate limiting and retry logic

4. **Synchronization**
   - Pull: API → Database
   - Push: Database → API
   - Conflict resolution and error handling

## Use Cases

### Development and Testing
- Local development with SQLite3
- API testing and debugging
- Data validation and quality checks

### Production Deployment
- Enterprise data management
- Backup and disaster recovery
- Performance optimization

### Data Analysis
- Offline data processing
- Custom reporting and analytics
- Data migration and transformation

## Technology Stack

### Core Technologies
- **Go** - Primary programming language
- **Cobra** - CLI framework
- **SQL** - Database operations
- **HTTP/JSON** - API communication

### Database Drivers
- **SQLite3** - `github.com/mattn/go-sqlite3`
- **PostgreSQL** - `github.com/lib/pq`
- **SQL Server** - `github.com/microsoft/go-mssqldb`

### Dependencies
- **Environment Management** - `github.com/joho/godotenv`
- **Logging** - Standard Go `log` package
- **HTTP Client** - Standard Go `net/http`

## Getting Started

### Quick Setup
```bash
# Clone and build
git clone <repository-url>
cd BadgerMaps_CLI
go build -o badgersync main.go

# Configure environment
cp env.example .env
# Edit .env with your settings

# Test connectivity
./badgersync test

# Pull data
./badgersync pull accounts
```

### Basic Workflow
1. **Configure** - Set up database and API credentials
2. **Initialize** - Create database schema
3. **Sync** - Pull or push data as needed
4. **Monitor** - Check logs and verify operations

## Configuration

### Environment Variables
- **Database Configuration** - Type, host, credentials
- **API Configuration** - API key, base URL
- **Logging Configuration** - Level, file path

### Database Setup
- **SQLite3** - File-based, no server required
- **PostgreSQL** - Server installation and user setup
- **SQL Server** - Instance configuration and authentication

### API Setup
- **Authentication** - API key generation and management
- **Permissions** - Read/write access configuration
- **Rate Limiting** - Request limits and throttling

## Data Model

### Core Entities

#### Accounts
- Customer and account information
- Contact details and preferences
- Business relationship data

#### Routes
- Route planning and optimization
- Waypoint management
- Scheduling and timing

#### Check-ins
- Location tracking and visits
- Time and location data
- Notes and observations

#### User Profiles
- User authentication and roles
- Personal information
- Access permissions

### Relationships
- Accounts ↔ Routes (one-to-many)
- Routes ↔ Check-ins (one-to-many)
- Users ↔ Profiles (one-to-one)
- Accounts ↔ Check-ins (one-to-many)

## Performance

### Optimization Strategies
- **Indexing** - Automatic index creation for performance
- **Batching** - Bulk operations for efficiency
- **Connection Pooling** - Reusable database connections
- **Caching** - Local data caching where appropriate

### Monitoring
- **Logging** - Comprehensive operation logging
- **Metrics** - Performance and usage statistics
- **Error Tracking** - Detailed error reporting


---

This overview provides a comprehensive introduction to the BadgerMaps CLI project. For detailed information on specific topics, refer to the individual documentation files in this directory. 