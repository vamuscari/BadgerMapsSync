# Database Module

This document provides a detailed overview of the database module in the BadgerMaps CLI.

- [Back to Architecture](./ARCHITECTURE.md)
- [README](../README.md)

## Overview

The `database` package provides an abstraction layer for interacting with the local database. It is designed to be flexible and support multiple database backends.

## `database.DB` Interface

The core of the database module is the `database.DB` interface. This interface defines a set of methods that any database implementation must provide. This allows the rest of the application to interact with the database in a generic way, without needing to know the specifics of the underlying database technology.

The `database.DB` interface includes methods for:

- Getting the database type (`GetType`)
- Getting the database connection string (`DatabaseConnection`)
- Managing database settings (`GetDatabaseSettings`, `SetDatabaseSettings`, `PromptDatabaseSettings`)
- Checking for the existence of tables, views, procedures, and triggers
- Validating and enforcing the database schema (`ValidateSchema`, `EnforceSchema`)
- Testing the database connection (`TestConnection`)
- Dropping all tables (`DropAllTables`)
- Connecting to and closing the database (`Connect`, `Close`)
- Getting the underlying `sql.DB` object (`GetDB`)

## Supported Databases

The BadgerMaps CLI currently supports the following database backends:

- **SQLite**: A lightweight, file-based database. This is the default option and is suitable for most use cases.
- **PostgreSQL**: A powerful, open-source object-relational database system.
- **Microsoft SQL Server (MSSQL)**: A relational database management system developed by Microsoft.

## SQL Scripts

All SQL commands are stored in `.sql` files within the `database/<db_type>/` directories. This approach keeps the Go code clean and separates the application logic from the database-specific SQL.

The `sqlCommandLoader` function in `database/database.go` is responsible for loading these SQL scripts.

### Parameter Placeholders

All SQL commands use `?` as the parameter placeholder, which is compatible with the standard `database/sql` package. The database drivers automatically replace these placeholders with the correct syntax for the specific database backend.

## Schema Management

The database schema is managed through the `EnforceSchema` and `ValidateSchema` methods of the `DB` interface.

- **`EnforceSchema`**: This method creates all the necessary tables, views, and other database objects. It is typically called during the initial setup of the application.
- **`ValidateSchema`**: This method checks if the existing database schema matches the expected schema. It is used to ensure that the database is in a consistent state before the application starts.

## Adding a New Database Backend

To add support for a new database, you need to:

1.  Create a new struct that implements the `database.DB` interface.
2.  Add a new case to the `LoadDatabaseSettings` function in `database/database.go` to instantiate your new database struct.
3.  Add the necessary SQL scripts for your new database to a new subdirectory in the `database` directory.
