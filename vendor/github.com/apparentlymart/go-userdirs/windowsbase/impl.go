// +build windows

package windowsbase

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32                  = windows.NewLazyDLL("Shell32.dll")
	ole32                    = windows.NewLazyDLL("Ole32.dll")
	procSHGetKnownFolderPath = shell32.NewProc("SHGetKnownFolderPath")
	procCoTaskMemFree        = ole32.NewProc("CoTaskMemFree")
)

func knownFolderDir(fid *FolderID) (string, error) {
	var path uintptr
	err := shGetKnownFolderPath(fid, 0, 0, &path)
	if err != nil {
		return "", err
	}
	defer coTaskMemFree(path)
	dir := syscall.UTF16ToString((*[1 << 16]uint16)(unsafe.Pointer(path))[:])
	return dir, nil
}

func shGetKnownFolderPath(fid *FolderID, dwFlags uint32, hToken syscall.Handle, pszPath *uintptr) (retval error) {
	r0, _, _ := procSHGetKnownFolderPath.Call(uintptr(unsafe.Pointer(fid)), uintptr(dwFlags), uintptr(hToken), uintptr(unsafe.Pointer(pszPath)), 0, 0)
	if r0 != 0 {
		return syscall.Errno(r0)
	}
	return nil
}

func coTaskMemFree(pv uintptr) {
	procCoTaskMemFree.Call(uintptr(pv), 0, 0)
}
