#!/usr/bin/env bash
set -euo pipefail

ACCOUNT_ID="${1:-174473766}"
BASE_URL="${BADGER_API_BASE:-https://badgerapis.badgermapping.com/api/2}"
API_KEY="${BADGER_API_KEY:-}"

if [[ -z "${API_KEY}" ]]; then
  echo "BADGER_API_KEY must be set in your environment" >&2
  exit 1
fi

ISO_TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

curl -sS -X POST "${BASE_URL%/}/appointments/" \
  -H "Authorization: Token ${API_KEY}" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "customer=${ACCOUNT_ID}" \
  --data-urlencode "comments=Met with Gainesville office to review pickup process (${ISO_TIMESTAMP})" \
  --data-urlencode "type=Onboarding" \
  --data-urlencode "log_datetime=${ISO_TIMESTAMP}" \
  --data-urlencode "extra_fields[Log Type]=Meeting" \
  --data-urlencode "extra_fields[Meeting Notes]=Discussed case volume and scheduled follow-up." |
  jq .
