# Demo Data & Screenshots

This guide explains how to seed demo data and capture GUI screenshots used by documentation.

- [README](../README.md)

## 1) Prepare a Fresh SQLite Database

```bash
rm -f test.db
go build -o badgermaps
./badgermaps --gui
# In the GUI, set DB type = sqlite3 and DB path = ./test.db, then Initialize Schema.
```

Alternatively, from the CLI you can initialize schema by running setup, or by using test harness helpers.

## 2) Load Demo Data

Option A: small sample dataset (`docs/demo.sql`)

```bash
sqlite3 test.db < docs/demo.sql
```

Option B: bulk generator (`scripts/reseed_demo.sh`)

```bash
./scripts/reseed_demo.sh test.db
# Optional volume overrides:
# BULK_ACCOUNTS=500 BULK_ROUTES=120 CHECKINS_PER=3 SYNC_RUNS=80 ./scripts/reseed_demo.sh test.db
```

This seeds Accounts, Routes, AccountCheckins, and historical job data so Explorer and Sync Center appear populated.

## 3) Launch GUI and Capture Screenshots

```bash
./badgermaps --gui
```

Capture and save images under `assets/screenshots/`:

- `home.png` for Home/Dashboard
- `config.png` for Configuration
- `sync-pull.png` for Sync Center (Pull)
- `sync-push.png` for Sync Center (Push)
- `explorer-accounts.png` for Explorer with Accounts

These filenames are referenced by README.

## Optional: Generate High-Resolution Screenshots

Use the headless helper in `scripts/generate_screenshots.go`.

```bash
# 2x scale (default)
go run scripts/generate_screenshots.go

# 3x scale
SCALE=3 go run scripts/generate_screenshots.go
```

Outputs are written to `assets/screenshots/`. The helper also seeds `assets/screenshots/demo.db` (git-ignored).
