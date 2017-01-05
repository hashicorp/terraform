// +build windows

package configuration

import (
	"os"
	"path/filepath"
	"syscall"
)

func (dp DiskPersistor) makeDirectory() error {
	dir := filepath.Dir(dp.filePath)

	err := os.MkdirAll(dir, dirPermissions)
	if err != nil {
		return err
	}

	p, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}

	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		return err
	}

	return syscall.SetFileAttributes(p, attrs|syscall.FILE_ATTRIBUTE_HIDDEN)
}
