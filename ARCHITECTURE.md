# Architecture

## Stack

**Go, single binary.**

- Fast directory walking and hashing, with easy cross-compilation for Windows, macOS, and Linux.
- One file to download, no runtime to install. That matters for a tool people try once on a whim.
- `os.Link` (hardlinks) and `os.Rename` (quarantine) work on NTFS, APFS, and ext4.
- The web UI and the MCP server can be served from the same binary later.

Dependencies stay minimal: a YAML parser for definitions, WebView2 bindings for the Windows window, and an MCP SDK in phase 6.

## Layout

```
cmd/cachevent/       main.go: CLI entry (scan, clean, quarantine, restore, purge)
internal/defs/       load and validate definitions, OS path expansion, data dir
internal/scan/       walk the disk, sum bytes on disk, match folders to definitions
internal/envs/       Python environment analyzer: owner project, orphans, packages
internal/clean/      decide what may be quarantined (only cleanable items from the last scan)
internal/quarantine/ move, restore, purge, manifest
definitions/         one YAML file per tool (community-editable), embedded in the binary
```

`cmd/cachevent/app.go` holds the scan → save → plan → move flow, which the CLI and the UI both use, so they can't drift apart on safety. `ui.go` + `ui/index.html` are the desktop app: an HTTP server on 127.0.0.1 with a random port, shown in a native window through WebView2 (`window_windows.go`, via `github.com/jchv/go-webview2`, pure Go with no cgo), falling back to an Edge/Chrome `--app` window with its own profile where WebView2 is missing. Every request must carry a per-launch random token (passed in the URL fragment) and the exact `127.0.0.1:port` Host header, so neither other websites nor DNS rebinding can drive it. The page is vanilla JS with no build step, and it inserts every file name as text, never as HTML. Links leave the app through `/api/open`, which only opens https sources the scan cites, in the default browser.

Planned: `internal/dedupe/` (phase 2). New packages are added only when a phase needs them.

**Environment analyzer.** Some things can't be described in YAML: who owns a virtual environment, and whether that owner still exists. `internal/envs` handles them and produces items shaped like definition items, with these stable IDs: `python-venv` (in-project, any folder with `pyvenv.cfg`), `pipenv-env`, `poetry-env`, `conda-env`. The verdict is always for the whole environment. Single packages are never removed, because that breaks whatever depends on them. Package sizes come from each `*.dist-info/RECORD`, and the same `name==version` found in several environments is reported as shared.

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

**Scanning engine.** On Windows each folder is read with `GetFileInformationByHandleEx(FileIdBothDirectoryInfo)`. One call returns ~64 KB of entries, each carrying its allocation size (bytes on disk), times, attributes, reparse tag and file ID, so sizing needs no per-file system call and no admin rights. Folders are read in parallel by at most about 4 walkers per CPU: a subfolder gets a new goroutine only when a slot is free, otherwise the current walker descends into it itself, so nothing ever waits on a semaphore and memory stays flat (a goroutine per folder held ~400 MB on a 1M-file drive). Listing buffers are pooled and names are decoded in place. The search for projects checks siblings and `pyvenv.cfg` against the listing already in memory instead of asking the disk. Numbers are in [BENCHMARKS.md](BENCHMARKS.md). Reading the NTFS MFT directly (WizTree-style) would be faster again but needs admin, so it isn't used.

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
