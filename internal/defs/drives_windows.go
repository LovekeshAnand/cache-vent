//go:build windows

package defs

import (
	"syscall"
	"unsafe"
)

var procGetDriveTypeW = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDriveTypeW")

// FixedDrives lists local fixed disks (C:\, D:\, ...), skipping removable, network and optical drives.
func FixedDrives() []string {
	var out []string
	for c := 'A'; c <= 'Z'; c++ {
		root := string(c) + `:\`
		p, _ := syscall.UTF16PtrFromString(root)
		if t, _, _ := procGetDriveTypeW.Call(uintptr(unsafe.Pointer(p))); t == 3 { // DRIVE_FIXED
			out = append(out, root)
		}
	}
	return out
}
