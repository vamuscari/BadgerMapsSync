# Server and Timezone Guide

This document explains the current server scheduling and timezone behavior, including recent updates to job scheduling, runtime visibility, and sync history storage.

- [README](../README.md)

## What Changed

The server and scheduler now support explicit timezone handling end-to-end:

- A global server timezone can be configured with `server.timezone` (IANA format, for example `America/New_York`).
- Each scheduled job can optionally define its own timezone override.
- Incoming webhook routes are protected by HMAC signature validation when webhooks are enabled.
- Internal runtime endpoints require bearer authentication and local-source address checks.
- Legacy `SyncHistory` rows (when that table exists) can store timezone context for start and completion timestamps.
- Server runtime exposes queue/activity endpoints used by the GUI Sync Center and the right-pane `Jobs` monitor.

## Server Timezone Configuration

`server.timezone` is optional. When present, it must be a valid IANA timezone name.

Example:

```yaml
server:
  host: localhost
  port: 8080
  timezone: America/New_York
  tls_enabled: false
  webhook_secret: "<set-a-shared-secret>"
  internal_api_token: "<generated-or-set-token>"
  log_requests: true
```

Ways to configure:

- GUI: Configuration tab, `Global Timezone (IANA)`.
- CLI: `./badgermaps server setup` prompts for `Global Timezone (IANA, optional)`.
- Direct YAML edit: set `server.timezone` in `config.yaml`.

If you change timezone while the server is already running, restart the server to apply scheduler timezone changes.

## Server Security Configuration

`server.webhook_secret` and `server.internal_api_token` secure server endpoints:

- `server.webhook_secret` is required when webhook routes are enabled (`account_create` and/or `checkin`).
- `server.internal_api_token` protects `/internal/*` endpoints.
- If `server.internal_api_token` is missing, the app generates one and persists it to config on load.

Ways to configure:

- GUI: Configuration tab, Server section (`Webhook Secret`, `Internal API Token`, token `Regenerate` action).
- CLI: `./badgermaps server setup` prompts for both values.
- Direct YAML edit: set `server.webhook_secret` and `server.internal_api_token`.

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

The server exposes internal endpoints for queue/activity visibility and command submission:

- `POST /internal/jobs/sync`
- `GET /internal/jobs`
- `GET /internal/jobs/{id}`
- `POST /internal/scheduled-jobs/run`
- `GET /internal/activity`

The runtime payload includes:

- Per-job `current_action` when a sync job is actively executing a stage.
- Activity-level `active_job_action` for quick status display.

The GUI Sync Center uses these endpoints to show active, queued, and recent jobs. The right-pane slide-out `Jobs` section uses the same endpoint and focuses on active and queued jobs with live action updates while running. Access requires both:

- `Authorization: Bearer <server.internal_api_token>`
- A local source address (loopback or a local interface address).

## Sync History Timezone Fields

`SyncHistory` is a legacy table. When present, it stores timezone context alongside UTC timestamps:

- `StartedAt` (UTC timestamp)
- `StartedAtTimezone` (IANA timezone string)
- `CompletedAt` (UTC timestamp)
- `CompletedAtTimezone` (IANA timezone string)

`JobLog` is the primary runtime/job history table. `SyncHistory` may still exist in older databases and is used for optional backfill into `JobLog`.

## Notes for Operators

- Scheduled jobs and action/job enabled states are persisted immediately to config storage.
- Scheduled job edits are blocked while the server process is running. Stop the server first, then add/edit/pause/resume/delete jobs.
- Legacy five-field cron schedules remain supported for backward compatibility, and six-field schedules with seconds are also supported.
- After job/schedule/timezone edits, start (or restart) the server so the scheduler loads the updated persisted jobs.
- Invalid timezone values are rejected during config and job save operations.
