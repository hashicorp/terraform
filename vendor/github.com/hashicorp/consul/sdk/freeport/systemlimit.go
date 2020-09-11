// +build !windows

package freeport

import "golang.org/x/sys/unix"

func systemLimit() (int, error) {
	var limit unix.Rlimit
	err := unix.Getrlimit(unix.RLIMIT_NOFILE, &limit)
	return int(limit.Cur), err
}
