package main

import (
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
