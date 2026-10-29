package main

// The desktop app: a local page served by this binary, shown in an app-mode
// browser window (Edge ships with Windows). No extra runtime or dependency.

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"cachevent/internal/clean"
	"cachevent/internal/defs"
	"cachevent/internal/quarantine"
	"cachevent/internal/scan"
)

//go:embed ui/index.html
var indexHTML []byte

// idleExit: with no window and no requests this long, the app quits. The page
// polls every few seconds while open, so this only fires once it's closed.
const idleExit = 2 * time.Minute

type uiServer struct {
	s     *session
	token string
	host  string

	mu       sync.Mutex
	scanning bool
	found    int
	scanErr  string
	lastSeen time.Time
}

func cmdUI(args []string) error {
	fs := flag.NewFlagSet("ui", flag.ExitOnError)
	noWindow := fs.Bool("no-window", false, "only print the address; open it in any browser")
	fs.Parse(args)

	s, err := newSession()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	tok := make([]byte, 16)
	rand.Read(tok)
	v := &uiServer{s: s, token: hex.EncodeToString(tok), host: ln.Addr().String(), lastSeen: time.Now()}
	go http.Serve(ln, v.routes())

	// The token rides in the fragment: it never reaches logs or Referer headers.
	url := fmt.Sprintf("http://%s/#%s", v.host, v.token)
	if *noWindow {
		fmt.Println(url)
		select {}
	}
	closed := make(chan struct{})
	if !openWindow(s.dataDir, url, closed) {
		fmt.Println("Open", url)
		openBrowser(url)
	}
	for {
		select {
		case <-closed:
			return nil
		case <-time.After(10 * time.Second):
			if v.idle() {
				return nil
			}
		}
	}
}

func (v *uiServer) idle() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return !v.scanning && time.Since(v.lastSeen) > idleExit
}

func (v *uiServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'")
		w.Write(indexHTML)
	})
	mux.HandleFunc("GET /api/state", v.state)
	mux.HandleFunc("POST /api/scan", v.startScan)
	mux.HandleFunc("POST /api/plan", v.plan)
	mux.HandleFunc("POST /api/clean", v.clean)
	mux.HandleFunc("POST /api/restore", v.restore)
	mux.HandleFunc("POST /api/purge", v.purge)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Host check blocks DNS rebinding; the token blocks other local pages, since a
		// custom header can't be sent cross-origin without a preflight we never answer.
		if r.Host != v.host {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.URL.Path != "/" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Token")), []byte(v.token)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		v.mu.Lock()
		v.lastSeen = time.Now()
		v.mu.Unlock()
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		mux.ServeHTTP(w, r)
	})
}

func (v *uiServer) state(w http.ResponseWriter, r *http.Request) {
	last, ok, err := v.s.lastScan()
	home, _ := os.UserHomeDir()
	v.mu.Lock()
	st := map[string]any{
		"scanning": v.scanning, "found": v.found, "scan_error": v.scanErr,
		"quarantine": v.s.store.List(), "max_age_days": int(quarantine.MaxAge.Hours() / 24), "home": home,
	}
	v.mu.Unlock()
	if ok {
		st["scan"] = last
	}
	if err != nil {
		st["scan_error"] = err.Error()
	}
	writeJSON(w, st)
}

func (v *uiServer) startScan(w http.ResponseWriter, r *http.Request) {
	var req struct{ Roots []string }
	if !readJSON(w, r, &req) {
		return
	}
	for _, root := range req.Roots {
		if !filepath.IsAbs(root) {
			http.Error(w, root+": use a full path, like D:\\", http.StatusBadRequest)
			return
		}
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.scanning {
		http.Error(w, "a scan is already running", http.StatusConflict)
		return
	}
	v.scanning, v.found, v.scanErr = true, 0, ""
	go func() {
		_, err := v.s.scan(req.Roots, func(n int) { v.mu.Lock(); v.found = n; v.mu.Unlock() })
		v.mu.Lock()
		v.scanning = false
		if err != nil {
			v.scanErr = err.Error()
		}
		v.mu.Unlock()
	}()
	w.WriteHeader(http.StatusAccepted)
}

func (v *uiServer) busy(w http.ResponseWriter) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.scanning {
		http.Error(w, "wait for the scan to finish", http.StatusConflict)
	}
	return v.scanning
}

// plan answers "what exactly would be moved?" for the confirmation dialog.
func (v *uiServer) plan(w http.ResponseWriter, r *http.Request) {
	var req struct{ Paths []string }
	if !readJSON(w, r, &req) || v.busy(w) {
		return
	}
	targets, err := v.s.plan(req.Paths)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if targets == nil {
		targets = []clean.Target{}
	}
	writeJSON(w, targets)
}

// clean plans again rather than trusting the client, then moves.
func (v *uiServer) clean(w http.ResponseWriter, r *http.Request) {
	var req struct{ Paths []string }
	if !readJSON(w, r, &req) || v.busy(w) {
		return
	}
	targets, err := v.s.plan(req.Paths)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res := v.s.clean(targets)
	v.markCleaned(res.Moved)
	writeJSON(w, res)
}

// markCleaned drops moved items from the saved scan so the list stays truthful.
func (v *uiServer) markCleaned(moved []quarantine.Manifest) {
	last, ok, err := v.s.lastScan()
	if !ok || err != nil || len(moved) == 0 {
		return
	}
	gone := map[string]bool{}
	for _, m := range moved {
		gone[m.OriginalPath] = true
	}
	kept := last.Items[:0]
	for _, it := range last.Items {
		if gone[it.Path] {
			continue
		}
		for _, m := range moved {
			if defs.Within(m.OriginalPath, it.Path) { // only some children moved (temp); re-measure
				it.Bytes, it.LastUsed = scan.Size(it.Path)
				break
			}
		}
		kept = append(kept, it)
	}
	last.Items = kept
	v.s.save(last)
}

func (v *uiServer) restore(w http.ResponseWriter, r *http.Request) {
	var req struct{ IDs []string }
	if !readJSON(w, r, &req) {
		return
	}
	var errs []error
	for _, id := range req.IDs {
		if _, err := v.s.store.Restore(id); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, map[string]any{"restored": len(req.IDs)})
}

func (v *uiServer) purge(w http.ResponseWriter, r *http.Request) {
	n, b, err := v.s.store.Purge(0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"purged": n, "bytes": b})
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

// openWindow starts Edge or Chrome in app mode (no tabs or address bar) with its
// own profile, and closes `closed` when that window is shut. It reports false
// if neither browser is installed.
func openWindow(dataDir, url string, closed chan struct{}) bool {
	for _, exe := range appBrowsers() {
		if _, err := os.Stat(exe); err != nil {
			if exe, err = exec.LookPath(exe); err != nil {
				continue
			}
		}
		cmd := exec.Command(exe, "--app="+url, "--user-data-dir="+filepath.Join(dataDir, "ui-profile"),
			"--no-first-run", "--no-default-browser-check", "--window-size=1180,820")
		if cmd.Start() != nil {
			continue
		}
		go func() {
			start := time.Now()
			cmd.Wait()
			// A quick exit means an already-running window took the URL over; the idle
			// timer ends the app then. A long-lived process ending means the window closed.
			if time.Since(start) > 5*time.Second {
				close(closed)
			}
		}()
		return true
	}
	return false
}

func appBrowsers() []string {
	switch runtime.GOOS {
	case "windows":
		var out []string
		for _, base := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")} {
			if base != "" {
				out = append(out, filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"),
					filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"))
			}
		}
		return out
	case "darwin":
		return []string{"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"}
	}
	return []string{"microsoft-edge", "google-chrome", "chromium", "chromium-browser"}
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}
