package scan

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"cachevent/internal/defs"
)

func write(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeVenv(t *testing.T, env string) {
	t.Helper()
	write(t, filepath.Join(env, "pyvenv.cfg"), 10)
	sp := filepath.Join(env, "lib", "python3.12", "site-packages")
	if runtime.GOOS == "windows" {
		sp = filepath.Join(env, "Lib", "site-packages")
	}
	record := filepath.Join(sp, "Requests-2.32.0.dist-info", "RECORD")
	write(t, record, 0)
	os.WriteFile(record, []byte("requests/__init__.py,sha256=x,5000\nrequests/api.py,sha256=y,3000\nRECORD,,\n"), 0o644)
}

func TestRun(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "app", "package-lock.json"), 10)
	write(t, filepath.Join(root, "app", "node_modules", "lib", "index.js"), 1000)
	write(t, filepath.Join(root, "app", "node_modules", "lib", "node_modules", "x.js"), 1000)
	// package.json without a lockfile (global install, bundled vendor folder): not matched.
	write(t, filepath.Join(root, "vendor", "package.json"), 10)
	write(t, filepath.Join(root, "vendor", "node_modules", "a.js"), 1000)
	write(t, filepath.Join(root, "proj", "requirements.txt"), 10)
	writeVenv(t, filepath.Join(root, "proj", ".venv"))
	writeVenv(t, filepath.Join(root, "bare", "venv"))
	// Quarantined items must not show up as cleanable again.
	write(t, filepath.Join(root, ".cachevent-quarantine", "x", "payload", "package-lock.json"), 10)
	write(t, filepath.Join(root, ".cachevent-quarantine", "x", "payload", "node_modules", "a.js"), 10)

	fixed := filepath.Join(t.TempDir(), "models")
	big := filepath.Join(fixed, "blob")
	write(t, big, 2<<20)
	if err := os.Link(big, filepath.Join(fixed, "blob-link")); err != nil {
		t.Fatal(err)
	}

	entries := []defs.Entry{
		{ID: "test-models", Name: "Models", Kind: "model", Safety: defs.Redownloadable,
			Paths: map[string][]string{defs.OSKey(): {fixed}}},
		{ID: "node-modules", Name: "node_modules", Kind: "dependencies", Safety: defs.Redownloadable,
			Pattern: &defs.Pattern{DirName: "node_modules", Siblings: []string{"package-lock.json", "yarn.lock"}}},
	}
	res := Run(Options{Defs: entries, Roots: []string{root}})

	byPath := map[string]Item{}
	for _, it := range res.Items {
		byPath[it.Path] = it
	}
	if len(res.Items) != 4 {
		t.Fatalf("got %d items, want 4: %+v", len(res.Items), res.Items)
	}

	m := byPath[fixed]
	if m.Bytes < 2<<20 || m.Bytes >= 3<<20 {
		t.Errorf("models = %d bytes; the hardlink should be counted once", m.Bytes)
	}
	if !m.Cleanable {
		t.Error("redownloadable models should be cleanable")
	}
	if _, ok := byPath[filepath.Join(root, "app", "node_modules")]; !ok {
		t.Error("node_modules next to package.json not found")
	}

	proj := byPath[filepath.Join(root, "proj", ".venv")]
	if proj.Env == nil || proj.Safety != defs.Redownloadable || proj.Env.Project != filepath.Join(root, "proj") {
		t.Fatalf("project venv: %+v", proj)
	}
	if p := proj.Env.Packages; len(p) != 1 || p[0].Name != "requests" || p[0].Version != "2.32.0" || p[0].Bytes != 8000 || p[0].OtherEnvs != 1 {
		t.Errorf("packages: %+v", p)
	}
	if bare := byPath[filepath.Join(root, "bare", "venv")]; bare.Safety != defs.Unknown || bare.Cleanable {
		t.Errorf("venv without a dependency file should be unknown: %+v", bare)
	}
}

func TestLinksAreNotFollowed(t *testing.T) {
	outside := t.TempDir()
	write(t, filepath.Join(outside, "big"), 1<<20)
	dir := t.TempDir()
	write(t, filepath.Join(dir, "small"), 100)
	link := filepath.Join(dir, "link")

	var err error
	if runtime.GOOS == "windows" {
		err = exec.Command("cmd", "/c", "mklink", "/J", link, outside).Run()
	} else {
		err = os.Symlink(outside, link)
	}
	if err != nil {
		t.Skip("can't create a link here:", err)
	}
	if n, _ := Size(dir); n >= 1<<20 {
		t.Errorf("size %d includes the linked folder", n)
	}
}
