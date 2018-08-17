// Copyright 2014 Docker authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the DOCKER-LICENSE file.

// +build !windows

package archive

import (
	"os"
	"syscall"
)

func hasHardlinks(fi os.FileInfo) bool {
	return fi.Sys().(*syscall.Stat_t).Nlink > 1
}
