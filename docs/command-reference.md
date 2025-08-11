# Command Reference

This document provides detailed information about the commands available in the BadgerMaps CLI.

## Command Structure

The BadgerMaps CLI uses a hierarchical command structure with subcommands and flags. The general syntax is:

```
badgermaps [global flags] command [subcommand] [arguments] [command flags]
```

## Global Flags

These flags can be used with any command:

- `--verbose` or `-v`: Enable verbose output with additional details
- `--quiet` or `-q`: Suppress all non-essential output
- `--debug`: Enable debug mode with maximum verbosity
- `--no-color`: Disable colored output
- `--config`: Specify a config file (default is $HOME/.badgermaps.yaml)

## Main Commands

### push

Send data to the BadgerMaps API.

```bash
# Push a specific account
badgermaps push account 123456

# Push multiple accounts
badgermaps push accounts 123456 123457

# Push all accounts
badgermaps push accounts
```

### pull

Retrieve data from the BadgerMaps API.

```bash
# Pull a specific account
badgermaps pull account 123456

# Pull multiple accounts
badgermaps pull accounts 123456 123457

# Pull all accounts
badgermaps pull accounts
```

### server

Run in server mode to listen for incoming webhooks from the BadgerMaps API.

```bash
# Start the server with default settings
badgermaps server

# Start the server on a specific port
badgermaps server --port 8080

# Start the server with TLS enabled
badgermaps server --tls
```

### test

Run tests and diagnostics to verify API connectivity and functionality.

```bash
# Test API connectivity
badgermaps test api

# Test a specific endpoint
badgermaps test endpoint customers

# Test all endpoints
badgermaps test all

# Save API response to a file
badgermaps test -s api
```

### utils

Utility commands for maintenance and database operations.

```bash
# Check database tables
badgermaps utils check-tables

# Create missing tables
badgermaps utils create-tables

# Verify database schema
badgermaps utils verify-schema
```

### auth

Authenticate with the BadgerMaps API.

```bash
# Start interactive authentication
badgermaps auth

# Authenticate with a token
badgermaps auth --token YOUR_API_TOKEN
```

### search

Find items by name or ID with fuzzy matching. The search displays the item's ID, name, type, and three additional fields to help with identification.

```bash
# Search for all item types
badgermaps search "Acme Corp"

# Search for accounts only
badgermaps search "Acme Corp" --type accounts

# Search for routes only
badgermaps search "Route" --type routes

# Search for locations only
badgermaps search "San Francisco" --type locations

# Search for profiles only
badgermaps search "John" --type profiles

# Force online search (bypassing cache)
badgermaps search "Acme Corp" --online
```

The search uses fuzzy finding to match keywords in various fields, not just the name. Results are displayed with additional fields specific to each item type:
- Accounts: Email, Phone, Owner
- Routes: Date, Start Address, Destination Address
- Locations: Address, City, State
- Profiles: Email, First Name, Last Name

### autocomplete

Generate shell autocompletion scripts.

```bash
# Generate Bash completion script
badgermaps autocomplete bash

# Generate Zsh completion script
badgermaps autocomplete zsh

# Generate Fish completion script
badgermaps autocomplete fish

# Generate PowerShell completion script
badgermaps autocomplete powershell

# Install completion for the current shell
badgermaps autocomplete install
```

### help

Display help information for any command.

```bash
# Display general help
badgermaps help

# Display help for a specific command
badgermaps help pull

# Display help for a subcommand
badgermaps help pull accounts
```

### version

Display version information.

```bash
badgermaps version
```

## Command Syntax Patterns

### Singular vs Plural Data Types

- Singular form (e.g., `account`): Operate on a specific item by ID
- Plural form (e.g., `accounts`): Retrieve a list of all items, then sync detailed data

### Bulk Operations

For bulk operations, use one of these patterns:

- `pull accounts`: Retrieve all accounts
- `pull account all`: Retrieve all accounts (alternative syntax)

## Exit Codes

The CLI uses the following exit codes:

- `0`: Success
- `1`: General error
- `2`: Misuse of shell builtins
- `3`: Command line parsing error
- `4`: Authentication failure
- `5`: API error
- `6`: Database error
- `7`: Network error
- `8`: Rate limit exceeded
- `9`: Timeout error