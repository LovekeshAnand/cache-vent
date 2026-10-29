package main

// The scan and clean flow, shared by the CLI and the desktop UI so both go
// through the same safety checks.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cachevent/definitions"
	"cachevent/internal/clean"
	"cachevent/internal/defs"
	"cachevent/internal/envs"
	"cachevent/internal/quarantine"
	"cachevent/internal/scan"
)

type session struct {
	dataDir string
	store   quarantine.Store
	entries []defs.Entry
}

// newSession loads definitions and purges expired quarantine items. There is
// no background service, so this is when the 14-day purge happens.
func newSession() (*session, error) {
	entries, err := defs.Load(definitions.FS)
	if err != nil {
		return nil, err
	}
	dd, err := defs.DataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dd, 0o755); err != nil {
		return nil, err
	}
	st := quarantine.Store{DataDir: dd}
	n, b, err := st.Purge(quarantine.MaxAge)
	if n > 0 {
		fmt.Fprintf(os.Stderr, "Purged %d quarantined items older than 14 days (%s).\n", n, human(b))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "purge:", err)
	}
	return &session{dd, st, entries}, nil
}

// scan searches roots (default: home) and saves the result for clean.
func (s *session) scan(roots []string, progress func(items int, dirs int64)) (scan.Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return scan.Result{}, err
	}
	if len(roots) == 0 {
		roots = []string{home}
	}
	// App data is covered by fixed-path definitions, and installed software is never
	// "your project": walking them for patterns is slow and risks false matches.
	skip := append([]string{filepath.Join(home, "Library"), s.dataDir}, s.store.Roots()...)
	for _, v := range []string{"SystemRoot", "ProgramFiles", "ProgramFiles(x86)", "ProgramData"} {
		skip = append(skip, os.Getenv(v))
	}
	res := scan.Run(scan.Options{Defs: s.entries, Envs: envs.Find(), Roots: roots, Skip: skip, Progress: progress})
	if err := s.save(res); err != nil {
		return res, err
	}
	return res, s.record(res)
}

// save keeps the last scan for clean. Temp file + rename, so a crash never
// leaves a half-written scan.
func (s *session) save(res scan.Result) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.dataDir, "last-scan.json.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.dataDir, "last-scan.json"))
}

// historyLine is one definition's total in one scan. Scope names the roots, so a
// home-only scan is never compared with an all-drives one.
type historyLine struct {
	Time  time.Time `json:"time"`
	ID    string    `json:"id"`
	Bytes int64     `json:"bytes"`
	Scope string    `json:"scope,omitempty"`
}

func scope(roots []string) string {
	keys := make([]string, len(roots))
	for i, r := range roots {
		keys[i] = defs.PathKey(r)
	}
	sort.Strings(keys)
	return strings.Join(keys, ";")
}

func totals(res scan.Result) map[string]int64 {
	sums := map[string]int64{}
	for _, it := range res.Items {
		sums[it.ID] += it.Bytes
	}
	return sums
}

// record appends the scan's per-definition totals to history.jsonl.
func (s *session) record(res scan.Result) error {
	sums := totals(res)
	ids := make([]string, 0, len(sums))
	for id := range sums {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	f, err := os.OpenFile(filepath.Join(s.dataDir, "history.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	sc := scope(res.Roots)
	for _, id := range ids {
		if err := enc.Encode(historyLine{res.ScannedAt, id, sums[id], sc}); err != nil {
			return err
		}
	}
	return nil
}

type growth struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Delta int64     `json:"delta"`
	Since time.Time `json:"since"`
}

// growth compares res with an earlier scan of the same roots: the latest one at
// least 7 days old, or else the oldest one at least a day old. Biggest change first.
// ponytail: reads the whole history file; fine for years of daily scans (~30 lines each).
func (s *session) growth(res scan.Result) []growth {
	f, err := os.Open(filepath.Join(s.dataDir, "history.jsonl"))
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := scope(res.Roots)
	byTime := map[time.Time]map[string]int64{}
	dec := json.NewDecoder(f)
	for {
		var l historyLine
		if dec.Decode(&l) != nil {
			break
		}
		if l.Scope != sc || !l.Time.Before(res.ScannedAt.Add(-24*time.Hour)) {
			continue
		}
		if byTime[l.Time] == nil {
			byTime[l.Time] = map[string]int64{}
		}
		byTime[l.Time][l.ID] = l.Bytes
	}
	var base time.Time
	week := res.ScannedAt.Add(-7 * 24 * time.Hour)
	for t := range byTime {
		switch {
		case !t.After(week) && (base.After(week) || t.After(base)):
			base = t // latest scan at least a week old
		case base.IsZero() || (base.After(week) && t.Before(base)):
			base = t // otherwise the oldest one
		}
	}
	if base.IsZero() {
		return nil
	}
	names := map[string]string{}
	for _, it := range res.Items {
		names[it.ID] = it.Name
	}
	now, then := totals(res), byTime[base]
	var out []growth
	for id, n := range now {
		if d := n - then[id]; d >= 50<<20 || d <= -(50<<20) {
			out = append(out, growth{id, names[id], d, base})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Delta > out[j].Delta })
	return out
}

// lastScan returns the saved scan; ok is false if there is none yet.
func (s *session) lastScan() (res scan.Result, ok bool, err error) {
	b, err := os.ReadFile(filepath.Join(s.dataDir, "last-scan.json"))
	if errors.Is(err, os.ErrNotExist) {
		return res, false, nil
	}
	if err != nil {
		return res, false, err
	}
	if err := json.Unmarshal(b, &res); err != nil {
		return res, false, fmt.Errorf("last scan is unreadable, scan again: %w", err)
	}
	return res, true, nil
}

func (s *session) plan(paths []string) ([]clean.Target, error) {
	last, ok, err := s.lastScan()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("no scan yet; scan first")
	}
	return clean.Plan(last, s.entries, paths, time.Now())
}

type cleanResult struct {
	Moved  []quarantine.Manifest `json:"moved"`
	Failed []string              `json:"failed"`
	Bytes  int64                 `json:"bytes"`
}

// delete removes targets for good. It goes through quarantine (move, then purge
// exactly those), so the one place files are deleted stays quarantine's purge.
func (s *session) delete(targets []clean.Target) (cleanResult, error) {
	r := s.clean(targets)
	ids := make([]string, len(r.Moved))
	for i, m := range r.Moved {
		ids[i] = m.ID
	}
	_, _, err := s.store.PurgeIDs(ids)
	return r, err
}

func (s *session) clean(targets []clean.Target) cleanResult {
	r := cleanResult{Moved: []quarantine.Manifest{}, Failed: []string{}}
	for _, t := range targets {
		m, err := s.store.Move(t.Path, t.DefinitionID, t.Bytes)
		if err != nil {
			r.Failed = append(r.Failed, err.Error())
			continue
		}
		r.Moved = append(r.Moved, m)
		r.Bytes += t.Bytes
	}
	return r
}
