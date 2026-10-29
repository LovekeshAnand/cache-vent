//go:build !windows

package main

// nativeWindow is Windows-only for now; elsewhere the app opens in an app-mode
// Edge or Chrome window.
func nativeWindow(dataDir, url string) bool { return false }
