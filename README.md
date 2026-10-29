# Cache Vent

See what every AI and dev tool is storing on your disk, what it is, and what's safe to remove. Read [description.md](description.md) for the why.

```
go build ./cmd/cachevent

cachevent scan                    # home folder + known tool locations
cachevent scan --root D:\         # also search another drive for projects and environments
cachevent scan --json             # stable machine-readable output
cachevent clean --dry-run PATH    # PATH from the scan; only items marked * are accepted
cachevent clean PATH
cachevent quarantine              # what's in quarantine
cachevent restore ID
cachevent purge --all             # the only command that deletes
```

**Desktop app.** Same binary, about 7 MB, no installer or runtime:

```
go build -ldflags "-s -w -H windowsgui -X main.defaultCmd=ui" -o CacheVent.exe ./cmd/cachevent
```

Double-click `CacheVent.exe` (or run `cachevent ui`). It serves a local page on 127.0.0.1 and opens it in an Edge/Chrome app window, and it exits when you close that window. `cachevent ui --no-window` prints the address for any browser.

Nothing is deleted outright: `clean` moves items to a quarantine folder on the same drive, and they're purged after 14 days. `user-data` and `unknown` items are never accepted by `clean`.

Docs: [ARCHITECTURE.md](ARCHITECTURE.md) · [DEFINITIONS.md](DEFINITIONS.md) (how to add a tool)
