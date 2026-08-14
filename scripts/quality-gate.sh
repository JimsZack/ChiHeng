#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

scan_paths=()
for path in app.go main.go internal frontend/src build docs scripts; do
  [[ -e "$path" ]] && scan_paths+=("$path")
done

if ((${#scan_paths[@]} == 0)); then
  echo "quality: no product paths found" >&2
  exit 1
fi

failed=0

emoji_pattern='[\x{1F000}-\x{1FAFF}\x{2600}-\x{27BF}\x{2300}-\x{23FF}\x{FE0F}\x{1F1E6}-\x{1F1FF}]'
if rg --pcre2 -n "$emoji_pattern" "${scan_paths[@]}" \
  -g '!build/bin/**' -g '!build/icons/**' -g '!*.png' -g '!*.ico' -g '!*.icns'; then
  echo "quality: emoji found in product files" >&2
  failed=1
fi

if rg -n --glob '*.go' --glob '*.ts' --glob '*.tsx' \
  '\b(TODO|FIXME|HACK|XXX)\b' app.go main.go internal frontend/src 2>/dev/null; then
  echo "quality: unresolved core marker found" >&2
  failed=1
fi

secret_pattern="(?i)(api[_-]?key|secret|token|password)[[:space:]]*[:=][[:space:]]*['\"]?[A-Za-z0-9_+/=-]{12,}"
if rg -n "$secret_pattern" "${scan_paths[@]}" \
  -g '!*.test.*' -g '!*.md' -g '!scripts/quality-gate.sh'; then
  echo "quality: possible embedded credential found" >&2
  failed=1
fi

if rg -n '(sk-[A-Za-z0-9_-]{16,}|AKIA[0-9A-Z]{16}|gh[pousr]_[A-Za-z0-9]{20,})' \
  "${scan_paths[@]}" -g '!*.md' -g '!scripts/quality-gate.sh'; then
  echo "quality: credential-shaped value found" >&2
  failed=1
fi

if ((failed)); then
  exit 1
fi

echo "quality: static policy scans passed"
