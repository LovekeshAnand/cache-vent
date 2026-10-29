package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cachevent/internal/scan"
)

func TestGrowth(t *testing.T) {
	s := &session{dataDir: t.TempDir()}
	now := time.Now().UTC()
	at := func(daysAgo int, roots []string, bytes int64) scan.Result {
		return scan.Result{ScannedAt: now.Add(-time.Duration(daysAgo) * 24 * time.Hour), Roots: roots,
			Items: []scan.Item{{ID: "cursor-cache", Name: "Cursor", Bytes: bytes}}}
	}
	home, all := []string{`C:\Users\me`}, []string{`C:\`, `D:\`}
	for _, r := range []scan.Result{at(30, home, 1<<30), at(8, home, 2<<30), at(3, home, 3<<30), at(8, all, 9<<30)} {
		if err := s.record(r); err != nil {
			t.Fatal(err)
		}
	}
	g := s.growth(at(0, home, 5<<30))
	// Baseline is the latest scan at least a week old with the same roots (8 days, 2 GB).
	if len(g) != 1 || g[0].Delta != 3<<30 {
		t.Fatalf("growth = %+v, want +3 GB against the 8-day-old home scan", g)
	}
	if g := s.growth(at(0, []string{`E:\`}, 5<<30)); len(g) != 0 {
		t.Errorf("different roots compared: %+v", g)
	}
}

func TestTrimProfileKeepsOnlySettings(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"Local State", "component_crx_cache/a.crx", "WidevineCdm/x.dll",
		"Default/Preferences", "Default/Local Storage/leveldb/000.log", "Default/Extensions/abc/ext.js", "Default/Cache/data_0"} {
		full := filepath.Join(dir, p)
		os.MkdirAll(filepath.Dir(full), 0o755)
		os.WriteFile(full, []byte("x"), 0o644)
	}
	trimProfile(dir)
	for p, want := range map[string]bool{"Local State": true, "Default/Preferences": true, "Default/Local Storage/leveldb/000.log": true,
		"component_crx_cache": false, "WidevineCdm": false, "Default/Extensions": false, "Default/Cache": false} {
		if _, err := os.Stat(filepath.Join(dir, p)); (err == nil) != want {
			t.Errorf("%s: exists=%v, want %v", p, err == nil, want)
		}
	}
}
