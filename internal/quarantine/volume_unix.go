//go:build !windows

package quarantine

import (
	"path/filepath"
	"syscall"
)

func dev(p string) (uint64, bool) {
	var st syscall.Stat_t
	if syscall.Lstat(p, &st) != nil {
		return 0, false
	}
	return uint64(st.Dev), true
}

func sameVolume(a, b string) bool {
	da, ok1 := dev(a)
	db, ok2 := dev(b)
	return ok1 && ok2 && da == db
}

// volumeRoot walks up to the mount point. ponytail: often not writable for users
// (e.g. /home); fine while Windows is the primary target.
func volumeRoot(path string) string {
	d, _ := dev(path)
	for {
		parent := filepath.Dir(path)
		if parent == path {
			return path
		}
		if pd, ok := dev(parent); !ok || pd != d {
			return path
		}
		path = parent
	}
}
