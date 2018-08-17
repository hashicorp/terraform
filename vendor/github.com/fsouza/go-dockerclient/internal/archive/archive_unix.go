// Copyright 2014 Docker authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the DOCKER-LICENSE file.

// +build !windows

package archive

import (
	"archive/tar"
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"github.com/docker/docker/pkg/idtools"
	"golang.org/x/sys/unix"
)

// CanonicalTarNameForPath returns platform-specific filepath
// to canonical posix-style path for tar archival. p is relative
// path.
func CanonicalTarNameForPath(p string) (string, error) {
	return p, nil // already unix-style
}

// fixVolumePathPrefix does platform specific processing to ensure that if
// the path being passed in is not in a volume path format, convert it to one.
func fixVolumePathPrefix(srcPath string) string {
	return srcPath
}

// getWalkRoot calculates the root path when performing a TarWithOptions.
// We use a separate function as this is platform specific. On Linux, we
// can't use filepath.Join(srcPath,include) because this will clean away
// a trailing "." or "/" which may be important.
func getWalkRoot(srcPath string, include string) string {
	return srcPath + string(filepath.Separator) + include
}

func getInodeFromStat(stat interface{}) (inode uint64, err error) {
	s, ok := stat.(*syscall.Stat_t)

	if ok {
		inode = uint64(s.Ino)
	}

	return
}

func getFileUIDGID(stat interface{}) (idtools.IDPair, error) {
	s, ok := stat.(*syscall.Stat_t)

	if !ok {
		return idtools.IDPair{}, errors.New("cannot convert stat value to syscall.Stat_t")
	}
	return idtools.IDPair{UID: int(s.Uid), GID: int(s.Gid)}, nil
}

func chmodTarEntry(perm os.FileMode) os.FileMode {
	return perm // noop for unix as golang APIs provide perm bits correctly
}

func setHeaderForSpecialDevice(hdr *tar.Header, name string, stat interface{}) (err error) {
	s, ok := stat.(*syscall.Stat_t)

	if ok {
		// Currently go does not fill in the major/minors
		if s.Mode&unix.S_IFBLK != 0 ||
			s.Mode&unix.S_IFCHR != 0 {
			hdr.Devmajor = int64(unix.Major(uint64(s.Rdev))) // nolint: unconvert
			hdr.Devminor = int64(unix.Minor(uint64(s.Rdev))) // nolint: unconvert
		}
	}

	return
}
