// +build linux darwin openbsd netbsd solaris

package module

import (
	"fmt"
	"os"
	"syscall"
)

// lookup the inode of a file on posix systems
func inode(path string) (uint64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	if st, ok := stat.Sys().(*syscall.Stat_t); ok {
		return st.Ino, nil
	}
	return 0, fmt.Errorf("could not determine file inode")
}
