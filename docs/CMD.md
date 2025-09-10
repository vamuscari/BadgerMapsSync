# CMD Modules

This document provides an overview of the `cmd` modules in the BadgerMaps CLI.

- [Back to Architecture](./ARCHITECTURE.md)
- [README](../README.md)

## Overview

The `cmd` package contains the implementation of the different CLI commands. Each subcommand is organized into its own subdirectory. The commands are built using the Cobra library, which provides a robust framework for creating command-line applications.

## Command Structure

Each command is defined in a `.go` file within a subdirectory of `cmd`. For example, the `pull` command is defined in `cmd/pull/pull.go`.

The basic structure of a command file includes:

- An `init()` function to register the command and its flags with the root command.
- A `Run` function that contains the logic to be executed when the command is called.

## Main Commands

The main commands available in the BadgerMaps CLI are:

### `config`

- **File**: `cmd/config/config.go`
- **Description**: Manages the application's configuration. It provides an interactive setup guide for new users and allows existing users to view and edit their configuration.

### `pull`

- **File**: `cmd/pull/pull.go`
- **Description**: Pulls data from the BadgerMaps API and stores it in the local database.
- **Subcommands**:
    - `pull all`: Pulls all data (accounts, check-ins, routes, and user profile).
    - `pull accounts`: Pulls only account data.
    - `pull checkins`: Pulls only check-in data.
    - `pull routes`: Pulls only route data.
    - `pull profile`: Pulls only the user profile data.
- **Helpers**: The `cmd/pull/helpers.go` file contains functions for pulling and storing the different types of data. These helpers are used by the `pull` commands to avoid code duplication.

### `push`

- **File**: `cmd/push/push.go`
- **Description**: Pushes pending changes from the local database to the BadgerMaps API. This command is not yet fully implemented.

### `server`

- **File**: `cmd/server/server.go`
- **Description**: Starts an HTTP server to listen for webhooks from BadgerMaps. This allows for real-time synchronization of data.
- **Configuration**: The server's configuration (host, port, webhook secret) is managed in `cmd/server/config.go`.
- **Middleware**: The `cmd/server/middleware.go` file contains middleware for verifying the webhook signature.

### `test`

- **File**: `cmd/test/test.go`
- **Description**: A command for testing the API and database connections.

### `version`

- **File**: `cmd/version/version.go`
- **Description**: Prints the version of the BadgerMaps CLI.

## Dependencies

The `cmd` modules depend on the `app`, `api`, and `database` packages to perform their tasks. The `app.State` is passed to each command, providing access to the configured API client and database interface.
