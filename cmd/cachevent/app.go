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
func (s *session) scan(roots []string, progress func(int)) (scan.Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return scan.Result{}, err
	}
	if len(roots) == 0 {
		roots = []string{home}
	}
	// App data is covered by fixed-path definitions; walking it for patterns is slow and noisy.
	skip := append([]string{filepath.Join(home, "AppData"), filepath.Join(home, "Library"), s.dataDir}, s.store.Roots()...)
	if sys := os.Getenv("SystemRoot"); sys != "" {
		skip = append(skip, sys)
	}
	res := scan.Run(scan.Options{Defs: s.entries, Envs: envs.Find(), Roots: roots, Skip: skip, Progress: progress})
	return res, s.save(res)
}

// save keeps the last scan for clean, and appends per-definition totals to history.
// ponytail: history mixes scans with different roots; store roots per line if that skews trends.
func (s *session) save(res scan.Result) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	// Temp file + rename, so a crash never leaves a half-written scan.
	tmp := filepath.Join(s.dataDir, "last-scan.json.tmp")
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, filepath.Join(s.dataDir, "last-scan.json")); err != nil {
		return err
	}
	sums := map[string]int64{}
	for _, it := range res.Items {
		sums[it.ID] += it.Bytes
	}
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
	for _, id := range ids {
		if err := enc.Encode(struct {
			Time  time.Time `json:"time"`
			ID    string    `json:"id"`
			Bytes int64     `json:"bytes"`
		}{res.ScannedAt, id, sums[id]}); err != nil {
			return err
		}
	}
	return nil
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
