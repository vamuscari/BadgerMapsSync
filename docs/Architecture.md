# Application Architecture

This document outlines the architecture of the BadgerMapsSync application, covering its package structure and the key design principles that guide its development.

## Overview

BadgerMapsSync is a Go application that provides both a Command-Line Interface (CLI) and a Graphical User Interface (GUI) for synchronizing data with the BadgerMaps API. It allows users to pull data into a local database, push local changes back to the API, and run a server to listen for webhooks.

## Core Technologies

-   **Go**: The primary programming language.
-   **Cobra**: A library for creating powerful modern CLI applications.
-   **Fyne**: A cross-platform GUI toolkit for Go.
-   **SQLite, PostgreSQL, MSSQL**: Supported database backends for local data storage.

## Package Structure

The application is organized into several packages, each with a distinct responsibility:

-   `main.go`: The application's entry point. It initializes the core `app` object and sets up the Cobra CLI commands.
-   `app`: Contains the central `App` struct, which holds the application's state, configuration, and core business logic. It acts as the orchestrator for all other packages.
-   `cmd`: Implements the various subcommands for the CLI (e.g., `pull`, `push`, `server`). These commands are thin wrappers that call the core logic in the `app` package.
-   `gui`: Contains the Fyne-based GUI. Like the `cmd` package, it provides a user-facing layer that interacts with the core `app` logic.
-   `database`: Provides a database abstraction layer. It includes a `DB` interface and concrete implementations for SQLite, PostgreSQL, and MSSQL. It is responsible for all database interactions, including schema management.
-   `api`: Contains the client for interacting with the BadgerMaps API.
-   `events`: Implements an event-driven system for application-level notifications (e.g., `PullComplete`, `ActionError`). It allows for decoupling different parts of the application.
-   `state`: A critical package for decoupling. It contains the `State` struct, which holds runtime state information, such as command-line flags (`Verbose`, `Debug`, `Quiet`).

## Architectural Principles & Patterns

### Dependency Flow

The architecture strictly follows a **unidirectional dependency flow**. High-level packages that manage business logic and user interaction (like `app`, `cmd`, `gui`) can depend on low-level packages that handle specific technical concerns (like `database`, `api`).

However, low-level packages are **never allowed to import high-level packages**. For example, the `database` package must not have any knowledge of the `app` or `gui` packages. This prevents import cycles and creates a more maintainable and testable codebase.

### Decoupling with the `state` Package

A key challenge in this architecture is providing low-level packages with necessary configuration or runtime information without violating the dependency rule. For instance, the `database` package's schema validation functions need to know if the application is running in `Verbose` or `Debug` mode to provide appropriate diagnostic output.

The solution is the `state` package:

1.  The `state.State` struct is a simple, lightweight data container with no dependencies on other application packages.
2.  The high-level `app` package owns and populates the `State` struct based on command-line flags or GUI settings.
3.  When the `app` package calls a function in a low-level package like `database`, it passes the `*state.State` object as a parameter.

This allows the `database` package to access the necessary information without needing to import the `app` package, thus avoiding a circular dependency.

### Event System vs. Diagnostic Logging

The application utilizes two distinct forms of communication:

1.  **Application-Level Eventing:** The `events` package provides a dispatcher for significant application events (e.g., `PullComplete`, `PushError`). This is used to trigger actions, update the GUI, or log major status changes. It is a high-level concern.

2.  **Diagnostic Logging:** For low-level, verbose output, such as the step-by-step process of validating a database schema, direct logging to the console (`fmt.Printf`) is used. This logging is explicitly guarded by flags (`Verbose`, `Debug`) passed down via the `state.State` object. This approach was chosen over the event system for these specific cases because this output is not a significant "event" for the application to act upon, but rather direct, immediate feedback to the user during a specific, isolated operation. Forcing this into the event system would have unnecessarily coupled the `database` package to the `events` package.
