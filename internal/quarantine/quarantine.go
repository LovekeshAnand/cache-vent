// Package quarantine moves cleaned items aside instead of deleting them. A move
// is a rename on the same volume, so it's instant and costs no space. Items can
// be restored until they are purged.
package quarantine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"cachevent/internal/defs"
)

const (
	volumeDirName = ".cachevent-quarantine"
	manifestName  = "manifest.json"
	payloadName   = "payload"
	rootsFile     = "quarantine-roots.txt"
	// MaxAge is how long items stay before they are purged.
	MaxAge = 14 * 24 * time.Hour
)

type Manifest struct {
	ID            string    `json:"id"`
	OriginalPath  string    `json:"original_path"`
	DefinitionID  string    `json:"definition_id"`
	Bytes         int64     `json:"bytes"`
	QuarantinedAt time.Time `json:"quarantined_at"`
	Dir           string    `json:"-"` // where the item lives in quarantine
}

// Store keeps its own quarantine in DataDir and remembers per-volume ones there.
type Store struct{ DataDir string }

func (s Store) home() string { return filepath.Join(s.DataDir, "quarantine") }

// Move puts path into quarantine on its own volume.
func (s Store) Move(path, definitionID string, bytes int64) (Manifest, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return Manifest{}, err
	}
	fi, err := os.Lstat(path)
	if err != nil {
		return Manifest{}, err
	}
	if fi.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
		return Manifest{}, fmt.Errorf("%s is a link or junction; not moving it", path)
	}
	for _, r := range append(s.Roots(), s.DataDir) {
		if defs.Within(path, r) {
			return Manifest{}, fmt.Errorf("%s is inside Cache Vent's own folders", path)
		}
		if defs.Within(r, path) {
			return Manifest{}, fmt.Errorf("%s contains Cache Vent's own folders; not moving it", path)
		}
	}
	if err := os.MkdirAll(s.home(), 0o755); err != nil {
		return Manifest{}, err
	}
	root := s.home()
	if !sameVolume(path, s.DataDir) {
		root = filepath.Join(volumeRoot(path), volumeDirName)
		if err := os.MkdirAll(root, 0o755); err != nil {
			return Manifest{}, fmt.Errorf("can't create quarantine on %s's volume: %w", path, err)
		}
		if err := s.remember(root); err != nil {
			return Manifest{}, err
		}
	}

	dir, err := os.MkdirTemp(root, definitionID+"-")
	if err != nil {
		return Manifest{}, err
	}
	m := Manifest{ID: filepath.Base(dir), OriginalPath: path, DefinitionID: definitionID, Bytes: bytes,
		QuarantinedAt: time.Now().UTC(), Dir: dir}
	// Manifest first: a crash after the rename still leaves a restorable item.
	if err := writeJSON(filepath.Join(dir, manifestName), m); err != nil {
		os.Remove(dir)
		return Manifest{}, err
	}
	if err := os.Rename(path, filepath.Join(dir, payloadName)); err != nil {
		os.Remove(filepath.Join(dir, manifestName))
		os.Remove(dir)
		// A rename either moves everything or nothing, so a file in use leaves the item intact.
		return Manifest{}, fmt.Errorf("%s couldn't be moved; close any program using it and retry: %w", path, err)
	}
	return m, nil
}

// List returns quarantined items, oldest first.
func (s Store) List() []Manifest {
	out := []Manifest{}
	for _, root := range s.Roots() {
		ents, _ := os.ReadDir(root)
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(root, e.Name())
			b, err := os.ReadFile(filepath.Join(dir, manifestName))
			var m Manifest
			if err != nil || json.Unmarshal(b, &m) != nil {
				continue
			}
			m.Dir = dir
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].QuarantinedAt.Before(out[j].QuarantinedAt) })
	return out
}

// Restore moves an item back to where it came from. It never overwrites.
func (s Store) Restore(id string) (Manifest, error) {
	m, err := s.find(id)
	if err != nil {
		return m, err
	}
	payload := filepath.Join(m.Dir, payloadName)
	if _, err := os.Lstat(payload); errors.Is(err, fs.ErrNotExist) {
		// The move never happened (crash between manifest and rename); drop the empty record.
		os.Remove(filepath.Join(m.Dir, manifestName))
		os.Remove(m.Dir)
		return m, fmt.Errorf("%s: nothing to restore, the original was never moved", id)
	}
	if _, err := os.Lstat(m.OriginalPath); !errors.Is(err, fs.ErrNotExist) {
		return m, fmt.Errorf("%s already exists; move it away first", m.OriginalPath)
	}
	if err := os.MkdirAll(filepath.Dir(m.OriginalPath), 0o755); err != nil {
		return m, err
	}
	if err := os.Rename(payload, m.OriginalPath); err != nil {
		return m, err
	}
	os.Remove(filepath.Join(m.Dir, manifestName))
	os.Remove(m.Dir)
	return m, nil
}

// Purge permanently removes items quarantined longer than olderThan (0 means all).
// This is the only place Cache Vent deletes files.
func (s Store) Purge(olderThan time.Duration) (n int, bytes int64, err error) {
	return s.purge(func(m Manifest) bool { return olderThan <= 0 || time.Since(m.QuarantinedAt) >= olderThan })
}

// PurgeIDs permanently removes the given quarantined items, e.g. right after a
// move when the user chose to delete instead of keep.
func (s Store) PurgeIDs(ids []string) (n int, bytes int64, err error) {
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	return s.purge(func(m Manifest) bool { return want[m.ID] })
}

func (s Store) purge(match func(Manifest) bool) (n int, bytes int64, err error) {
	var errs []error
	for _, m := range s.List() {
		if !match(m) {
			continue
		}
		// Payload first: if this fails halfway, the manifest keeps the item listed and retried.
		if e := os.RemoveAll(filepath.Join(m.Dir, payloadName)); e != nil {
			errs = append(errs, e)
			continue
		}
		if e := os.RemoveAll(m.Dir); e != nil {
			errs = append(errs, e)
			continue
		}
		n++
		bytes += m.Bytes
	}
	return n, bytes, errors.Join(errs...)
}

func (s Store) find(id string) (Manifest, error) {
	for _, m := range s.List() {
		if m.ID == id {
			return m, nil
		}
	}
	return Manifest{}, fmt.Errorf("%s: not in quarantine", id)
}

// Roots is the data-dir quarantine plus every per-volume one used so far.
func (s Store) Roots() []string {
	roots := []string{s.home()}
	b, _ := os.ReadFile(filepath.Join(s.DataDir, rootsFile))
	for _, line := range strings.Split(string(b), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			roots = append(roots, line)
		}
	}
	return roots
}

func (s Store) remember(root string) error {
	for _, r := range s.Roots() {
		if r == root {
			return nil
		}
	}
	f, err := os.OpenFile(filepath.Join(s.DataDir, rootsFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, root)
	return err
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
