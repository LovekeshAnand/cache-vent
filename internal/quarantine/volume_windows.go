//go:build windows

package quarantine

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var procGetVolumePathNameW = syscall.NewLazyDLL("kernel32.dll").NewProc("GetVolumePathNameW")

// volumeRoot is the mount point holding path: a drive root, or a folder another
// volume is mounted into. It works for paths that don't exist yet.
func volumeRoot(path string) string {
	buf := make([]uint16, 1024)
	if p, err := syscall.UTF16PtrFromString(path); err == nil {
		if r, _, _ := procGetVolumePathNameW.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf))); r != 0 {
			return syscall.UTF16ToString(buf)
		}
	}
	return filepath.VolumeName(path) + `\`
}

func sameVolume(a, b string) bool { return strings.EqualFold(volumeRoot(a), volumeRoot(b)) }
