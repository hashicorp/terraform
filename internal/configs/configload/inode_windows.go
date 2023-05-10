// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build windows
// +build windows

package configload

// no syscall.Stat_t on windows, return 0 for inodes
func inode(path string) (uint64, error) {
	return 0, nil
}
