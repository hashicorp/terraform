// Copyright 2014 Docker authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the DOCKER-LICENSE file.

package archive

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/longpath"
)

// CanonicalTarNameForPath returns platform-specific filepath
// to canonical posix-style path for tar archival. p is relative
// path.
func CanonicalTarNameForPath(p string) (string, error) {
	// windows: convert windows style relative path with backslashes
	// into forward slashes. Since windows does not allow '/' or '\'
	// in file names, it is mostly safe to replace however we must
	// check just in case
	if strings.Contains(p, "/") {
		return "", fmt.Errorf("Windows path contains forward slash: %s", p)
	}
	return strings.Replace(p, string(os.PathSeparator), "/", -1), nil

}

// fixVolumePathPrefix does platform specific processing to ensure that if
// the path being passed in is not in a volume path format, convert it to one.
func fixVolumePathPrefix(srcPath string) string {
	return longpath.AddPrefix(srcPath)
}

// getWalkRoot calculates the root path when performing a TarWithOptions.
// We use a separate function as this is platform specific.
func getWalkRoot(srcPath string, include string) string {
	return filepath.Join(srcPath, include)
}

func getInodeFromStat(stat interface{}) (inode uint64, err error) {
	// do nothing. no notion of Inode in stat on Windows
	return
}

func getFileUIDGID(stat interface{}) (idtools.IDPair, error) {
	// no notion of file ownership mapping yet on Windows
	return idtools.IDPair{0, 0}, nil
}

// chmodTarEntry is used to adjust the file permissions used in tar header based
// on the platform the archival is done.
func chmodTarEntry(perm os.FileMode) os.FileMode {
	//perm &= 0755 // this 0-ed out tar flags (like link, regular file, directory marker etc.)
	permPart := perm & os.ModePerm
	noPermPart := perm &^ os.ModePerm
	// Add the x bit: make everything +x from windows
	permPart |= 0111
	permPart &= 0755

	return noPermPart | permPart
}

func setHeaderForSpecialDevice(hdr *tar.Header, name string, stat interface{}) (err error) {
	// do nothing. no notion of Rdev, Nlink in stat on Windows
	return
}
