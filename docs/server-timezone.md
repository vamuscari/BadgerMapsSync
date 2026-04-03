# Server and Timezone Guide

This document explains the current server scheduling and timezone behavior, including recent updates to job scheduling, runtime visibility, and sync history storage.

- [README](../README.md)

## What Changed

The server and scheduler now support explicit timezone handling end-to-end:

- A global server timezone can be configured with `server.timezone` (IANA format, for example `America/New_York`).
- Each scheduled job can optionally define its own timezone override.
- Sync history now stores timezone context for start and completion timestamps.
- Server runtime now exposes a jobs snapshot endpoint used by the GUI Server tab.

## Server Timezone Configuration

`server.timezone` is optional. When present, it must be a valid IANA timezone name.

Example:

```yaml
server:
  host: localhost
  port: 8080
  timezone: America/New_York
  tls_enabled: false
  log_requests: true
```

Ways to configure:

- GUI: Server tab, `Global Timezone (IANA)`.
- CLI: `./badgermaps server setup` prompts for `Global Timezone (IANA, optional)`.
- Direct YAML edit: set `server.timezone` in `config.yaml`.

If you change timezone while the server is already running, restart the server to apply scheduler timezone changes.

## Scheduled Jobs Timezone Resolution

For each scheduled job, effective timezone is resolved in this order:

1. Job-level `timezone` override (if set)
2. Global `server.timezone` (if set)
3. Host local timezone (`time.Local`)

This applies to:

- Cron trigger evaluation
- Next-run calculations shown in the app

## Cron Expression Format

The scheduler accepts both legacy five-field and seconds-aware six-field cron expressions:

- Five-field: `minute hour day-of-month month day-of-week`
- Six-field: `second minute hour day-of-month month day-of-week`

Example valid expression:

`0 0 20 * * *`

Another valid legacy expression:

`0 20 * * *`

## Runtime Jobs Visibility

The server exposes internal local-only endpoints for queue/activity visibility:

- `GET /internal/jobs`
- `GET /internal/jobs/{id}`
- `GET /internal/activity`

The GUI Server tab uses these endpoints to show active, queued, and recent jobs. Access is restricted to local requests.

## Sync History Timezone Fields

`SyncHistory` now stores timezone context alongside UTC timestamps:

- `StartedAt` (UTC timestamp)
- `StartedAtTimezone` (IANA timezone string)
- `CompletedAt` (UTC timestamp)
- `CompletedAtTimezone` (IANA timezone string)

This schema is supported across SQLite, PostgreSQL, and MSSQL, including column migrations for existing databases.

## Notes for Operators

- Scheduled jobs and action/job enabled states are persisted immediately to config storage.
- Scheduled job edits are blocked while the server process is running. Stop the server first, then add/edit/pause/resume/delete jobs.
- Legacy five-field cron schedules remain supported for backward compatibility, and six-field schedules with seconds are also supported.
- After job/schedule/timezone edits, start (or restart) the server so the scheduler loads the updated persisted jobs.
- Invalid timezone values are rejected during config and job save operations.
