//go:build windows

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	webview2 "github.com/jchv/go-webview2"
)

// The window's message loop must run on the main OS thread.
func init() { runtime.LockOSThread() }

// nativeWindow shows the app in its own window, drawn by the WebView2 engine that
// ships with Windows: no browser, tabs, history or extensions. It blocks until the
// window is closed, and reports false if the WebView2 runtime is missing.
func nativeWindow(dataDir, url string) bool {
	os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", strings.Join(leanBrowserFlags, " "))
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		DataPath:      filepath.Join(dataDir, "webview"),
		AutoFocus:     true,
		WindowOptions: webview2.WindowOptions{Title: "Cache Vent", IconId: 1, Width: 1200, Height: 820, Center: true},
	})
	if w == nil {
		return false
	}
	defer w.Destroy()
	w.SetSize(880, 560, webview2.HintMin)
	w.Navigate(url)
	w.Run()
	return true
}
