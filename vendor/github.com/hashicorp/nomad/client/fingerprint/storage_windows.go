package fingerprint

import (
	"fmt"
	"path/filepath"
	"syscall"
)

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zstorage_windows.go storage_windows.go

//sys	getDiskFreeSpaceEx(dirName *uint16, availableFreeBytes *uint64, totalBytes *uint64, totalFreeBytes *uint64) (err error) = kernel32.GetDiskFreeSpaceExW

// diskFree inspects the filesystem for path and returns the volume name and
// the total and free bytes available on the file system.
func (f *StorageFingerprint) diskFree(path string) (volume string, total, free uint64, err error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to determine absolute path for %s", path)
	}

	volume = filepath.VolumeName(absPath)

	absPathp, err := syscall.UTF16PtrFromString(absPath)
	if err != nil {
		return "", 0, 0, fmt.Errorf("failed to convert \"%s\" to UTF16: %v", absPath, err)
	}

	if err := getDiskFreeSpaceEx(absPathp, nil, &total, &free); err != nil {
		return "", 0, 0, fmt.Errorf("failed to get free disk space for %s: %v", absPath, err)
	}

	return volume, total, free, nil
}
