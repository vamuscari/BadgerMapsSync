# Single Pull Response Capture

This document describes how the CLI should support exporting the raw payload when running
single-resource pull commands (account, route, check-in, profile) while still storing the
record in the local database through the existing pull helpers.

## Goals

- Allow `badgermaps pull account <id>` (and similar) to save the fetched resource as JSON.
- Keep database persistence and event emissions exactly as they are today.
- Mirror the CLI test harness’ `--save` behavior, but scoped to individual pull commands.

## High-Level Flow

1. User runs a single pull command with new flags:
   - `--save-response` (boolean): opts into saving with a timestamped default filename.
   - `--response-file <path>`: explicit output target (use `-` for stdout).
2. The CLI presenter invokes the corresponding pull helper, which now returns
   `(payload, error)` instead of just `error`.
3. On success, the presenter routes the payload to a shared response-save helper that:
   - Marshals to indented JSON.
   - Writes to the requested destination (creating directories as needed).
   - Emits CLI events so the user sees where the file landed.
4. The pull helper continues to:
   - Dispatch `pull.start`, `pull.complete`, and `pull.error` events.
   - Store the resource via existing `Store*` functions.

## Components & Responsibilities

| Component | Responsibility |
|-----------|----------------|
| `app/pull/pull_helpers.go` | Return the fetched model (`*api.Account`, `*api.Checkin`, `*api.Route`, `*api.UserProfile`) alongside the existing error so callers can reuse the payload. No change to DB writes or events. |
| `cli/pull/pull.go` | Define new flags on `pull account`, `pull route`, `pull checkin`, and `pull profile`. |
| `cli/pull/presenter.go` | Pass the flag values when calling pull helpers, and invoke the response-save helper if requested. |
| `cli/pull/response_saver.go` *(new)* | Contain reusable logic for marshaling payloads, choosing filenames (e.g., `pull-account-123-20240206-153000.json`), handling stdout, and logging outcomes. |
| `README.md` | Document the new flags in the CLI usage section with an example. |
| Tests | Update `cli/pull/pull_test.go` to assert the new flag behavior, including writing a file and keeping DB state unchanged. Existing helper tests should verify the new return values. |

## Filename & Save Rules

- Default directory: `./pull-responses/`.
- Default filename template: `pull-<resource>-<id>-<timestamp>.json`.
- When `--response-file -` is used, write the pretty JSON to stdout.
- When directories in `--response-file` do not exist, create them with `0755`.
- Overwrite existing files to keep the workflow deterministic.

## Error Handling

- Saving failures should not mask a successful pull; surface them as warnings and
  return the error from the command so CI/users know to re-run.
- If the pull itself fails, skip saving and return the pull error unchanged.

## Testing

- Extend the existing CLI pull test to cover `--save-response` and verify the file content.
- Add table-driven tests for the response saver helper (stdout, custom paths, default naming).
- Run `go test ./cli/pull` and `go test ./app/pull` after the change, using a local `GOCACHE`
  if the default cache directory is sandboxed.

## Rollout Notes

- Scheduler, GUI, and server integrations can remain untouched; they will ignore the new
  helper return values until they opt in.
- Because the CLI is the only consumer of the save flags, no config migrations are required.
