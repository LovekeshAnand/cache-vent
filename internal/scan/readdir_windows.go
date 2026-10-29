//go:build windows

package scan

import (
	"encoding/binary"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// bufs reuses the 64 KB listing buffers (8-byte aligned, as the API requires):
// allocating one per directory cost ~800 MB of garbage on a 12k-folder tree.
var bufs = sync.Pool{New: func() any { return new([8192]uint64) }}

var procGetFileInformationByHandleEx = syscall.NewLazyDLL("kernel32.dll").NewProc("GetFileInformationByHandleEx")

const (
	fileIDBothDirectoryInfo = 10 // FILE_INFO_BY_HANDLE_CLASS
	fileListDirectory       = 0x1
	attrDirectory           = 0x10
	attrReparsePoint        = 0x400
	tagMountPoint           = 0xA0000003
	tagSymlink              = 0xA000000C
	errNoMoreFiles          = syscall.Errno(18)
)

// readDir lists a directory in large batches with FILE_ID_BOTH_DIR_INFO, which
// already carries each entry's allocation size (bytes on disk, after compression
// and sparseness), times and file ID. That makes sizing a tree one call per
// ~64 KB of entries instead of one call per file, and needs no admin rights.
// ponytail: NTFS updates a hardlink's directory entry lazily, so a stale link can
// report an old size; reading the MFT directly (WizTree-style) needs admin.
func readDir(dir string) ([]entry, error) {
	p, err := syscall.UTF16PtrFromString(longPath(dir))
	if err != nil {
		return nil, err
	}
	// OPEN_REPARSE_POINT: if dir itself is a junction, list nothing rather than follow it.
	h, err := syscall.CreateFile(p, fileListDirectory, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(h)

	vol := strings.ToLower(filepath.VolumeName(dir))
	words := bufs.Get().(*[8192]uint64)
	defer bufs.Put(words)
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&words[0])), len(words)*8)
	var out []entry
	for {
		r, _, e := procGetFileInformationByHandleEx.Call(uintptr(h), fileIDBothDirectoryInfo, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if r == 0 {
			if e == errNoMoreFiles {
				return out, nil
			}
			return out, e
		}
		for off := 0; off+104 <= len(buf); {
			rec := buf[off:]
			next := binary.LittleEndian.Uint32(rec[0:])
			nameLen := int(binary.LittleEndian.Uint32(rec[60:]))
			if 104+nameLen > len(rec) {
				break
			}
			// Records are 8-byte aligned, so the name at +104 can be read in place.
			name := syscall.UTF16ToString(unsafe.Slice((*uint16)(unsafe.Pointer(&rec[104])), nameLen/2))
			if name != "." && name != ".." {
				attrs := binary.LittleEndian.Uint32(rec[56:])
				isDir := attrs&attrDirectory != 0
				link := false
				if attrs&attrReparsePoint != 0 {
					// The reparse tag sits in EaSize. Any reparse directory (junction, symlink,
					// cloud folder) is not entered. Files: only links are skipped; dedup and
					// cloud files are counted by what they actually occupy locally.
					tag := binary.LittleEndian.Uint32(rec[64:])
					link = isDir || tag == tagSymlink || tag == tagMountPoint
				}
				out = append(out, entry{
					name:  name,
					isDir: isDir,
					link:  link,
					size:  int64(binary.LittleEndian.Uint64(rec[48:])),
					mtime: filetime(binary.LittleEndian.Uint64(rec[24:])),
					key:   fileKey{vol, binary.LittleEndian.Uint64(rec[96:])},
				})
			}
			if next == 0 {
				break
			}
			off += int(next)
		}
	}
}

func filetime(ft uint64) time.Time {
	if ft == 0 {
		return time.Time{}
	}
	return time.Unix(0, (int64(ft)-116444736000000000)*100)
}

// longPath adds the \\?\ prefix raw Win32 calls need past MAX_PATH.
func longPath(p string) string {
	switch {
	case len(p) < 240, strings.HasPrefix(p, `\\?\`):
		return p
	case strings.HasPrefix(p, `\\`):
		return `\\?\UNC\` + p[2:]
	}
	return `\\?\` + p
}
