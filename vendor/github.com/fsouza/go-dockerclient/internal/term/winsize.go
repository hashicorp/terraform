// Copyright 2014 Docker authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the DOCKER-LICENSE file.

// +build !windows

package term

import "golang.org/x/sys/unix"

// GetWinsize returns the window size based on the specified file descriptor.
func GetWinsize(fd uintptr) (*Winsize, error) {
	uws, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	ws := &Winsize{Height: uws.Row, Width: uws.Col, x: uws.Xpixel, y: uws.Ypixel}
	return ws, err
}
