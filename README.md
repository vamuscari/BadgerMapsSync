# BadgerMaps CLI

A command line interface for interacting with the BadgerMaps API.

## Overview

The BadgerMaps CLI allows you to synchronize data between the BadgerMaps API and a local database. It provides a powerful interface for managing customer accounts, routes, check-ins, and user profiles. The CLI supports `sqlite3`, `postgres`, and `mssql` as database backends and features a dynamic schema that adapts to your BadgerMaps data.

## Installation

```bash
# Clone the repository
git clone https://github.com/vamuscari/BadgerMaps_CLI.git

# Navigate to the project directory
cd BadgerMaps_CLI

# Build the application
go build -o badgermaps
```

## Getting Started

The first time you run the CLI, you should use the interactive `config` command to set up your API key and database connection.

```bash
./badgermaps config
```

This will guide you through the process of configuring the CLI and creating the necessary database schema.

## Main Commands

- `config`: Run the interactive setup to configure the CLI.
- `pull`: Retrieve data from the BadgerMaps API and store it in your local database.
  - `pull account [id]`: Pull a single account.
  - `pull accounts`: Pull all accounts.
  - `pull checkin [id]`: Pull a single checkin.
  - `pull checkins`: Pull all checkins.
  - `pull route [id]`: Pull a single route.
  - `pull routes`: Pull all routes.
  - `pull profile`: Pull your user profile.
  - `pull all`: Pull all data (accounts, checkins, routes, and profile).
- `push`: Push data from your local database to the BadgerMaps API.
- `server`: Run in server mode to listen for webhooks from the BadgerMaps API.
- `version`: Display the version of the CLI.

## Database

The CLI uses a local database to store the data pulled from the BadgerMaps API. The database schema is created and managed automatically by the CLI.

### Dynamic Schema

The database features a dynamic schema that adapts to your BadgerMaps data. The `AccountsWithLabels` view is dynamically created based on the `DataSets` in your BadgerMaps account, providing a convenient way to view your accounts with custom labels.

Triggers are used to keep the `AccountsWithLabels` view and other parts of the schema up-to-date when the `DataSets` table is modified.

### Supported Databases

- `sqlite3`
- `postgres`
- `mssql`

## License

This project is licensed under the terms of the license included in the repository.