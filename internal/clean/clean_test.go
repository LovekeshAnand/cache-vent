package clean

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cachevent/internal/defs"
	"cachevent/internal/scan"
)

func TestPlan(t *testing.T) {
	dir := t.TempDir()
	nm := filepath.Join(dir, "node_modules")
	sessions := filepath.Join(dir, "sessions")
	temp := filepath.Join(dir, "temp")
	for _, d := range []string{nm, sessions, filepath.Join(temp, "old"), filepath.Join(temp, "new")} {
		os.MkdirAll(d, 0o755)
	}
	os.WriteFile(filepath.Join(temp, "old", "f"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(temp, "new", "f"), []byte("x"), 0o644)
	now := time.Now()
	long := now.Add(-30 * 24 * time.Hour)
	os.Chtimes(filepath.Join(temp, "old", "f"), long, long)
	os.Chtimes(filepath.Join(temp, "old"), long, long)

	last := scan.Result{Items: []scan.Item{
		{ID: "node-modules", Path: nm, Safety: defs.Redownloadable, Cleanable: true, Bytes: 10},
		{ID: "sessions", Path: sessions, Safety: defs.UserData},
		{ID: "temp", Path: temp, Safety: defs.Regenerable, Cleanable: true},
	}}
	entries := []defs.Entry{
		{ID: "node-modules", Safety: defs.Redownloadable},
		{ID: "sessions", Safety: defs.UserData},
		{ID: "temp", Safety: defs.Regenerable, CleanOlderThanDays: 7},
	}

	if ts, err := Plan(last, entries, []string{nm}, now); err != nil || len(ts) != 1 || ts[0].Path != nm {
		t.Errorf("node_modules: %+v, %v", ts, err)
	}
	if _, err := Plan(last, entries, []string{sessions}, now); err == nil {
		t.Error("user-data was accepted")
	}
	if _, err := Plan(last, entries, []string{filepath.Join(dir, "elsewhere")}, now); err == nil {
		t.Error("path not in the scan was accepted")
	}
	ts, err := Plan(last, entries, []string{temp}, now)
	if err != nil || len(ts) != 1 || ts[0].Path != filepath.Join(temp, "old") {
		t.Errorf("temp: want only the old child, got %+v, %v", ts, err)
	}

	// A newer definition downgraded to unknown wins over a stale scan.
	entries[0].Safety = defs.Unknown
	if _, err := Plan(last, entries, []string{nm}, now); err == nil {
		t.Error("downgraded entry was accepted")
	}
}
