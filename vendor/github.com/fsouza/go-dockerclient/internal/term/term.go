// Copyright 2014 Docker authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the DOCKER-LICENSE file.

package term

// Winsize represents the size of the terminal window.
type Winsize struct {
	Height uint16
	Width  uint16
	x      uint16
	y      uint16
}
