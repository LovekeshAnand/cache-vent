//go:build !windows

package scan

import (
	"os"
	"strconv"
	"syscall"
)

func readDir(dir string) ([]entry, error) {
	des, err := os.ReadDir(dir)
	out := make([]entry, 0, len(des))
	for _, d := range des {
		info, ierr := d.Info()
		if ierr != nil {
			continue
		}
		e := entry{name: d.Name(), isDir: d.IsDir(), link: d.Type()&linkMode != 0, size: info.Size(), mtime: info.ModTime()}
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			e.size = int64(st.Blocks) * 512
			e.key = fileKey{strconv.FormatUint(uint64(st.Dev), 10), uint64(st.Ino)}
		}
		out = append(out, e)
	}
	return out, err
}
