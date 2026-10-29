//go:build windows

package scan

import (
	"io/fs"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procGetCompressedFileSizeW = kernel32.NewProc("GetCompressedFileSizeW")
	procGetDiskFreeSpaceW      = kernel32.NewProc("GetDiskFreeSpaceW")
)

type fileKey struct{ vol, hi, lo uint32 }

// diskSize returns the bytes a file occupies on disk (compressed/sparse aware,
// rounded up to clusters) and, if it has other hardlinks, a key to count it once.
// ponytail: files MFT-resident (< ~700 B) count as one cluster, Explorer shows 0; files over
// MAX_PATH fall back to their length.
func (s *sizer) diskSize(path string, info fs.FileInfo) (int64, *fileKey) {
	n := info.Size()
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return n, nil
	}
	var hi uint32
	lo, _, e := procGetCompressedFileSizeW.Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&hi)))
	if errno, _ := e.(syscall.Errno); uint32(lo) != 0xFFFFFFFF || errno == 0 {
		n = int64(hi)<<32 | int64(uint32(lo))
	}
	if c := s.cluster(path); c > 0 {
		n = (n + c - 1) / c * c
	}

	// ponytail: only files >= 1 MiB are checked for hardlinks (it costs an open). Small
	// hardlinked files, e.g. uv venvs linked from the uv cache, are counted per link.
	if info.Size() < 1<<20 {
		return n, nil
	}
	h, err := syscall.CreateFile(p, 0, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil, syscall.OPEN_EXISTING, syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return n, nil
	}
	defer syscall.CloseHandle(h)
	var bi syscall.ByHandleFileInformation
	if syscall.GetFileInformationByHandle(h, &bi) != nil || bi.NumberOfLinks < 2 {
		return n, nil
	}
	return n, &fileKey{bi.VolumeSerialNumber, bi.FileIndexHigh, bi.FileIndexLow}
}

func (s *sizer) cluster(path string) int64 {
	vol := filepath.VolumeName(path)
	if c, ok := s.clusters[vol]; ok {
		return c
	}
	var c int64
	if root, err := syscall.UTF16PtrFromString(vol + `\`); err == nil {
		var spc, bps, free, total uint32
		r, _, _ := procGetDiskFreeSpaceW.Call(uintptr(unsafe.Pointer(root)), uintptr(unsafe.Pointer(&spc)),
			uintptr(unsafe.Pointer(&bps)), uintptr(unsafe.Pointer(&free)), uintptr(unsafe.Pointer(&total)))
		if r != 0 {
			c = int64(spc) * int64(bps)
		}
	}
	s.clusters[vol] = c
	return c
}
