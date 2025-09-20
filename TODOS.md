# TODOs

## Core Architecture Refactoring

### Event System

- [x] **Refactor the Event System:** Overhaul the current event system to be more scalable and structured, enabling features like enhanced webhook triggers.
    - [x] **Use String-Based Event Types:** Replace the `EventType` integer enum with a hierarchical, dot-separated string format (e.g., `pull.start`, `webhook.received.account.updated`). This allows any package to define new events without modifying a central file.
    - [x] **Introduce Structured Event Payloads:** Define specific structs for each event's payload instead of using a generic `interface{}`. These structs should implement a common interface.
    - [x] **Enhance Dispatcher with Wildcard Subscriptions:** Upgrade the `EventDispatcher` to allow listeners to subscribe to patterns (e.g., `pull.complete.*` or `*.accounts`), making listeners more powerful and reducing filtering boilerplate.
- [ ] **Enhanced Webhook Triggers:** Implement a system to trigger actions directly from incoming webhooks, built on the refactored event system.
    - [ ] Action configuration should support `source: webhook`, `object: <struct_type>`, and `event: <Create|Update|Delete>`.

## Core Functionality

- [ ] **Dry Run Mode:** Add a `--dry-run` flag to the `push` command to preview changes before applying them.
- [ ] **Conflict Resolution:** Define a strategy for handling sync conflicts (e.g., prompt user, last-write-wins).
- [ ] **Incremental Sync:** ~~Implement a mechanism to only fetch records that have changed since the last sync.~~
    - **Note:** The BadgerMaps API `GET /customers/` endpoint does not currently support a `since` parameter or return a `last_modified_date` field in the list view, making incremental sync for accounts not feasible at this time. A full sync is required.

## Server Customization

- [x] **Implement cron job support for scheduling tasks.**
    - [x] Allow scheduling of API calls.
    - [x] Allow scheduling of actions.
- [x] **Health Check Endpoint:** Add a `/health` endpoint to the server to verify its status and database connectivity.
- [x] **Webhook History and Replay:**
    - [x] Log all incoming webhook requests (headers and body) to a database table for auditing.
    - [x] Add a CLI command (`badgermaps server replay-webhook <id>`) to re-process a specific logged webhook.
    - [x] Include a configurable "catch-all" toggle to log all webhook traffic, even for events without defined actions.

## GUI Enhancements

### Omnibox Search

- [x] **Implement Omnibox for Accounts.**
    - [x] Accounts: Search by `AccountID` and `full_name`.
- [ ] Implement Omnibox for Check-ins, and Routes.
    - [ ] Check-ins: Search by `AccountID` and `full_name`. (Note: Check-in pulls by `AccountID`)
    - [ ] Routes: Search by `RouteID` and `route_name`.
- [ ] Add a configuration button (4 dots) next to each Omnibox.
    - [ ] The configuration should contain checkboxes to enable/disable specific search fields.
    - [ ] All fields should be enabled by default.
    - **Note:** The current API `SearchAccounts` endpoint does not support filtering by specific fields. This feature will be blocked until the API is updated.

### Explorer View

- [ ] Implement pagination for the data table.
- [ ] Add the ability to sort the table by clicking on a column header.

### Feature Parity

- [ ] **Test Command:** Add a "Test" tab or button to the GUI to run the API and database connection tests.
- [ ] **Version Information:** Display the application version in the GUI (e.g., in the "Home" or "About" tab).
- [ ] **Create Actions:** Add the ability to create new event actions from scratch in the "Actions" tab.
- [ ] **Replay Webhooks:** Add a feature to the "Server" or a new "Webhooks" tab to view the webhook log and replay specific webhooks.

## CLI Enhancements

- [ ] **View Pending Changes:** Add a `push list` command to the CLI to view pending changes, similar to the feature in the GUI's "Push" tab.

## Logging

- [x] **Implement a feature to write log/debug output to a file.**
    - [x] The log file location should be configurable.
    - [x] By default, the log file should be written to the current working directory (PWD).

## Testing & Developer Experience

- [ ] Add tests for saving configuration settings.
- [ ] **CI/CD Pipeline:** Set up a pipeline (e.g., GitHub Actions) to automatically run tests and build the application on commits.
