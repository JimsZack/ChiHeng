#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

./scripts/quality-gate.sh
go test -race -shuffle=on -count=1 ./...
go vet ./...
go mod verify
pnpm --dir frontend install --frozen-lockfile
pnpm --dir frontend run check
pnpm --dir frontend run test
pnpm --dir frontend run build

if command -v wails >/dev/null 2>&1; then
  wails build -clean
else
  echo "build: Wails CLI unavailable; install the pinned CLI before packaging" >&2
  exit 1
fi
