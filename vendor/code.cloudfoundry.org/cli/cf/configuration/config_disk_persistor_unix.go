// +build !windows

package configuration

import (
	"os"
	"path/filepath"
)

func (dp DiskPersistor) makeDirectory() error {
	return os.MkdirAll(filepath.Dir(dp.filePath), dirPermissions)
}
