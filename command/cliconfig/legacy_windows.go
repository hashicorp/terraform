// +build windows

package cliconfig

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	windowsShellDLL      = syscall.MustLoadDLL("Shell32.dll")
	windowsGetFolderPath = windowsShellDLL.MustFindProc("SHGetFolderPathW")
)

const CSIDL_APPDATA = 26

func legacyConfigFile() (string, error) {
	dir, err := windowsAppDataDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "terraform.rc"), nil
}

func legacyConfigDir() (string, error) {
	dir, err := windowsAppDataDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, "terraform.d"), nil
}

func windowsAppDataDir() (string, error) {
	b := make([]uint16, syscall.MAX_PATH)

	// See: http://msdn.microsoft.com/en-us/library/windows/desktop/bb762181(v=vs.85).aspx
	r, _, err := windowsGetFolderPath.Call(0, CSIDL_APPDATA, 0, 0, uintptr(unsafe.Pointer(&b[0])))
	if uint32(r) != 0 {
		return "", err
	}

	return syscall.UTF16ToString(b), nil
}
