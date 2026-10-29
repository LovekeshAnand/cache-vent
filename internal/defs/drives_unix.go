//go:build !windows

package defs

import "os"

// FixedDrives returns the home folder: walking / would enter /proc, /sys and network mounts.
// ponytail: list real mounts from /proc/mounts and getfsstat if whole-disk scans matter here.
func FixedDrives() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{home}
}
