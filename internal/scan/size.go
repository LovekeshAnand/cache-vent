package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// linkMode covers symlinks, junctions and other reparse points (e.g. cloud placeholders).
const linkMode = fs.ModeSymlink | fs.ModeIrregular

// dedupeMin: files at least this big are counted once across hardlinks (by file ID).
// ponytail: smaller hardlinked files count per link; tracking every file ID costs
// ~40 B per file of memory for a few % accuracy.
const dedupeMin = 64 << 10

type fileKey struct {
	vol string
	id  uint64
}

// entry is one directory entry as readDir reports it; size is bytes on disk.
type entry struct {
	name  string
	isDir bool
	link  bool // symlink, junction or other reparse directory: never followed
	size  int64
	mtime time.Time
	key   fileKey
}

type sizer struct {
	sem     chan struct{} // slots for extra walker goroutines
	skipped atomic.Int64  // directories that couldn't be read
	dirs    atomic.Int64
	onDir   func(dirs int64)
}

func newSizer() *sizer {
	// I/O bound: more readers than cores keeps SSD queues full; HDDs cap themselves.
	return &sizer{sem: make(chan struct{}, max(8, 4*runtime.NumCPU()))}
}

// Size measures one path the same way Run does.
func Size(path string) (int64, time.Time) { return newSizer().size(path) }

// spawn runs f on a new goroutine when a slot is free, and inline otherwise. A
// goroutine per folder parked on a semaphore held ~400 MB on a 1M-file drive;
// this keeps the walker count at the slot limit and never blocks.
func (s *sizer) spawn(wg *sync.WaitGroup, f func()) {
	select {
	case s.sem <- struct{}{}:
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-s.sem }()
			f()
		}()
	default:
		f()
	}
}

func (s *sizer) list(dir string) []entry {
	ents, err := readDir(dir)
	if err != nil {
		s.skipped.Add(1)
	}
	if n := s.dirs.Add(1); s.onDir != nil && n%2048 == 0 {
		s.onDir(n)
	}
	return ents
}

// tally accumulates one item's size. Hardlinks are counted once per item, so an
// item's size is what removing it frees, and an item nested in another one (a
// project inside the temp folder) still measures its own bytes.
type tally struct {
	bytes  atomic.Int64
	newest atomic.Int64 // unix nanos
	mu     sync.Mutex
	seen   map[fileKey]bool
}

// first reports whether a file is seen for the first time in this item.
func (t *tally) first(k fileKey) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.seen[k] {
		return false
	}
	t.seen[k] = true
	return true
}

func (t *tally) touch(m time.Time) {
	if m.IsZero() {
		return
	}
	n := m.UnixNano()
	for {
		cur := t.newest.Load()
		if n <= cur || t.newest.CompareAndSwap(cur, n) {
			return
		}
	}
}

// count adds a listing's files to t and returns the subdirectories to descend.
func (s *sizer) count(dir string, ents []entry, t *tally) []string {
	var sum int64
	var subs []string
	for _, e := range ents {
		if e.link {
			continue
		}
		t.touch(e.mtime)
		if e.isDir {
			subs = append(subs, filepath.Join(dir, e.name))
			continue
		}
		if e.size >= dedupeMin && e.key.id != 0 && !t.first(e.key) {
			continue
		}
		sum += e.size
	}
	t.bytes.Add(sum)
	return subs
}

// size sums bytes on disk under root and finds the newest modification time,
// listing subdirectories in parallel. It never follows links or junctions.
func (s *sizer) size(root string) (int64, time.Time) {
	fi, err := os.Lstat(root)
	if err != nil || fi.Mode()&linkMode != 0 {
		return 0, time.Time{}
	}
	t := tally{seen: map[fileKey]bool{}}
	if !fi.IsDir() {
		// A single file (e.g. a temp file): find its entry in the parent listing.
		for _, e := range s.list(filepath.Dir(root)) {
			if strings.EqualFold(e.name, fi.Name()) && !e.link {
				return e.size, e.mtime
			}
		}
		return fi.Size(), fi.ModTime()
	}
	t.touch(fi.ModTime())
	var wg sync.WaitGroup
	var walk func(dir string, ents []entry)
	walk = func(dir string, ents []entry) {
		for _, sub := range s.count(dir, ents, &t) {
			s.spawn(&wg, func() { walk(sub, s.list(sub)) })
		}
	}
	walk(root, s.list(root))
	wg.Wait()
	return t.bytes.Load(), time.Unix(0, t.newest.Load())
}
