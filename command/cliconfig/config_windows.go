// +build windows

package cliconfig

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell         = syscall.MustLoadDLL("Shell32.dll")
	getFolderPath = shell.MustFindProc("SHGetFolderPathW")
)

const CSIDL_APPDATA = 26

func configFile() (string, error) {
	dir, err := homeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "terraform.rc"), nil
}

func configDir() (string, error) {
	dir, err := homeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "terraform.d"), nil
}

func homeDir() (string, error) {
	b := make([]uint16, syscall.MAX_PATH)

	// See: http://msdn.microsoft.com/en-us/library/windows/desktop/bb762181(v=vs.85).aspx
	r, _, err := getFolderPath.Call(0, CSIDL_APPDATA, 0, 0, uintptr(unsafe.Pointer(&b[0])))
	if uint32(r) != 0 {
		return "", err
	}

	return syscall.UTF16ToString(b), nil
}

func replaceFileAtomic(source, destination string) error {
	// On Windows, renaming one file over another is not atomic and certain
	// error conditions can result in having only the source file and nothing
	// at the destination file. Instead, we need to call into the MoveFileEx
	// Windows API function.
	srcPtr, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return &os.LinkError{"replace", source, destination, err}
	}
	destPtr, err := syscall.UTF16PtrFromString(destination)
	if err != nil {
		return &os.LinkError{"replace", source, destination, err}
	}

	flags := uint32(windows.MOVEFILE_REPLACE_EXISTING | windows.MOVEFILE_WRITE_THROUGH)
	err = windows.MoveFileEx(srcPtr, destPtr, flags)
	if err != nil {
		return &os.LinkError{"replace", source, destination, err}
	}
	return nil
}
