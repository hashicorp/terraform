// Copyright 2014 Docker authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the DOCKER-LICENSE file.

package archive

import (
	"os"
	"path/filepath"
)

// SplitPathDirEntry splits the given path between its directory name and its
// basename by first cleaning the path but preserves a trailing "." if the
// original path specified the current directory.
func SplitPathDirEntry(path string) (dir, base string) {
	cleanedPath := filepath.Clean(filepath.FromSlash(path))

	if specifiesCurrentDir(path) {
		cleanedPath += string(os.PathSeparator) + "."
	}

	return filepath.Dir(cleanedPath), filepath.Base(cleanedPath)
}

// specifiesCurrentDir returns whether the given path specifies
// a "current directory", i.e., the last path segment is `.`.
func specifiesCurrentDir(path string) bool {
	return filepath.Base(path) == "."
}
