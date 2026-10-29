//go:build !windows

package scan

import (
	"io/fs"
	"syscall"
)

type fileKey struct{ dev, ino uint64 }

// diskSize returns the bytes a file occupies on disk and, if it has other
// hardlinks, a key to count it once.
func (s *sizer) diskSize(path string, info fs.FileInfo) (int64, *fileKey) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return info.Size(), nil
	}
	n := int64(st.Blocks) * 512
	if st.Nlink > 1 {
		return n, &fileKey{uint64(st.Dev), uint64(st.Ino)}
	}
	return n, nil
}
