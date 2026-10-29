// Package envs inspects Python environments: which project owns each one,
// whether that project still exists, and which packages take the space.
// Verdicts are per environment; single packages are never cleaned, because
// removing one breaks whatever depends on it.
package envs

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"cachevent/internal/defs"
)

// venvDocs: "A virtual environment is ... considered as disposable – it should be
// simple to delete and recreate it from scratch."
const venvDocs = "https://docs.python.org/3/library/venv.html"

type Package struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Bytes     int64  `json:"bytes"`
	OtherEnvs int    `json:"other_envs"` // other scanned environments holding this exact version
}

type Info struct {
	Manager  string    `json:"manager"` // venv, pipenv, poetry, conda
	Project  string    `json:"project,omitempty"`
	Orphaned bool      `json:"orphaned"`
	Packages []Package `json:"packages,omitempty"` // largest first
}

// Verdict is what the analyzer says about one environment. ID works like a definition ID.
type Verdict struct {
	ID, Name, Safety, IfDeleted, Source string
	Info                                Info
}

type Found struct {
	Path    string
	Manager string
}

// Central folders where environment managers keep environments outside projects.
var central = []struct {
	manager string
	env     string // overrides paths when set
	paths   map[string][]string
}{
	// https://python-poetry.org/docs/configuration/#cache-dir
	{"poetry", "", map[string][]string{
		"windows": {`%LOCALAPPDATA%\pypoetry\Cache\virtualenvs`},
		"macos":   {"~/Library/Caches/pypoetry/virtualenvs"},
		"linux":   {"~/.cache/pypoetry/virtualenvs"},
	}},
	// https://pipenv.pypa.io/en/latest/virtualenv.html
	{"pipenv", "WORKON_HOME", map[string][]string{
		"windows": {`%USERPROFILE%\.virtualenvs`},
		"macos":   {"~/.local/share/virtualenvs"},
		"linux":   {"~/.local/share/virtualenvs"},
	}},
	{"conda", "", map[string][]string{
		"windows": {`%USERPROFILE%\miniconda3\envs`, `%USERPROFILE%\anaconda3\envs`, `%USERPROFILE%\miniforge3\envs`, `%USERPROFILE%\.conda\envs`},
		"macos":   {"~/miniconda3/envs", "~/anaconda3/envs", "~/miniforge3/envs", "~/.conda/envs"},
		"linux":   {"~/miniconda3/envs", "~/anaconda3/envs", "~/miniforge3/envs", "~/.conda/envs"},
	}},
}

// Find lists the environments in the central folders.
func Find() []Found {
	var out []Found
	for _, c := range central {
		dirs := c.paths[defs.OSKey()]
		if v := os.Getenv(c.env); c.env != "" && v != "" {
			dirs = []string{v}
		}
		for _, d := range dirs {
			d = defs.Expand(d)
			if d == "" {
				continue
			}
			ents, _ := os.ReadDir(d)
			for _, e := range ents {
				if e.IsDir() { // false for junctions and symlinks
					out = append(out, Found{filepath.Join(d, e.Name()), c.manager})
				}
			}
		}
	}
	return out
}

// IsVenv reports whether dir is a virtual environment (venv, virtualenv, uv, poetry, pipenv).
func IsVenv(dir string) bool {
	fi, err := os.Lstat(filepath.Join(dir, "pyvenv.cfg"))
	return err == nil && fi.Mode().IsRegular()
}

func Inspect(dir, manager string) Verdict {
	info := Info{Manager: manager, Packages: packages(dir)}
	v := Verdict{Safety: defs.Unknown, IfDeleted: "Its owner couldn't be determined, so it's left alone."}
	switch manager {
	case "venv":
		v.ID, v.Name = "python-venv", "Python virtual environment"
		info.Project = filepath.Dir(dir)
		if hasManifest(info.Project) {
			v.Safety, v.Source = defs.Redownloadable, venvDocs
			v.IfDeleted = "Recreate it and reinstall from the project's dependency file (e.g. `uv sync` or `pip install -r requirements.txt`)."
		} else {
			v.IfDeleted = "No dependency file next to it, so its packages may not be reproducible."
		}
	case "pipenv":
		v.ID, v.Name = "pipenv-env", "Pipenv environment"
		b, err := os.ReadFile(filepath.Join(dir, ".project"))
		if err != nil {
			break
		}
		info.Project = strings.TrimSpace(string(b))
		switch {
		case exists(info.Project):
			v.Safety, v.Source = defs.Redownloadable, venvDocs
			v.IfDeleted = "Run `pipenv install` in the project to recreate it."
		// Only call it orphaned if the folder that held the project is still there;
		// otherwise it may live on a drive that isn't plugged in.
		case notExist(info.Project) && exists(filepath.Dir(info.Project)):
			info.Orphaned = true
			v.Safety, v.Source = defs.Redownloadable, venvDocs
			v.IfDeleted = "Its project folder no longer exists, so nothing uses it."
		}
	case "poetry":
		// The owner is filled in by the scan via PoetryHash, once it has seen the projects.
		v.ID, v.Name = "poetry-env", "Poetry environment"
		v.Safety, v.Source = defs.Redownloadable, venvDocs
		v.IfDeleted = "Poetry recreates it on the next `poetry install` in its project."
	case "conda":
		v.ID, v.Name = "conda-env", "Conda environment"
		v.IfDeleted = "Conda environments are often built by hand without an environment file; check before removing."
	}
	v.Info = info
	return v
}

// poetryEnvRe matches Poetry's env folder names: <name>-<8-char path hash>-py<version>.
var poetryEnvRe = regexp.MustCompile(`^.+-([A-Za-z0-9_-]{8})-py\d+\.\d+$`)

// PoetryEnvHash extracts the project-path hash from a Poetry env folder name.
func PoetryEnvHash(envDir string) (string, bool) {
	m := poetryEnvRe.FindStringSubmatch(filepath.Base(envDir))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// PoetryHash is the hash Poetry puts in env names for a project folder:
// urlsafe-base64(sha256(normcase(path)))[:8], as in poetry's generate_env_name.
// ponytail: poetry also resolves symlinks (realpath) first; projects reached through a
// link won't match and just show no owner.
func PoetryHash(projectDir string) string {
	p := filepath.Clean(projectDir)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	sum := sha256.Sum256([]byte(p))
	return base64.URLEncoding.EncodeToString(sum[:])[:8]
}

func hasManifest(dir string) bool {
	for _, f := range []string{"pyproject.toml", "requirements.txt", "Pipfile", "setup.py", "setup.cfg", "uv.lock", "poetry.lock"} {
		if exists(filepath.Join(dir, f)) {
			return true
		}
	}
	m, _ := filepath.Glob(filepath.Join(dir, "requirements*.txt"))
	return len(m) > 0
}

// packages reads installed packages from *.dist-info folders; sizes come from RECORD.
func packages(env string) []Package {
	dirs, _ := filepath.Glob(filepath.Join(env, "lib", "python*", "site-packages"))
	dirs = append(dirs, filepath.Join(env, "Lib", "site-packages"))
	seen := map[string]bool{}
	out := []Package{}
	for _, sp := range dirs {
		infos, _ := filepath.Glob(filepath.Join(sp, "*.dist-info"))
		for _, d := range infos {
			if seen[defs.PathKey(d)] {
				continue
			}
			seen[defs.PathKey(d)] = true
			base := strings.TrimSuffix(filepath.Base(d), ".dist-info")
			i := strings.LastIndex(base, "-")
			if i <= 0 {
				continue
			}
			name := strings.NewReplacer("_", "-", ".", "-").Replace(strings.ToLower(base[:i]))
			out = append(out, Package{Name: name, Version: base[i+1:], Bytes: recordBytes(filepath.Join(d, "RECORD"))})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Bytes > out[j].Bytes })
	return out
}

// recordBytes sums the size column of a wheel RECORD file (path,hash,size).
func recordBytes(path string) int64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	var n int64
	for {
		rec, err := r.Read()
		if err == io.EOF || err != nil {
			return n
		}
		if len(rec) >= 3 {
			if v, err := strconv.ParseInt(rec[2], 10, 64); err == nil {
				n += v
			}
		}
	}
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func notExist(p string) bool { _, err := os.Stat(p); return errors.Is(err, fs.ErrNotExist) }
