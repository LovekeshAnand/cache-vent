package envs

import (
	"os"
	"path/filepath"
	"testing"

	"cachevent/internal/defs"
)

func pipenv(t *testing.T, project string) string {
	t.Helper()
	env := t.TempDir()
	if project != "" {
		if err := os.WriteFile(filepath.Join(env, ".project"), []byte(project+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return env
}

func TestPipenvOwnership(t *testing.T) {
	base := t.TempDir()
	live := filepath.Join(base, "live")
	os.Mkdir(live, 0o755)

	if v := Inspect(pipenv(t, live), "pipenv"); v.Info.Orphaned || v.Safety != defs.Redownloadable || v.Info.Project != live {
		t.Errorf("live project: %+v", v)
	}
	if v := Inspect(pipenv(t, filepath.Join(base, "gone")), "pipenv"); !v.Info.Orphaned || v.Safety != defs.Redownloadable {
		t.Errorf("deleted project should be orphaned: %+v", v)
	}
	// Parent missing too (e.g. a drive that isn't plugged in): not provably orphaned.
	if v := Inspect(pipenv(t, filepath.Join(base, "nodrive", "proj")), "pipenv"); v.Info.Orphaned || v.Safety != defs.Unknown {
		t.Errorf("missing parent should be unknown: %+v", v)
	}
	if v := Inspect(pipenv(t, ""), "pipenv"); v.Safety != defs.Unknown {
		t.Errorf("no .project should be unknown: %+v", v)
	}
}

func TestCondaIsNeverCleanable(t *testing.T) {
	if v := Inspect(t.TempDir(), "conda"); defs.Cleanable(v.Safety) {
		t.Errorf("conda env marked %s", v.Safety)
	}
}
