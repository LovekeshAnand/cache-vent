// Package clean decides what may be moved to quarantine. It only accepts paths
// the last scan found and marked cleanable.
package clean

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cachevent/internal/defs"
	"cachevent/internal/envs"
	"cachevent/internal/scan"
)

type Target struct {
	Path         string
	DefinitionID string
	Bytes        int64
}

// Plan resolves paths against the last scan. Entries with clean_older_than_days
// yield only their children untouched that long.
func Plan(last scan.Result, entries []defs.Entry, paths []string, now time.Time) ([]Target, error) {
	byID := map[string]defs.Entry{}
	for _, e := range entries {
		byID[e.ID] = e
	}
	items := map[string]scan.Item{}
	for _, it := range last.Items {
		items[defs.PathKey(it.Path)] = it
	}
	var out []Target
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}
		it, ok := items[defs.PathKey(abs)]
		if !ok {
			return nil, fmt.Errorf("%s: not in the last scan; run `cachevent scan` and use a path it lists", p)
		}
		e, isDef := byID[it.ID]
		// Re-check against current definitions: a newer one may have downgraded it.
		if !it.Cleanable || (isDef && !defs.Cleanable(e.Safety)) {
			return nil, fmt.Errorf("%s: %s, never cleaned", p, it.Safety)
		}
		// The scan may be old: re-check what an environment looks like now.
		if it.Env != nil {
			if v := envs.Inspect(abs, it.Env.Manager); !defs.Cleanable(v.Safety) {
				return nil, fmt.Errorf("%s: now %s (%s)", p, v.Safety, v.IfDeleted)
			}
		}
		if protected(abs) {
			return nil, fmt.Errorf("%s: refusing to move a home folder, drive root, or anything holding Cache Vent's data", p)
		}
		if sync := syncedFolder(abs); sync != "" {
			return nil, fmt.Errorf("%s: inside the synced folder %s; moving it would delete it on every synced device", p, sync)
		}
		fi, err := os.Lstat(abs)
		if err != nil {
			return nil, err
		}
		if fi.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			return nil, fmt.Errorf("%s: link or junction, not followed", p)
		}
		if e.CleanOlderThanDays > 0 {
			ts, err := oldChildren(abs, it.ID, now.Add(-time.Duration(e.CleanOlderThanDays)*24*time.Hour))
			if err != nil {
				return nil, err
			}
			out = append(out, ts...)
			continue
		}
		out = append(out, Target{abs, it.ID, it.Bytes})
	}
	return out, nil
}

func oldChildren(dir, id string, cutoff time.Time) ([]Target, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Target
	for _, e := range ents {
		if e.Type()&(os.ModeSymlink|os.ModeIrregular) != 0 {
			continue
		}
		p := filepath.Join(dir, e.Name())
		n, newest := scan.Size(p)
		if !newest.IsZero() && newest.Before(cutoff) {
			out = append(out, Target{p, id, n})
		}
	}
	return out, nil
}

// protected is true for drive roots, and for the home folder, the data dir and anything containing them.
func protected(p string) bool {
	if filepath.Dir(p) == p {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return true
	}
	dd, err := defs.DataDir()
	if err != nil {
		return true
	}
	return defs.Within(home, p) || defs.Within(dd, p)
}

// syncedFolder returns the cloud-sync folder holding p, or "". Matching by folder
// name can refuse a folder that only looks synced; refusing is the safe mistake.
func syncedFolder(p string) string {
	var roots []string
	for _, v := range []string{"OneDrive", "OneDriveConsumer", "OneDriveCommercial"} {
		if d := os.Getenv(v); d != "" {
			roots = append(roots, d)
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		roots = append(roots, filepath.Join(home, "Library", "CloudStorage"), filepath.Join(home, "Library", "Mobile Documents"))
	}
	for _, r := range roots {
		if defs.Within(p, r) {
			return r
		}
	}
	for dir := p; filepath.Dir(dir) != dir; dir = filepath.Dir(dir) {
		switch name := strings.ToLower(filepath.Base(dir)); {
		case name == "dropbox", name == "google drive", name == "my drive", name == "icloud drive", name == "iclouddrive",
			strings.HasPrefix(name, "onedrive"):
			return dir
		}
	}
	return ""
}
