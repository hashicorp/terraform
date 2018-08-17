// Copyright 2014 Docker authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the DOCKER-LICENSE file.

// +build !linux

package archive

func getWhiteoutConverter(format WhiteoutFormat) tarWhiteoutConverter {
	return nil
}
