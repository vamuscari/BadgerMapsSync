# Repository Guidelines

## Project Structure & Module Organization
Source lives under `cli/` for Cobra commands, `app/` for shared business logic, and `gui/` for Fyne views. Persistence helpers and embedded SQL are in `database/`, while cross-cutting services (events, utilities) sit in `events/` and `utils/`. Assets, icons, and platform bundles reside in `assets/` and `dist/`. Tests accompany their packages as `_test.go` files; integration fixtures belong with the feature they exercise.

All database queries must live in the embedded SQL files under `database/<dialect>/`. Application code should obtain statements through `DB.GetSQL(...)` rather than inlining SQL strings, so that each backend stays in sync and schema changes remain declarative. Use positional `?` placeholders in every SQL asset so we stay compatible with Go's `database/sql` APIs across drivers.

## Build, Test, and Development Commands
Run `go build -o badgermaps` from the repo root to produce the CLI binary. Use `go test ./...` for the full suite or scope down (e.g. `go test ./cli/...`) during focused work, and clear stale build state with `go clean -cache` when test results look inconsistent. After every test run, build the application again to verify binaries stay healthy. Launch the GUI via `./badgermaps --gui` after building, or execute CLI subcommands directly with `./badgermaps sync`. For release bundles, invoke `./build.sh` to cross-compile macOS and Windows artifacts.

## Coding Style & Naming Conventions
Target Go 1.22 features and format every change with `gofmt` (or `goimports`). Group imports as stdlib, third-party, then local `badgermaps/...` packages. Use CamelCase for exported types/functions, camelCase for internal names, and ALL_CAPS only for constants. Keep structs public only when they represent API or config models; favor private fields for runtime state. Document public APIs or non-obvious logic with concise comments.

## Testing Guidelines
Rely on Go's `testing` package with table-driven cases where scenarios multiply. Name entry points `Test<Subject>` and arrange helpers with `_test.go` suffixes. When adding features, include regression coverage and run `go test -cover ./...` before sending patches. Prefer deterministic fixtures and exercise event emissions where behavior depends on them.

## Commit & Pull Request Guidelines
Follow the existing Conventional Commit style: `feat:`, `fix:`, `docs:`, and similar prefixes summarize intent. Keep messages in the imperative and reference issue IDs when relevant. Pull requests should describe scope, note affected commands or GUIs, and call out testing evidence (commands run, screenshots for UI). Ensure build artifacts are excluded, and update docs or configuration samples alongside code changes.
