//go:build windows

package quarantine

import (
	"path/filepath"
	"strings"
)

// ponytail: drive-letter compare; a volume mounted into a folder looks like its parent drive,
// so the rename fails with a clear error instead of moving. Use GetVolumePathName if that matters.
func sameVolume(a, b string) bool {
	return strings.EqualFold(filepath.VolumeName(a), filepath.VolumeName(b))
}

func volumeRoot(path string) string { return filepath.VolumeName(path) + `\` }
