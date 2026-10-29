package defs_test

import (
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"

	"cachevent/definitions"
	"cachevent/internal/defs"
)

func TestShippedDefinitionsAreValid(t *testing.T) {
	entries, err := defs.Load(definitions.FS)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 15 {
		t.Fatalf("got %d entries", len(entries))
	}
}

const good = `tool: X
entries:
  - id: x-cache
    name: X cache
    paths: {windows: ['C:\x'], linux: ["/x"]}
    kind: cache
    safety: regenerable
    if_deleted: Rebuilt.
    verified: {by: me, on: "2026-10-29", source: https://example.com}
`

func TestLoadRejectsBadEntries(t *testing.T) {
	if _, err := defs.Load(fstest.MapFS{"x.yaml": {Data: []byte(good)}}); err != nil {
		t.Fatalf("good entry rejected: %v", err)
	}
	bad := map[string]string{
		"no source":       "tool: X\nentries:\n  - {id: x, name: X, paths: {linux: [/x]}, kind: cache, safety: regenerable, if_deleted: y, verified: {by: me, on: d}}\n",
		"bad id":          "tool: X\nentries:\n  - {id: X_Y, name: X, paths: {linux: [/x]}, kind: cache, safety: unknown, if_deleted: y}\n",
		"unknown field":   "tool: X\nentries:\n  - {id: x, name: X, paths: {linux: [/x]}, kind: cache, safety: unknown, if_deleted: y, oops: 1}\n",
		"dedupe cache":    "tool: X\nentries:\n  - {id: x, name: X, paths: {linux: [/x]}, kind: cache, safety: unknown, if_deleted: y, dedupe: true}\n",
		"paths+pattern":   "tool: X\nentries:\n  - {id: x, name: X, paths: {linux: [/x]}, pattern: {dir_name: a}, kind: cache, safety: unknown, if_deleted: y}\n",
		"no if_deleted":   "tool: X\nentries:\n  - {id: x, name: X, paths: {linux: [/x]}, kind: cache, safety: unknown}\n",
		"bad os":          "tool: X\nentries:\n  - {id: x, name: X, paths: {win: [/x]}, kind: cache, safety: unknown, if_deleted: y}\n",
		"user-data no by": "tool: X\nentries:\n  - {id: x, name: X, paths: {linux: [/x]}, kind: cache, safety: user-data, if_deleted: y}\n",
	}
	for name, src := range bad {
		if _, err := defs.Load(fstest.MapFS{"x.yaml": {Data: []byte(src)}}); err == nil {
			t.Errorf("%s: loaded, want error", name)
		}
	}
	if _, err := defs.Load(fstest.MapFS{"a.yaml": {Data: []byte(good)}, "b.yaml": {Data: []byte(good)}}); err == nil {
		t.Error("duplicate id loaded, want error")
	}
}

func TestExpand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CV_TEST_DIR", dir)
	p := "$CV_TEST_DIR/models"
	if runtime.GOOS == "windows" {
		p = `%CV_TEST_DIR%\models`
	}
	if got, want := defs.Expand(p), filepath.Join(dir, "models"); got != want {
		t.Errorf("Expand(%q) = %q, want %q", p, got, want)
	}
	if got := defs.Expand("%CV_TEST_UNSET%/x"); got != "" {
		t.Errorf("unset variable expanded to %q, want empty", got)
	}
	if got := defs.Expand("relative/path"); got != "" {
		t.Errorf("relative path expanded to %q, want empty", got)
	}
}
