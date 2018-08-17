// Copyright 2014 Docker authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the DOCKER-LICENSE file.

package term

import "github.com/Azure/go-ansiterm/winterm"

// GetWinsize returns the window size based on the specified file descriptor.
func GetWinsize(fd uintptr) (*Winsize, error) {
	info, err := winterm.GetConsoleScreenBufferInfo(fd)
	if err != nil {
		return nil, err
	}

	winsize := &Winsize{
		Width:  uint16(info.Window.Right - info.Window.Left + 1),
		Height: uint16(info.Window.Bottom - info.Window.Top + 1),
	}

	return winsize, nil
}
