# Bloatscope

Disk tool that explains what AI and dev tools store and what's safe to remove. Read [description.md](description.md) for the product, [ARCHITECTURE.md](ARCHITECTURE.md) for the design, and [DEFINITIONS.md](DEFINITIONS.md) for the definition format.

## Safety rules (non-negotiable)

- Never delete anything. Cleaning means moving to quarantine (`internal/quarantine`), and only `purge` removes files from quarantine.
- Scanning is read-only. Nothing outside the bloatscope data directory and quarantine folders may be written.
- Never follow symlinks or junctions while walking.
- Never mark an entry `regenerable` or `redownloadable` without a source in `verified.source`. Default to `unknown`.
- `user-data` and `unknown` entries are never suggested for cleaning.
- Hardlink only entries with `dedupe: true`, only on the same volume, and only after a full hash match. Swap via temp name + rename.
- Tests that touch the filesystem use `t.TempDir()`, never the real home directory.

## Conventions

- Go, standard library first. Ask before adding a dependency.
- `--json` output is a stable contract. Add fields; don't rename or remove them.
- Windows is the primary target, but every path goes through the OS-specific expansion in `internal/defs`.

## Commands

```
go build ./cmd/bloatscope
go test ./...
go run ./cmd/bloatscope scan
```
