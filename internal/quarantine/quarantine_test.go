package quarantine

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setup(t *testing.T) (Store, string, []byte) {
	t.Helper()
	st := Store{DataDir: t.TempDir()}
	item := filepath.Join(t.TempDir(), "node_modules")
	content := []byte("module.exports = 42\n")
	if err := os.MkdirAll(filepath.Join(item, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(item, "pkg", "index.js"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	return st, item, content
}

func TestMoveRestoreRoundTrip(t *testing.T) {
	st, item, content := setup(t)
	m, err := st.Move(item, "node-modules", int64(len(content)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(item); !os.IsNotExist(err) {
		t.Fatal("original still there after move")
	}
	if got := st.List(); len(got) != 1 || got[0].OriginalPath != item {
		t.Fatalf("list: %+v", got)
	}

	if _, err := st.Restore(m.ID); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(item, "pkg", "index.js"))
	if err != nil || !bytes.Equal(got, content) {
		t.Fatalf("restored content %q, %v", got, err)
	}
	if n := len(st.List()); n != 0 {
		t.Errorf("%d items left in quarantine", n)
	}
}

func TestRestoreNeverOverwrites(t *testing.T) {
	st, item, _ := setup(t)
	m, err := st.Move(item, "node-modules", 0)
	if err != nil {
		t.Fatal(err)
	}
	os.Mkdir(item, 0o755)
	if _, err := st.Restore(m.ID); err == nil {
		t.Fatal("restore overwrote an existing path")
	}
	if len(st.List()) != 1 {
		t.Error("item should still be in quarantine")
	}
}

func TestPurgeOnlyExpired(t *testing.T) {
	st, item, _ := setup(t)
	old, err := st.Move(item, "node-modules", 0)
	if err != nil {
		t.Fatal(err)
	}
	old.QuarantinedAt = time.Now().Add(-15 * 24 * time.Hour)
	b, _ := json.Marshal(old)
	os.WriteFile(filepath.Join(old.Dir, manifestName), b, 0o644)

	_, fresh, _ := setup(t)
	if _, err := st.Move(fresh, "node-modules", 0); err != nil {
		t.Fatal(err)
	}

	if n, _, err := st.Purge(MaxAge); n != 1 || err != nil {
		t.Fatalf("purged %d, %v; want 1", n, err)
	}
	if left := st.List(); len(left) != 1 || left[0].OriginalPath != fresh {
		t.Errorf("left: %+v", left)
	}
}
