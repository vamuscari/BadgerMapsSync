#!/usr/bin/env bash
set -euo pipefail

DB_PATH="${1:-test.db}"

echo "[reseed] Target DB: ${DB_PATH}"

if ! command -v sqlite3 >/dev/null 2>&1; then
  echo "[reseed] Error: sqlite3 CLI not found in PATH" >&2
  exit 1
fi

rm -f "$DB_PATH"

# Create schema in dependency-safe order (matches RequiredTables in code)
declare -a TABLES=(
  Accounts
  AccountCheckins
  AccountLocations
  AccountsPendingChanges
  AccountCheckinsPendingChanges
  Routes
  RouteWaypoints
  UserProfiles
  DataSets
  DataSetValues
  FieldMaps
  Configurations
  SyncHistory
  CommandLog
  WebhookLog
)

for t in "${TABLES[@]}"; do
  sql="database/sqlite3/Create${t}Table.sql"
  if [[ -f "$sql" ]]; then
    echo "[reseed] Creating table: ${t}"
    sqlite3 "$DB_PATH" < "$sql"
  else
    echo "[reseed] Warning: missing schema file: $sql" >&2
  fi
done

# Optional view
if [[ -f database/sqlite3/CreateAccountsWithLabelsView.sql ]]; then
  echo "[reseed] Creating view: AccountsWithLabels"
  sqlite3 "$DB_PATH" < database/sqlite3/CreateAccountsWithLabelsView.sql
fi

# Seed baseline config + field maps if present
if [[ -f database/sqlite3/InsertConfigurations.sql ]]; then
  echo "[reseed] Inserting configurations"
  sqlite3 "$DB_PATH" < database/sqlite3/InsertConfigurations.sql
fi
if [[ -f database/sqlite3/InsertFieldMaps.sql ]]; then
  echo "[reseed] Inserting field maps"
  sqlite3 "$DB_PATH" < database/sqlite3/InsertFieldMaps.sql
fi

# Load demo data
if [[ -f docs/demo.sql ]]; then
  echo "[reseed] Loading demo data"
  sqlite3 "$DB_PATH" < docs/demo.sql
else
  echo "[reseed] Warning: docs/demo.sql not found; skipping demo load" >&2
fi

# Optional bulk volume (env overrides)
BULK_ACCOUNTS="${BULK_ACCOUNTS:-100}"
BULK_ROUTES="${BULK_ROUTES:-30}"
CHECKINS_PER="${CHECKINS_PER:-2}"
SYNC_RUNS="${SYNC_RUNS:-25}"

echo "[reseed] Bulk seeding: Accounts=$BULK_ACCOUNTS Routes=$BULK_ROUTES Checkins/Account=$CHECKINS_PER SyncRuns=$SYNC_RUNS"

# Generate bulk INSERTs (single transaction)
tmp_sql="$(mktemp)"
{
  echo "BEGIN;"

  # Accounts
  start_acct=$(sqlite3 "$DB_PATH" "SELECT IFNULL(MAX(AccountId),0)+1 FROM Accounts;")
  for ((i=0;i< BULK_ACCOUNTS;i++)); do
    id=$((start_acct+i))
    fn="Demo"
    ln="User$id"
    email="demo${id}@example.com"
    phone="+1-555-1$(printf '%04d' "$id")"
    cust="DEMO-$(printf '%04d' "$id")"
    owner=$(( (id % 3) + 1 ))
    case $owner in 1) own="alice@company.com";; 2) own="bob@company.com";; 3) own="carol@company.com";; esac
    echo "INSERT INTO Accounts (AccountId, FirstName, LastName, FullName, Email, PhoneNumber, CustomerId, Notes, AccountOwner) VALUES ($id, '$fn', '$ln', '$fn $ln', '$email', '$phone', '$cust', 'Demo generated', '$own');"
  done

  # Checkins per account
  start_chk=$(sqlite3 "$DB_PATH" "SELECT IFNULL(MAX(CheckinId),1000)+1 FROM AccountCheckins;")
  chk=$start_chk
  for ((i=0;i< BULK_ACCOUNTS;i++)); do
    acct=$((start_acct+i))
    for ((c=1;c<= CHECKINS_PER;c++)); do
      ts="2025-01-10T0$(( (c%9)+1 )):$(printf '%02d' $(( (acct+c)%60 )))":00Z
      echo "INSERT INTO AccountCheckins (CheckinId, CrmId, AccountId, LogDatetime, Type, Comments, CreatedBy) VALUES ($chk, 'CHK-$chk', $acct, '$ts', 'Visit', 'Auto-generated checkin', 'system');"
      chk=$((chk+1))
    done
  done

  # Routes
  start_route=$(sqlite3 "$DB_PATH" "SELECT IFNULL(MAX(RouteId),100)+1 FROM Routes;")
  for ((i=0;i< BULK_ROUTES;i++)); do
    rid=$((start_route+i))
    rdate="2025-01-12"
    echo "INSERT INTO Routes (RouteId, Name, RouteDate, Duration, StartAddress, DestinationAddress, StartTime) VALUES ($rid, 'Route $rid', '$rdate', $((30 + (rid%90))), '100 First St', '200 Second St', '08:00');"
  done

  # Sync History
  for ((i=1;i<= SYNC_RUNS;i++)); do
    corr="bulk-$(printf '%04d' "$i")"
    dir=$([ $((i%2)) -eq 0 ] && echo pull || echo push)
    stat=$([ $((i%5)) -eq 0 ] && echo failed || echo completed)
    items=$(( (i*3) % 50 ))
    errs=$([ "$stat" = failed ] && echo 1 || echo 0)
    started="2025-01-15T08:$(printf '%02d' $((i%60)))":00Z
    dur=$((5 + (i%20)))
    echo "INSERT INTO SyncHistory (CorrelationId, RunType, Direction, Source, Initiator, Status, ItemsProcessed, ErrorCount, StartedAt, CompletedAt, DurationSeconds, Summary, Details) VALUES ('$corr','manual','$dir','demo','user','$stat',$items,$errs,'$started','$started',$dur,'Bulk $dir run #$i','Auto-generated demo run');"
  done

  echo "COMMIT;"
} > "$tmp_sql"

echo "[reseed] Bulk INSERTs -> $tmp_sql"
sqlite3 "$DB_PATH" < "$tmp_sql"
rm -f "$tmp_sql"

echo "[reseed] Row counts:"
sqlite3 "$DB_PATH" "SELECT 'Accounts', COUNT(*) FROM Accounts UNION ALL SELECT 'Routes', COUNT(*) FROM Routes UNION ALL SELECT 'AccountCheckins', COUNT(*) FROM AccountCheckins UNION ALL SELECT 'SyncHistory', COUNT(*) FROM SyncHistory;" | awk -F'|' '{printf "  %-16s %s\n", $1, $2}'

echo "[reseed] Done."
