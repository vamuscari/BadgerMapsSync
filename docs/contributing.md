# Contributing to BadgerMaps CLI

## Overview

Thank you for your interest in contributing to the BadgerMaps CLI! This document provides guidelines and information for contributors.

## Getting Started

### Prerequisites

- Go 1.19 or higher
- Git
- Make (optional, for build scripts)
- SQLite3, PostgreSQL, or SQL Server (for testing)

### Development Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/your-username/BadgerMaps_CLI.git
   cd BadgerMaps_CLI
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Build the Project**
   ```bash
   go build -o badgersync main.go
   ```

4. **Set Up Environment**
   ```bash
   cp env.example .env
   # Edit .env with your configuration
   ```

5. **Run Tests**
   ```bash
   go test ./...
   ```

## Project Structure

```
BadgerMaps_CLI/
├── api/                    # API client implementation
│   └── client.go          # Main API client
├── database/              # Database layer
│   ├── client.go          # Database client
│   ├── sql_loader.go      # SQL file loader
│   ├── sqlite3/           # SQLite3 SQL files
│   ├── postgres/          # PostgreSQL SQL files
│   └── mssql/             # SQL Server SQL files
├── testing/               # Test files and mock server
│   ├── mock_server.go     # Mock API server
│   ├── push_test.go       # Push functionality tests
│   └── json/              # Test JSON responses
├── docs/                  # Documentation
├── main.go                # Main CLI application
├── go.mod                 # Go module file
├── go.sum                 # Go module checksums
├── env.example            # Environment configuration example
├── README.md              # Project README
├── LICENSE                # Project license
└── Makefile               # Build and development scripts
```

## Development Guidelines

### Code Style

- Follow Go formatting standards (`gofmt`)
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions focused and concise
- Use proper error handling

### Error Handling

Always handle errors properly:

```go
// Good
result, err := someFunction()
if err != nil {
    return fmt.Errorf("failed to execute function: %w", err)
}

// Bad
result, _ := someFunction()
```

### Logging

Use structured logging with appropriate levels:

```go
// Info level for general operations
log.Printf("Processing %d accounts", len(accounts))

// Debug level for detailed information
log.Printf("Account %d: %s %s", account.ID, account.FirstName, account.LastName)

// Error level for errors
log.Printf("Failed to process account %d: %v", account.ID, err)
```

### Database Operations

- Use transactions for multi-step operations
- Always close database connections
- Use prepared statements for repeated queries
- Handle database-specific syntax differences

```go
// Good: Using transactions
tx, err := db.Begin()
if err != nil {
    return fmt.Errorf("failed to begin transaction: %w", err)
}
defer tx.Rollback()

// ... perform operations ...

if err := tx.Commit(); err != nil {
    return fmt.Errorf("failed to commit transaction: %w", err)
}
```

### API Operations

- Implement proper retry logic
- Handle rate limiting
- Use appropriate HTTP status codes
- Validate API responses

```go
// Good: Retry logic with backoff
for attempts := 0; attempts < maxRetries; attempts++ {
    resp, err := client.Do(req)
    if err == nil {
        return resp, nil
    }
    
    if attempts < maxRetries-1 {
        time.Sleep(time.Duration(attempts+1) * time.Second)
    }
}
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestFunctionName

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Writing Tests

- Write tests for all new functionality
- Use descriptive test names
- Test both success and error cases
- Use table-driven tests for multiple scenarios

```go
func TestStoreAccounts(t *testing.T) {
    tests := []struct {
        name     string
        accounts []api.Account
        wantErr  bool
    }{
        {
            name: "valid accounts",
            accounts: []api.Account{
                {ID: 1, FirstName: "John", LastName: "Doe"},
            },
            wantErr: false,
        },
        {
            name:     "empty accounts",
            accounts: []api.Account{},
            wantErr:  false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Mock Server

The project includes a mock server for testing API interactions:

```bash
# Start mock server
cd testing
go run mock_server.go

# Run tests against mock server
go test -v ./...
```

## Database Development

### Adding New Tables

1. Create SQL files for each database type:
   - `database/sqlite3/create_new_table.sql`
   - `database/postgres/create_new_table.sql`
   - `database/mssql/create_new_table.sql`

2. Update the schema initialization in `database/client.go`

3. Add corresponding merge/insert/delete SQL files

### SQL File Guidelines

- Use database-specific syntax when necessary
- Include proper indexes for performance
- Use parameterized queries for security
- Handle database-specific data types

```sql
-- SQLite3 example
CREATE TABLE IF NOT EXISTS new_table (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- PostgreSQL example
CREATE TABLE IF NOT EXISTS new_table (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## API Development

### Adding New Endpoints

1. Define the data structure in `api/client.go`
2. Implement the API method
3. Add corresponding database operations
4. Update CLI commands in `main.go`
5. Add tests for the new functionality

### API Response Handling

```go
type NewResponse struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func (c *Client) GetNewData() ([]NewResponse, error) {
    resp, err := c.makeRequest("GET", "/new-endpoint/", nil)
    if err != nil {
        return nil, fmt.Errorf("failed to get new data: %w", err)
    }
    
    var data []NewResponse
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    return data, nil
}
```

## CLI Development

### Adding New Commands

1. Define the command in `main.go`
2. Implement the command logic
3. Add help text and documentation
4. Add tests for the command

```go
var newCmd = &cobra.Command{
    Use:   "new-command",
    Short: "Description of the new command",
    Long: `Detailed description of the new command
    and its usage examples.`,
    Run: func(cmd *cobra.Command, args []string) {
        // Command implementation
    },
}
```

## Documentation

### Updating Documentation

- Update relevant documentation when adding features
- Include usage examples
- Document configuration options
- Update troubleshooting sections

### Documentation Structure

- `docs/api.md` - API integration documentation
- `docs/database.md` - Database configuration and schema
- `docs/environment.md` - Environment variable configuration
- `docs/contributing.md` - This file

## Pull Request Process

### Before Submitting

1. **Test Your Changes**
   ```bash
   go test ./...
   go build -o badgersync main.go
   ./badgersync test
   ```

2. **Check Code Style**
   ```bash
   go fmt ./...
   go vet ./...
   ```

3. **Update Documentation**
   - Update relevant documentation files
   - Add comments for new functions
   - Update README if necessary

### Pull Request Guidelines

1. **Title**: Use clear, descriptive titles
2. **Description**: Explain what the PR does and why
3. **Tests**: Include tests for new functionality
4. **Documentation**: Update documentation as needed
5. **Breaking Changes**: Clearly mark breaking changes

### Example Pull Request

```markdown
## Description
Adds support for new data type synchronization.

## Changes
- Added new API endpoint for data retrieval
- Implemented database storage for new data type
- Added CLI command for syncing new data
- Updated documentation

## Testing
- Added unit tests for new functionality
- Tested with SQLite3, PostgreSQL, and SQL Server
- Verified API integration works correctly

## Breaking Changes
None

## Checklist
- [x] Tests added and passing
- [x] Documentation updated
- [x] Code follows project style
- [x] No breaking changes
```

## Code Review

### Review Process

1. **Automated Checks**: CI/CD pipeline runs tests and linting
2. **Manual Review**: At least one maintainer reviews the PR
3. **Feedback**: Address any feedback or requested changes
4. **Merge**: PR is merged after approval

### Review Guidelines

- Check code quality and style
- Verify tests are comprehensive
- Ensure documentation is updated
- Test functionality manually if needed
- Provide constructive feedback

## Release Process

### Versioning

The project uses semantic versioning (MAJOR.MINOR.PATCH):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Steps

1. **Update Version**: Update version in `main.go`
2. **Update Changelog**: Document changes
3. **Create Tag**: Create git tag for the release
4. **Build Binaries**: Build for target platforms
5. **Publish Release**: Create GitHub release with binaries

## Getting Help

### Questions and Issues

- **GitHub Issues**: Use GitHub issues for bug reports and feature requests
- **Discussions**: Use GitHub Discussions for questions and general discussion
- **Documentation**: Check the docs folder for detailed information

### Communication

- Be respectful and constructive
- Provide clear, detailed information
- Include relevant code examples
- Share error messages and logs

## License

By contributing to this project, you agree that your contributions will be licensed under the same license as the project (see LICENSE file).

Thank you for contributing to BadgerMaps CLI! 