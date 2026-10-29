// Package scan finds what the definitions and the environment analyzer describe
// on disk and measures it. It never writes anything.
package scan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	ScannedAt time.Time `json:"scanned_at"`
	Roots     []string  `json:"roots"`
	Items     []Item    `json:"items"`
	Warnings  []string  `json:"warnings,omitempty"`
}

type Options struct {
	Defs  []defs.Entry
	Envs  []envs.Found // central environments (poetry, pipenv, conda)
	Roots []string     // searched for pattern entries and in-project environments
	Skip  []string     // never descended during the root search
}

func Run(opt Options) Result {
	s := newSizer()
	res := Result{ScannedAt: time.Now().UTC(), Roots: opt.Roots, Items: []Item{}}
	add := func(it Item) {
		it.Bytes, it.LastUsed = s.size(it.Path)
		it.Cleanable = defs.Cleanable(it.Safety) && it.Bytes > 0
		res.Items = append(res.Items, it)
	}
	claimed := map[string]bool{} // measured already; the root search skips these
	claim := func(p string) bool {
		fi, err := os.Lstat(p)
		if err != nil || claimed[defs.PathKey(p)] {
			return false
		}
		if fi.Mode()&linkMode != 0 {
			res.Warnings = append(res.Warnings, p+": link or junction, not followed")
			return false
		}
		claimed[defs.PathKey(p)] = true
		return true
	}

	var patterns []defs.Entry
	for _, e := range opt.Defs {
		if e.Pattern != nil {
			patterns = append(patterns, e)
			continue
		}
		for _, p := range e.ResolvedPaths() {
			if claim(p) {
				add(fromEntry(e, p))
			}
		}
	}
	for _, f := range opt.Envs {
		if claim(f.Path) {
			add(fromEnv(f.Path, envs.Inspect(f.Path, f.Manager)))
		}
	}

	skip := map[string]bool{}
	for _, p := range opt.Skip {
		skip[defs.PathKey(p)] = true
	}
	for _, root := range opt.Roots {
		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if path == root {
					res.Warnings = append(res.Warnings, err.Error())
				}
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			k := defs.PathKey(path)
			if claimed[k] || skip[k] {
				return filepath.SkipDir
			}
			if envs.IsVenv(path) {
				claimed[k] = true
				add(fromEnv(path, envs.Inspect(path, "venv")))
				return filepath.SkipDir
			}
			for _, e := range patterns {
				if matches(path, d.Name(), e.Pattern) {
					claimed[k] = true
					add(fromEntry(e, path))
					return filepath.SkipDir
				}
			}
			return nil
		})
	}

	countSharedPackages(res.Items)
	sort.SliceStable(res.Items, func(i, j int) bool { return res.Items[i].Bytes > res.Items[j].Bytes })
	if s.skipped > 0 {
		res.Warnings = append(res.Warnings, fmt.Sprintf("%d files or folders could not be read (permission denied or in use)", s.skipped))
	}
	return res
}

func matches(path, name string, p *defs.Pattern) bool {
	if defs.PathKey(name) != defs.PathKey(p.DirName) {
		return false
	}
	if len(p.Siblings) == 0 {
		return true
	}
	for _, s := range p.Siblings {
		if _, err := os.Lstat(filepath.Join(filepath.Dir(path), s)); err == nil {
			return true
		}
	}
	return false
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
