package scan

import (
	"io/fs"
	"path/filepath"
	"time"
)

// linkMode covers symlinks, junctions and other reparse points (e.g. cloud placeholders).
const linkMode = fs.ModeSymlink | fs.ModeIrregular

type sizer struct {
	seen     map[fileKey]bool // hardlinked files already counted
	clusters map[string]int64 // Windows cluster size per volume
	skipped  int              // entries that couldn't be read
}

func newSizer() *sizer { return &sizer{seen: map[fileKey]bool{}, clusters: map[string]int64{}} }

// Size measures one path the same way Run does.
func Size(path string) (int64, time.Time) { return newSizer().size(path) }

// size sums bytes on disk under root and finds the newest modification time.
// It never follows links or junctions, and counts a hardlinked file once per sizer.
func (s *sizer) size(root string) (int64, time.Time) {
	var total int64
	var newest time.Time
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			s.skipped++
			return nil
		}
		if d.Type()&linkMode != 0 {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			s.skipped++
			return nil
		}
		if info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		if d.IsDir() {
			return nil
		}
		n, key := s.diskSize(path, info)
		if key != nil {
			if s.seen[*key] {
				return nil
			}
			s.seen[*key] = true
		}
		total += n
		return nil
	})
	return total, newest
}
