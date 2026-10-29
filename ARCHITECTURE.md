# Architecture

## Stack

**Go, single binary.**

- Fast directory walking and hashing, with easy cross-compilation for Windows, macOS, and Linux.
- One file to download, no runtime to install. That matters for a tool people try once on a whim.
- `os.Link` (hardlinks) and `os.Rename` (quarantine) work on NTFS, APFS, and ext4.
- The web UI and the MCP server can be served from the same binary later.

Dependencies stay minimal: a YAML parser for definitions, and an MCP SDK in phase 6.

## Layout

```
cmd/Cache Vent/     main.go: CLI entry (scan, dedupe, clean, restore, ui, mcp)
internal/scan/      walk the disk, sum sizes, match folders to definitions
internal/defs/      load and validate definitions
internal/dedupe/    find identical files, hardlink them
internal/quarantine/ move, restore, purge, manifest
definitions/        one YAML file per tool (community-editable)
```

New packages are added only when a phase needs them.

## Data flow

```
definitions/*.yaml ──► defs.Load ──► expand paths for this OS
                                          │
                                          ▼
                       scan.Run ──► per-definition sizes + last-used time
                                          │
                     ┌────────────────────┼────────────────────┐
                     ▼                    ▼                    ▼
              report (table/JSON)   dedupe.Find          history (JSONL)
                                          │
                                          ▼
                                   quarantine / hardlink
```

## Key decisions

**Scanning is read-only.** Phase 1 never writes anywhere except its own data directory.

**Don't follow symlinks or junctions.** Windows junctions can loop (e.g. `Application Data` → itself), and following links double-counts sizes. Hardlinked files are counted once per file ID.

**Size means bytes on disk, not file length.** Sparse files and compressed NTFS files otherwise show inflated numbers.

**Dedupe uses free hashes where possible.** Ollama blob filenames are `sha256-<hash>`, and Hugging Face LFS blobs are named by their SHA-256. Use those names to group candidates, then verify by hashing before linking. Other files: group by size → hash the first and last 1 MB → full SHA-256 only on matches.

**Hardlink rules:**
- Same volume only. Hardlinks can't cross drives, so the report must say so instead of failing silently.
- Only for read-only content types (model weights). Never for anything an app writes to in place.
- Replace via a temp name and rename, so a crash never leaves a missing file.

**Quarantine is a rename, not a copy.** The quarantine folder lives on the same volume as the item (`<volume>\.Cache Vent-quarantine\`), so moving is instant and costs no extra space. A `manifest.json` per item records the original path, size, date, and definition ID. Purging happens at the start of the next run. There's no background service.

**History is append-only JSONL.** One line per scan per definition: `{time, id, bytes}`. That's enough for "grew X this week." Move to SQLite only if queries get slow.

**Data directory:** `%LOCALAPPDATA%\Cache Vent` on Windows, `~/Library/Application Support/Cache Vent` on macOS, `$XDG_DATA_HOME/Cache Vent` on Linux.

## Output contract

`--json` output is a stable interface (used by the UI, the MCP server, and scripts):

```json
{
  "scanned_at": "2026-10-29T12:00:00Z",
  "items": [
    {
      "id": "huggingface-hub-cache",
      "name": "Hugging Face model cache",
      "path": "C:\\Users\\me\\.cache\\huggingface\\hub",
      "bytes": 50682871808,
      "safety": "redownloadable",
      "last_used": "2026-08-17T09:12:00Z",
      "duplicate_bytes": 17179869184
    }
  ]
}
```
