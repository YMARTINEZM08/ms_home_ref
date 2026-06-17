#!/usr/bin/env bash
# Capture golden-contract fixtures from digital_bff (source of truth) and ms_home
# for the same requests, into test/contract/fixtures/<case>.{golden,actual}.json.
#
# Usage:
#   DIGITAL_BFF=https://web-bff.qa.example.com \
#   MS_HOME=https://ms-home.qa.example.com \
#   AUTH="Bearer <token>"   # optional, for logged-in cases
#   ./scripts/capture-fixtures.sh
#
# Then diff:  go test ./test/contract -run HomeParity
set -euo pipefail

: "${DIGITAL_BFF:?set DIGITAL_BFF base URL}"
: "${MS_HOME:?set MS_HOME base URL}"
AUTH="${AUTH:-}"

DIR="$(cd "$(dirname "$0")/.." && pwd)/test/contract/fixtures"
mkdir -p "$DIR"

# case|path|extra curl args
CASES=(
  "web-home-anon|/content/page/es-mx/|"
  "pocket-home-anon|/content/screen/es-mx/home|"
  "web-home-preview|/content/page/es-mx/|-H x-preview:1"
  "web-home-logged|/content/page/es-mx/|-H Authorization:${AUTH}"
)

for entry in "${CASES[@]}"; do
  IFS='|' read -r name path extra <<<"$entry"
  [[ "$extra" == *"Authorization:"* && -z "$AUTH" ]] && { echo "skip $name (no AUTH)"; continue; }
  echo "capturing $name ..."
  # shellcheck disable=SC2086
  curl -fsS $extra "$DIGITAL_BFF$path" >"$DIR/$name.golden.json"
  # shellcheck disable=SC2086
  curl -fsS $extra "$MS_HOME$path"      >"$DIR/$name.actual.json"
done

echo "done → $DIR"
echo "run: go test ./test/contract -run HomeParity"
