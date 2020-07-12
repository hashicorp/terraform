// +build js

package configload

import (
	"errors"
)

func inode(path string) (uint64, error) {
	return 0, errors.New("cannot look up file inode on JavaScript platform")
}
