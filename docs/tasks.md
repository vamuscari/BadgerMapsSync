# BadgerMaps CLI - Pending Tasks

This document lists the requirements from the specification that have not been completed yet.

## Pending Status Table for Accounts

A critical pending task is to implement a proper pending status table for accounts that need to be added, updated, or deleted. Current issues:

- The file `create_accounts_pending_changes_table.sql` creates a table named "Accounts" with a Status field, but this conflicts with the regular Accounts table.
- A separate table (e.g., "AccountsPendingChanges") should be created to track pending changes.
- The table should include fields to track:
  - The type of change (add, update, delete)
  - The status of the change (pending, completed, failed)
  - Timestamps for when the change was requested and processed
  - Reference to the original account ID (for updates and deletes)

## Other Incomplete Requirements

Based on the specification document, the following items have not been fully implemented:

### Command Structure
- Implementation of the `config` command for configuring the application

### API and Database Integration
- Proper implementation of rate limit handling with exponential backoff and jitter

### Configuration and Authentication
- Complete implementation of configuration management using Viper
- Secure storage of API tokens
- Implementation of token refresh mechanism for OAuth authentication

### Server and Webhook Features
- Implementation of server mode with webhook support
- Implementation of built-in scheduler with cron-like syntax

### Search and Autocomplete Features
- Implementation of the `search` command
- Implementation of caching for search results and autocomplete data

### User Interface and Feedback
- Implementation of consistent color scheme across the application
- Implementation of clear, actionable error messages with error codes

### Development and Testing
- Generation of man page for Unix-like systems
- Implementation of embedding SQL files for easier deployment