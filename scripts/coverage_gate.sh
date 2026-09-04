#!/usr/bin/env bash
# Fails when total Go statement coverage in a profile is below the floor.
#   scripts/coverage_gate.sh coverage.out scripts/coverage_floor.txt
set -euo pipefail
profile="${1:?coverage profile}"
floor_file="${2:?floor file}"
floor=$(tr -d '[:space:]%' < "$floor_file")
total=$(go tool cover -func="$profile" | awk '/^total:/ {gsub("%","",$3); print $3}')
echo "coverage: total=${total}% floor=${floor}%"
awk -v t="$total" -v f="$floor" 'BEGIN { if (t + 0 < f + 0) { print "FATAL: coverage " t "% is below the floor " f "%"; exit 1 } }'
echo "OK: coverage at or above floor"
