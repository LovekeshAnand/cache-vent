#!/bin/sh
# End-to-end check of an installed cachevent on macOS or Linux, the way a user runs it.
# Uses a throwaway HOME, so nothing real is scanned, moved or deleted.
#   sh scripts/e2e.sh [path/to/cachevent]
set -eu
CV="${1:-cachevent}"
command -v "$CV" >/dev/null 2>&1 || [ -x "$CV" ] || { echo "cachevent not found: $CV" >&2; exit 1; }
CV=$(command -v "$CV")

# The fake home lives in its own variable: cleanup must never depend on $HOME.
FAKE_HOME=$(mktemp -d)
case "$FAKE_HOME" in /tmp/* | /var/folders/* | /private/var/folders/* | "${TMPDIR:-/tmp}"*) ;; *) echo "unexpected temp dir $FAKE_HOME" >&2; exit 1 ;; esac
export HOME="$FAKE_HOME"
unset XDG_DATA_HOME XDG_CACHE_HOME || true
trap 'rm -rf "$FAKE_HOME"' EXIT
fail() { echo "FAIL: $*" >&2; exit 1; }
ok() { echo "ok   $*"; }
sum() { (cd "$1" && find . -type f -exec cat {} + | (sha256sum 2>/dev/null || shasum -a 256) | cut -d' ' -f1); }
sizeof() { python3 -c "import json,sys; d=json.load(open(sys.argv[1])); print(sum(i['bytes'] for i in d['items'] if i['id']==sys.argv[2]))" "$@"; }

"$CV" version | grep -q '^cachevent ' || fail "version"
ok "runs: $("$CV" version)"

P="$HOME/code/app"
mkdir -p "$P/node_modules/left-pad" "$HOME/.claude/projects"
echo '{}' > "$P/package.json"; echo '{}' > "$P/package-lock.json"
head -c 300000 /dev/urandom > "$P/node_modules/left-pad/index.js"
echo chat > "$HOME/.claude/projects/session.jsonl"
touch -t 202401010000 "$P/node_modules" "$P/node_modules/left-pad" "$P/node_modules/left-pad/index.js"
before=$(sum "$P/node_modules")

"$CV" scan --json --all --root "$HOME/code" > "$HOME/scan.json" 2>/dev/null
[ "$(sizeof "$HOME/scan.json" node-modules)" -ge 300000 ] || fail "scan did not size node_modules"
grep -q '"claude-code-sessions"' "$HOME/scan.json" || fail "scan missed Claude Code history"
ok "scan finds and sizes items"

if "$CV" clean --yes "$HOME/.claude/projects" >/dev/null 2>&1; then fail "cleaned user data"; fi
[ -f "$HOME/.claude/projects/session.jsonl" ] || fail "user data moved"
ok "refuses to clean your data"

"$CV" clean --yes "$P/node_modules" >/dev/null
[ ! -e "$P/node_modules" ] || fail "clean did not move node_modules"
id=$("$CV" quarantine | awk '/^node-modules-/ {print $1; exit}')
[ -n "$id" ] || fail "item not listed in quarantine"
ok "moves to quarantine ($id)"

"$CV" restore "$id" >/dev/null
[ "$(sum "$P/node_modules")" = "$before" ] || fail "restore is not byte-identical"
ok "restores byte-identical"

"$CV" clean --yes "$P/node_modules" >/dev/null
"$CV" purge --all --yes >/dev/null
[ -z "$("$CV" quarantine | awk '/^node-modules-/')" ] || fail "purge left the item"
ok "purge deletes only what is in quarantine"

log="$HOME/ui.log"
"$CV" ui --no-window > "$log" 2>&1 &
pid=$!
for _ in 1 2 3 4 5 6 7 8 9 10; do grep -q '^http' "$log" && break; sleep 0.5; done
url=$(grep -m1 '^http' "$log") || fail "ui did not print its address"
base=${url%%#*}; token=${url#*#}
curl -fsS "$base" -o "$HOME/page.html" && grep -q '<title>Cache Vent</title>' "$HOME/page.html" || fail "ui page"
[ "$(curl -s -o /dev/null -w '%{http_code}' "${base}api/state")" = 403 ] || fail "api answered without the token"
curl -fsS -H "X-Token: $token" "${base}api/state" | grep -q '"quarantine"' || fail "api state with token"
kill "$pid" 2>/dev/null || true
ok "desktop interface serves the page and guards its API"

echo "all checks passed"
