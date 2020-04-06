// +build !windows

package windowsbase

import (
	"errors"
)

func knownFolderDir(id *FolderID) (string, error) {
	return "", errors.New("cannot use Windows known folders on a non-Windows platform")
}
