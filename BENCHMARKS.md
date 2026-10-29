# Benchmarks

All numbers are from one machine, measured on 2026-10-29. They are reproducible with the commands below, but your results will differ with disk, file count and antivirus.

| | |
|---|---|
| CPU | Intel Core i7-11850H, 8 cores / 16 threads |
| Memory | 16 GB |
| Disk | Micron 512 GB NVMe SSD, NTFS: `C:` 150 GB (147 GB used), `D:` 326 GB (293 GB used) |
| OS | Windows 11 Pro |
| Build | Go 1.27, `go build ./cmd/cachevent` |

Every run is warm: one untimed run first, then the median of three. Time is wall clock for the whole process, and memory is its peak working set, sampled every 20 ms.

## Real drives

The baseline is the standard way to size a tree in Go: a single-threaded `filepath.WalkDir` that calls `Info()` on every file and sums sizes. Like Cache Vent, it doesn't follow links. It does strictly less work than Cache Vent, which also matches 26 tool definitions, finds projects and virtual environments, reads `RECORD` files for package sizes and de-duplicates hardlinks.

| Scan | Folders | Files | Time | Peak memory |
|---|---:|---:|---:|---:|
| `filepath.WalkDir`, all of `D:` | 210,450 | 1,182,191 | 8.68 s | 15 MB |
| **`cachevent scan --root D:\`** | 215,501 | (same tree) | **2.53 s** | **41 MB** |
| **`cachevent scan --all-drives`** (`C:` + `D:`) | 231,033 | | **2.85 s** | **42 MB** |
| **`cachevent scan`** (home folder) | 19,576 | | **0.38 s** | **29 MB** |

On the same drive, Cache Vent is **3.4× faster** than the baseline while doing more work. Folder counts differ slightly because Cache Vent also measures the known tool locations (for example `%LOCALAPPDATA%`) on top of the root it searches.

`--all-drives` deliberately skips `C:\Windows`, `Program Files`, `ProgramData` and other system folders, which hold no developer caches. That's why a whole `C:` costs it about 0.4 s, while the baseline needs 18.9 s to walk all 780,777 files on `C:`. That comparison isn't like-for-like, so it isn't in the table.

What the all-drives scan found on this machine: 1,089 items, 39.5 GB, of which 30.3 GB is verified safe to remove.

## Synthetic tree (reproducible anywhere)

[`BenchmarkWalk`](internal/scan/scan_test.go) builds 100 projects, each with a lockfile and a `node_modules` holding 60 packages × 5 files: about 12,200 folders and 30,000 files. It then compares a full scan with the same `filepath.WalkDir` baseline.

```
go test ./internal/scan -run XXX -bench Walk -benchmem -count 5
```

| | Time / op | Allocated / op | Allocations / op |
|---|---:|---:|---:|
| `cachevent` | **145 ms** | 20 MB | 181k |
| `stdlib-walkdir` | 479 ms | 23 MB | 237k |

## Desktop app

| | |
|---|---|
| Executable | 7.9 MB, single file, no installer or runtime |
| Launch to window | 0.8 s |
| Memory, idle after launch | 55 MB app process + 173 MB for WebView2 (6 processes, shared with the OS component) |

## Why it's fast

- **One system call per ~64 KB of folder entries.** On Windows, `GetFileInformationByHandleEx(FileIdBothDirectoryInfo)` returns each entry's allocated size, times, attributes, reparse tag and file ID in a single batch. Most walkers make one extra `stat` call per file.
- **Bounded parallelism that never blocks.** Up to about 4 walkers per CPU. A subfolder gets a new goroutine only when a slot is free, otherwise the current walker descends into it itself.
- **No second look at the disk.** Project detection (lockfiles next to `node_modules`, `pyvenv.cfg`, `pyproject.toml`) checks the listing already in memory.
- **Pooled buffers and in-place name decoding.** These keep allocation close to the baseline's.

It doesn't read the NTFS master file table directly, as WizTree does. That would be faster again, but it needs administrator rights, and Cache Vent runs as a normal user.

## History

| Version | `D:` | All drives | Peak memory (`D:`) |
|---|---:|---:|---:|
| First version, `filepath.WalkDir` + per-file `Lstat` | 105 s | – | – |
| Batched listing, a goroutine per folder | 2.6 s | 13.3 s | 452 MB |
| Bounded walkers, pooled buffers (current) | 2.5 s | 2.9 s | 41 MB |

## Reproducing the real-drive numbers

The baseline is about 20 lines of Go:

```go
filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
	if err == nil && !d.IsDir() {
		if fi, err := d.Info(); err == nil {
			bytes += fi.Size()
		}
	}
	return nil
})
```

Time it and Cache Vent with PowerShell's `Measure-Command`, or with [hyperfine](https://github.com/sharkdp/hyperfine): `hyperfine --warmup 1 "cachevent scan --json --root D:\\"`. Scanning only reads. It writes nothing except Cache Vent's own last-scan file and history.
