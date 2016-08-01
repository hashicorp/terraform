// +build windows

package module

// no syscall.Stat_t on windows, return 0 for inodes
func inode(path string) (uint64, error) {
	return 0, nil
}
