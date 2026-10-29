// Package defs loads definitions and expands their paths for the current OS.
package defs

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// Safety levels. See DEFINITIONS.md.
const (
	Regenerable    = "regenerable"
	Redownloadable = "redownloadable"
	UserData       = "user-data"
	Unknown        = "unknown"
)

// Entry is one thing a tool stores on disk.
type Entry struct {
	ID          string              `yaml:"id"`
	Name        string              `yaml:"name"`
	Paths       map[string][]string `yaml:"paths"`
	Pattern     *Pattern            `yaml:"pattern"`
	EnvOverride string              `yaml:"env_override"`
	Kind        string              `yaml:"kind"`
	Safety      string              `yaml:"safety"`
	IfDeleted   string              `yaml:"if_deleted"`
	Dedupe      bool                `yaml:"dedupe"`
	// CleanOlderThanDays makes clean move only the top-level children untouched
	// for this many days, instead of the whole folder (e.g. temp folders).
	CleanOlderThanDays int      `yaml:"clean_older_than_days"`
	Verified           Verified `yaml:"verified"`
	Tool               string   `yaml:"-"`
}

// Pattern matches folders found anywhere under the scan roots.
type Pattern struct {
	DirName  string   `yaml:"dir_name"`
	Siblings []string `yaml:"siblings"` // only match next to one of these files or folders
}

type Verified struct {
	By      string `yaml:"by"`
	On      string `yaml:"on"`
	Version string `yaml:"version"`
	Source  string `yaml:"source"`
}

type file struct {
	Tool     string  `yaml:"tool"`
	Homepage string  `yaml:"homepage"`
	Entries  []Entry `yaml:"entries"`
}

var (
	idRe     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	winVarRe = regexp.MustCompile(`%([^%]+)%`)
	kinds    = map[string]bool{"model": true, "cache": true, "dependencies": true, "build-output": true, "logs": true,
		"session-history": true, "virtual-disk": true, "user-data": true, "temp": true}
	safeties = map[string]bool{Regenerable: true, Redownloadable: true, UserData: true, Unknown: true}
	osKeys   = map[string]bool{"windows": true, "macos": true, "linux": true}
)

// Cleanable reports whether items with this safety level may be suggested for cleaning.
func Cleanable(safety string) bool { return safety == Regenerable || safety == Redownloadable }

// Load reads and validates every *.yaml file in fsys.
func Load(fsys fs.FS) ([]Entry, error) {
	names, err := fs.Glob(fsys, "*.yaml")
	if err != nil {
		return nil, err
	}
	seen := map[string]string{}
	var all []Entry
	for _, name := range names {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, err
		}
		dec := yaml.NewDecoder(bytes.NewReader(data))
		dec.KnownFields(true)
		var f file
		if err := dec.Decode(&f); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if f.Tool == "" {
			return nil, fmt.Errorf("%s: missing tool", name)
		}
		for _, e := range f.Entries {
			e.Tool = f.Tool
			if err := validate(e); err != nil {
				return nil, fmt.Errorf("%s: %s: %w", name, e.ID, err)
			}
			if prev, ok := seen[e.ID]; ok {
				return nil, fmt.Errorf("%s: id %q already used in %s", name, e.ID, prev)
			}
			seen[e.ID] = name
			all = append(all, e)
		}
	}
	return all, nil
}

func validate(e Entry) error {
	switch {
	case !idRe.MatchString(e.ID):
		return errors.New("id must be kebab-case")
	case e.Name == "":
		return errors.New("missing name")
	case (len(e.Paths) == 0) == (e.Pattern == nil):
		return errors.New("needs exactly one of paths or pattern")
	case e.Pattern != nil && e.Pattern.DirName == "":
		return errors.New("pattern needs dir_name")
	case !kinds[e.Kind]:
		return fmt.Errorf("unknown kind %q", e.Kind)
	case !safeties[e.Safety]:
		return fmt.Errorf("unknown safety %q", e.Safety)
	case e.IfDeleted == "":
		return errors.New("missing if_deleted")
	case Cleanable(e.Safety) && e.Verified.Source == "":
		return fmt.Errorf("%s needs verified.source", e.Safety)
	case e.Safety != Unknown && (e.Verified.By == "" || e.Verified.On == ""):
		return errors.New("needs verified.by and verified.on")
	case e.Dedupe && e.Kind != "model":
		return errors.New("dedupe is only allowed for kind model")
	case e.CleanOlderThanDays < 0:
		return errors.New("clean_older_than_days must be positive")
	}
	for k := range e.Paths {
		if !osKeys[k] {
			return fmt.Errorf("unknown OS %q in paths", k)
		}
	}
	return nil
}

// OSKey is this OS's key in Entry.Paths.
func OSKey() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "macos"
	}
	return "linux"
}

// ResolvedPaths returns the entry's absolute paths on this OS. A set env_override wins.
func (e Entry) ResolvedPaths() []string {
	if e.EnvOverride != "" {
		if v := os.Getenv(e.EnvOverride); v != "" && filepath.IsAbs(v) {
			return []string{filepath.Clean(v)}
		}
	}
	var out []string
	for _, p := range e.Paths[OSKey()] {
		if x := Expand(p); x != "" {
			out = append(out, x)
		}
	}
	return out
}

// Expand turns a definition path into an absolute path: %VAR% on Windows, $VAR
// elsewhere, and ~ everywhere. It returns "" when a variable is unset or the
// result isn't absolute, so callers skip it instead of scanning a literal "%FOO%".
func Expand(p string) string {
	missing := false
	look := func(name string) string {
		v := os.Getenv(name)
		if v == "" {
			missing = true
		}
		return v
	}
	if runtime.GOOS == "windows" {
		p = winVarRe.ReplaceAllStringFunc(p, func(m string) string { return look(m[1 : len(m)-1]) })
	} else {
		p = os.Expand(p, look)
	}
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		p = home + p[1:]
	}
	if missing || !filepath.IsAbs(p) {
		return ""
	}
	return filepath.Clean(p)
}

// PathKey normalizes a path for comparison: cleaned, and case-folded on Windows.
func PathKey(p string) string {
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		return strings.ToLower(p)
	}
	return p
}

// DataDir is where Cache Vent keeps its own files (last scan, history, quarantine).
func DataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if d := os.Getenv("LOCALAPPDATA"); d != "" {
			return filepath.Join(d, "cachevent"), nil
		}
		return "", errors.New("LOCALAPPDATA is not set")
	case "darwin":
		home, err := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "cachevent"), err
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "cachevent"), nil
	}
	home, err := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "cachevent"), err
}
