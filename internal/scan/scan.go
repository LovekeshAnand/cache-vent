// Package scan finds what the definitions and the environment analyzer describe
// on disk and measures it. It never writes anything.
package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"cachevent/internal/defs"
	"cachevent/internal/envs"
)

// Item is one thing found on disk. Its JSON form is a stable contract: add
// fields, never rename or remove them.
type Item struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Kind      string     `json:"kind"`
	Bytes     int64      `json:"bytes"`
	Safety    string     `json:"safety"`
	Cleanable bool       `json:"cleanable"`
	IfDeleted string     `json:"if_deleted"`
	Source    string     `json:"source,omitempty"`
	LastUsed  time.Time  `json:"last_used,omitzero"`
	Env       *envs.Info `json:"env,omitempty"`
}

type Result struct {
	ScannedAt   time.Time `json:"scanned_at"`
	Roots       []string  `json:"roots"`
	Items       []Item    `json:"items"`
	Warnings    []string  `json:"warnings,omitempty"`
	DirsScanned int64     `json:"dirs_scanned"`
	DurationMS  int64     `json:"duration_ms"`
}

type Options struct {
	Defs  []defs.Entry
	Envs  []envs.Found // central environments (poetry, pipenv, conda)
	Roots []string     // searched for pattern entries and in-project environments
	Skip  []string     // never descended during the root search
	// Progress, if set, is called as items are found and folders are read.
	Progress func(items int, dirs int64)
}

// neverWalk are folder names the root search never enters: our own quarantine
// (its contents would show up as cleanable again), and OS or app-managed folders
// that fixed-path definitions cover. Names starting with "$" are system folders
// ($Recycle.Bin, $WINDOWS.~BT, ...) and are skipped too.
var neverWalk = map[string]bool{
	".cachevent-quarantine": true, "system volume information": true, "windows.old": true, "appdata": true,
}

type run struct {
	opt      Options
	s        *sizer
	mu       sync.Mutex
	res      Result
	claimed  map[string]bool // measured already; the root search skips these
	skip     map[string]bool
	projects map[string]string // Poetry path hash -> project folder (any folder with pyproject.toml)
}

func Run(opt Options) Result {
	start := time.Now()
	r := &run{opt: opt, s: newSizer(), claimed: map[string]bool{}, skip: map[string]bool{}, projects: map[string]string{},
		res: Result{ScannedAt: start.UTC(), Roots: opt.Roots, Items: []Item{}}}
	for _, p := range opt.Skip {
		if p != "" {
			r.skip[defs.PathKey(p)] = true
		}
	}
	r.s.onDir = func(dirs int64) { r.progress() }

	var patterns []defs.Entry
	for _, e := range opt.Defs {
		if e.Pattern != nil {
			patterns = append(patterns, e)
			continue
		}
		for _, p := range e.ResolvedPaths() {
			if r.claim(p) {
				r.add(fromEntry(e, p))
			}
		}
	}
	for _, f := range opt.Envs {
		if r.claim(f.Path) {
			r.add(fromEnv(f.Path, envs.Inspect(f.Path, f.Manager)))
		}
	}

	for _, root := range opt.Roots {
		fi, err := os.Lstat(root)
		switch {
		case err != nil:
			r.warn(err.Error())
		case fi.Mode()&linkMode != 0:
			r.warn(root + ": link or junction, not followed")
		case fi.IsDir():
			var wg sync.WaitGroup
			r.walk(root, r.s.list(root), patterns, &wg)
			wg.Wait()
		}
	}

	r.resolvePoetryOwners()
	countSharedPackages(r.res.Items)
	sort.SliceStable(r.res.Items, func(i, j int) bool { return r.res.Items[i].Bytes > r.res.Items[j].Bytes })
	if n := r.s.skipped.Load(); n > 0 {
		r.warn(fmt.Sprintf("%d folders could not be read (permission denied or in use)", n))
	}
	r.res.DirsScanned = r.s.dirs.Load()
	r.res.DurationMS = time.Since(start).Milliseconds()
	return r.res
}

// walk looks at one folder's listing: subfolders matching a pattern become items,
// virtual environments become items, everything else is searched in parallel.
func (r *run) walk(dir string, ents []entry, patterns []defs.Entry, wg *sync.WaitGroup) {
	names := make(map[string]bool, len(ents))
	for _, e := range ents {
		names[strings.ToLower(e.name)] = true
	}
	if names["pyproject.toml"] {
		r.mu.Lock()
		r.projects[envs.PoetryHash(dir)] = dir
		r.mu.Unlock()
	}
	for _, e := range ents {
		if !e.isDir || e.link {
			continue
		}
		lower := strings.ToLower(e.name)
		p := filepath.Join(dir, e.name)
		if neverWalk[lower] || strings.HasPrefix(lower, "$") || r.isSkipped(p) {
			continue
		}
		if m := match(lower, names, patterns); m != nil {
			if r.claim(p) {
				r.add(fromEntry(*m, p))
			}
			continue
		}
		r.s.spawn(wg, func() {
			sub := r.s.list(p)
			for _, c := range sub {
				if !c.isDir && strings.EqualFold(c.name, "pyvenv.cfg") {
					if r.claim(p) {
						r.add(fromEnv(p, envs.Inspect(p, "venv")))
					}
					return
				}
			}
			r.walk(p, sub, patterns, wg)
		})
	}
}

// match returns the pattern entry for a folder named lower whose parent holds names.
func match(lower string, names map[string]bool, patterns []defs.Entry) *defs.Entry {
	for i, e := range patterns {
		if lower != strings.ToLower(e.Pattern.DirName) {
			continue
		}
		if len(e.Pattern.Siblings) == 0 {
			return &patterns[i]
		}
		for _, s := range e.Pattern.Siblings {
			if names[strings.ToLower(s)] {
				return &patterns[i]
			}
		}
	}
	return nil
}

func (r *run) isSkipped(p string) bool {
	k := defs.PathKey(p)
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.claimed[k] || r.skip[k]
}

// claim marks p as an item; false if it's missing, a link, or already claimed.
func (r *run) claim(p string) bool {
	fi, err := os.Lstat(p)
	if err != nil {
		return false
	}
	if fi.Mode()&linkMode != 0 {
		r.warn(p + ": link or junction, not followed")
		return false
	}
	k := defs.PathKey(p)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.claimed[k] {
		return false
	}
	r.claimed[k] = true
	return true
}

func (r *run) add(it Item) {
	it.Bytes, it.LastUsed = r.s.size(it.Path)
	it.Cleanable = defs.Cleanable(it.Safety) && it.Bytes > 0
	r.mu.Lock()
	r.res.Items = append(r.res.Items, it)
	r.mu.Unlock()
	r.progress()
}

func (r *run) progress() {
	if r.opt.Progress == nil {
		return
	}
	r.mu.Lock()
	n := len(r.res.Items)
	r.mu.Unlock()
	r.opt.Progress(n, r.s.dirs.Load())
}

func (r *run) warn(msg string) {
	r.mu.Lock()
	r.res.Warnings = append(r.res.Warnings, msg)
	r.mu.Unlock()
}

// resolvePoetryOwners links Poetry envs to the projects found under the roots.
func (r *run) resolvePoetryOwners() {
	for _, it := range r.res.Items {
		if it.Env == nil || it.Env.Manager != "poetry" {
			continue
		}
		if h, ok := envs.PoetryEnvHash(it.Path); ok {
			it.Env.Project = r.projects[h]
		}
	}
}

// countSharedPackages sets OtherEnvs: how many other environments hold the same package version.
func countSharedPackages(items []Item) {
	counts := map[string]int{}
	for _, it := range items {
		if it.Env != nil {
			for _, p := range it.Env.Packages {
				counts[p.Name+"=="+p.Version]++
			}
		}
	}
	for _, it := range items {
		if it.Env != nil {
			for j := range it.Env.Packages {
				p := &it.Env.Packages[j]
				p.OtherEnvs = counts[p.Name+"=="+p.Version] - 1
			}
		}
	}
}

func fromEntry(e defs.Entry, path string) Item {
	return Item{ID: e.ID, Name: e.Name, Path: path, Kind: e.Kind, Safety: e.Safety, IfDeleted: e.IfDeleted, Source: e.Verified.Source}
}

func fromEnv(path string, v envs.Verdict) Item {
	info := v.Info
	return Item{ID: v.ID, Name: v.Name, Path: path, Kind: "dependencies", Safety: v.Safety, IfDeleted: v.IfDeleted, Source: v.Source, Env: &info}
}
