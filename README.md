# BadgerMapsSync

BadgerMapsSync is a command-line interface (CLI) and graphical user interface (GUI) for interacting with the BadgerMaps API. It allows you to synchronize data between the BadgerMaps API and a local database, providing a powerful tool for managing customer accounts, routes, check-ins, and user profiles.

## Features

-   **Two-Way Data Sync**: Pull data from the BadgerMaps API to a local database and push local changes back to the API.
-   **Multiple Database Backends**: Supports SQLite, PostgreSQL, and Microsoft SQL Server.
-   **Webhook Server**: Run in server mode to listen for real-time updates from BadgerMaps webhooks.
-   **Event-Driven Actions**: Configure custom actions to be triggered by specific events (e.g., `PullComplete`, `PushComplete`).
-   **Cross-Platform GUI**: A user-friendly graphical interface built with Fyne for managing data and configurations.
-   **Interactive Setup**: An interactive configuration wizard to get you started quickly.

## Supported Databases

-   **SQLite**: Default, lightweight, and file-based.
-   **PostgreSQL**: Powerful, open-source object-relational database.
-   **Microsoft SQL Server (MSSQL)**: Enterprise-grade relational database.

## GUI

The application includes a graphical user interface (GUI) built with the Fyne toolkit, providing a user-friendly way to interact with its features.

### GUI Features

-   **Pull Tab**: Pull data from the BadgerMaps API, either all at once or by specific IDs.
-   **Push Tab**: Push local changes to the BadgerMaps API.
-   **Events Tab**: View and manage event-driven actions.
-   **Explorer Tab**: A database explorer to view the contents of the local database.
-   **Configuration Tab**: Configure API credentials, database settings, and other application settings.
-   **Debug Tab**: View debug information.
-   **Log View**: See real-time log messages from the application.
-   **Details View**: View details about selected items.

## Building and Running

### Building the Project

To build the project, you need to have Go installed. Then, run the following command from the project's root directory:

```bash
go build -o badgermaps
```

This will create an executable file named `badgermaps` in the project's root directory.

### Running the Project

To run the project, you can use the following command:

```bash
./badgermaps [command]
```

Replace `[command]` with one of the available commands. You can see the list of available commands by running:

```bash
./badgermaps --help
```

To run the GUI, use the `gui` command:

```bash
./badgermaps gui
```

### Running Tests

To run the tests, you can use the following command:

```bash
go test ./...
```

## Development Conventions

### Building After Changes

After making any changes to the code, it is recommended to build the project to ensure that the changes have not introduced any compilation errors.

```bash
go build -o badgermaps
```

### Testing Conventions

Before building the project, it is recommended to run the tests to ensure that the changes have not introduced any regressions.

```bash
go test ./...
```

### Shared Logic

To avoid code duplication, both the `gui` and `cmd` packages should utilize shared helper methods and data structures from the `app` package. This ensures that core business logic is decoupled from the user interface.

### Configuration Management

The project follows a modular approach to configuration management. Each major component (e.g., `api`, `database`, `server`) is responsible for managing its own configuration settings. The `gui` and `cmd` packages should not set configuration keys directly but should interact with the configuration through the `app.App` instance.

## License

This project is licensed under the terms of the MIT License. See the [LICENSE](LICENSE) file for more details.