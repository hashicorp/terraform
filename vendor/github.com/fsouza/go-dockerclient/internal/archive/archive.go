// Copyright 2014 Docker authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the DOCKER-LICENSE file.

package archive

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/pkg/fileutils"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/pools"
	"github.com/docker/docker/pkg/system"
	"github.com/sirupsen/logrus"
)

const (
	// Uncompressed represents the uncompressed.
	Uncompressed Compression = iota
	// Bzip2 is bzip2 compression algorithm.
	Bzip2
	// Gzip is gzip compression algorithm.
	Gzip
	// Xz is xz compression algorithm.
	Xz
)

const (
	modeISDIR  = 040000  // Directory
	modeISFIFO = 010000  // FIFO
	modeISREG  = 0100000 // Regular file
	modeISLNK  = 0120000 // Symbolic link
	modeISBLK  = 060000  // Block special file
	modeISCHR  = 020000  // Character special file
	modeISSOCK = 0140000 // Socket
)

// Compression is the state represents if compressed or not.
type Compression int

// Extension returns the extension of a file that uses the specified compression algorithm.
func (compression *Compression) Extension() string {
	switch *compression {
	case Uncompressed:
		return "tar"
	case Bzip2:
		return "tar.bz2"
	case Gzip:
		return "tar.gz"
	case Xz:
		return "tar.xz"
	}
	return ""
}

// WhiteoutFormat is the format of whiteouts unpacked
type WhiteoutFormat int

// TarOptions wraps the tar options.
type TarOptions struct {
	IncludeFiles     []string
	ExcludePatterns  []string
	Compression      Compression
	NoLchown         bool
	UIDMaps          []idtools.IDMap
	GIDMaps          []idtools.IDMap
	ChownOpts        *idtools.IDPair
	IncludeSourceDir bool
	// WhiteoutFormat is the expected on disk format for whiteout files.
	// This format will be converted to the standard format on pack
	// and from the standard format on unpack.
	WhiteoutFormat WhiteoutFormat
	// When unpacking, specifies whether overwriting a directory with a
	// non-directory is allowed and vice versa.
	NoOverwriteDirNonDir bool
	// For each include when creating an archive, the included name will be
	// replaced with the matching name from this map.
	RebaseNames map[string]string
	InUserNS    bool
}

// TarWithOptions creates an archive from the directory at `path`, only including files whose relative
// paths are included in `options.IncludeFiles` (if non-nil) or not in `options.ExcludePatterns`.
func TarWithOptions(srcPath string, options *TarOptions) (io.ReadCloser, error) {

	// Fix the source path to work with long path names. This is a no-op
	// on platforms other than Windows.
	srcPath = fixVolumePathPrefix(srcPath)

	pm, err := fileutils.NewPatternMatcher(options.ExcludePatterns)
	if err != nil {
		return nil, err
	}

	pipeReader, pipeWriter := io.Pipe()

	compressWriter, err := CompressStream(pipeWriter, options.Compression)
	if err != nil {
		return nil, err
	}

	go func() {
		ta := newTarAppender(
			idtools.NewIDMappingsFromMaps(options.UIDMaps, options.GIDMaps),
			compressWriter,
			options.ChownOpts,
		)
		ta.WhiteoutConverter = getWhiteoutConverter(options.WhiteoutFormat)

		defer func() {
			// Make sure to check the error on Close.
			if err := ta.TarWriter.Close(); err != nil {
				logrus.Errorf("Can't close tar writer: %s", err)
			}
			if err := compressWriter.Close(); err != nil {
				logrus.Errorf("Can't close compress writer: %s", err)
			}
			if err := pipeWriter.Close(); err != nil {
				logrus.Errorf("Can't close pipe writer: %s", err)
			}
		}()

		// this buffer is needed for the duration of this piped stream
		defer pools.BufioWriter32KPool.Put(ta.Buffer)

		// In general we log errors here but ignore them because
		// during e.g. a diff operation the container can continue
		// mutating the filesystem and we can see transient errors
		// from this

		stat, err := os.Lstat(srcPath)
		if err != nil {
			return
		}

		if !stat.IsDir() {
			// We can't later join a non-dir with any includes because the
			// 'walk' will error if "file/." is stat-ed and "file" is not a
			// directory. So, we must split the source path and use the
			// basename as the include.
			if len(options.IncludeFiles) > 0 {
				logrus.Warn("Tar: Can't archive a file with includes")
			}

			dir, base := SplitPathDirEntry(srcPath)
			srcPath = dir
			options.IncludeFiles = []string{base}
		}

		if len(options.IncludeFiles) == 0 {
			options.IncludeFiles = []string{"."}
		}

		seen := make(map[string]bool)

		for _, include := range options.IncludeFiles {
			rebaseName := options.RebaseNames[include]

			walkRoot := getWalkRoot(srcPath, include)
			filepath.Walk(walkRoot, func(filePath string, f os.FileInfo, err error) error {
				if err != nil {
					logrus.Errorf("Tar: Can't stat file %s to tar: %s", srcPath, err)
					return nil
				}

				relFilePath, err := filepath.Rel(srcPath, filePath)
				if err != nil || (!options.IncludeSourceDir && relFilePath == "." && f.IsDir()) {
					// Error getting relative path OR we are looking
					// at the source directory path. Skip in both situations.
					return nil
				}

				if options.IncludeSourceDir && include == "." && relFilePath != "." {
					relFilePath = strings.Join([]string{".", relFilePath}, string(filepath.Separator))
				}

				skip := false

				// If "include" is an exact match for the current file
				// then even if there's an "excludePatterns" pattern that
				// matches it, don't skip it. IOW, assume an explicit 'include'
				// is asking for that file no matter what - which is true
				// for some files, like .dockerignore and Dockerfile (sometimes)
				if include != relFilePath {
					skip, err = pm.Matches(relFilePath)
					if err != nil {
						logrus.Errorf("Error matching %s: %v", relFilePath, err)
						return err
					}
				}

				if skip {
					// If we want to skip this file and its a directory
					// then we should first check to see if there's an
					// excludes pattern (e.g. !dir/file) that starts with this
					// dir. If so then we can't skip this dir.

					// Its not a dir then so we can just return/skip.
					if !f.IsDir() {
						return nil
					}

					// No exceptions (!...) in patterns so just skip dir
					if !pm.Exclusions() {
						return filepath.SkipDir
					}

					dirSlash := relFilePath + string(filepath.Separator)

					for _, pat := range pm.Patterns() {
						if !pat.Exclusion() {
							continue
						}
						if strings.HasPrefix(pat.String()+string(filepath.Separator), dirSlash) {
							// found a match - so can't skip this dir
							return nil
						}
					}

					// No matching exclusion dir so just skip dir
					return filepath.SkipDir
				}

				if seen[relFilePath] {
					return nil
				}
				seen[relFilePath] = true

				// Rename the base resource.
				if rebaseName != "" {
					var replacement string
					if rebaseName != string(filepath.Separator) {
						// Special case the root directory to replace with an
						// empty string instead so that we don't end up with
						// double slashes in the paths.
						replacement = rebaseName
					}

					relFilePath = strings.Replace(relFilePath, include, replacement, 1)
				}

				if err := ta.addTarFile(filePath, relFilePath); err != nil {
					logrus.Errorf("Can't add file %s to tar: %s", filePath, err)
					// if pipe is broken, stop writing tar stream to it
					if err == io.ErrClosedPipe {
						return err
					}
				}
				return nil
			})
		}
	}()

	return pipeReader, nil
}

// CompressStream compresses the dest with specified compression algorithm.
func CompressStream(dest io.Writer, compression Compression) (io.WriteCloser, error) {
	p := pools.BufioWriter32KPool
	buf := p.Get(dest)
	switch compression {
	case Uncompressed:
		writeBufWrapper := p.NewWriteCloserWrapper(buf, buf)
		return writeBufWrapper, nil
	case Gzip:
		gzWriter := gzip.NewWriter(dest)
		writeBufWrapper := p.NewWriteCloserWrapper(buf, gzWriter)
		return writeBufWrapper, nil
	case Bzip2, Xz:
		// archive/bzip2 does not support writing, and there is no xz support at all
		// However, this is not a problem as docker only currently generates gzipped tars
		return nil, fmt.Errorf("Unsupported compression format %s", (&compression).Extension())
	default:
		return nil, fmt.Errorf("Unsupported compression format %s", (&compression).Extension())
	}
}

type tarWhiteoutConverter interface {
	ConvertWrite(*tar.Header, string, os.FileInfo) (*tar.Header, error)
	ConvertRead(*tar.Header, string) (bool, error)
}

type tarAppender struct {
	TarWriter *tar.Writer
	Buffer    *bufio.Writer

	// for hardlink mapping
	SeenFiles  map[uint64]string
	IDMappings *idtools.IDMappings
	ChownOpts  *idtools.IDPair

	// For packing and unpacking whiteout files in the
	// non standard format. The whiteout files defined
	// by the AUFS standard are used as the tar whiteout
	// standard.
	WhiteoutConverter tarWhiteoutConverter
}

func newTarAppender(idMapping *idtools.IDMappings, writer io.Writer, chownOpts *idtools.IDPair) *tarAppender {
	return &tarAppender{
		SeenFiles:  make(map[uint64]string),
		TarWriter:  tar.NewWriter(writer),
		Buffer:     pools.BufioWriter32KPool.Get(nil),
		IDMappings: idMapping,
		ChownOpts:  chownOpts,
	}
}

// addTarFile adds to the tar archive a file from `path` as `name`
func (ta *tarAppender) addTarFile(path, name string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}

	var link string
	if fi.Mode()&os.ModeSymlink != 0 {
		var err error
		link, err = os.Readlink(path)
		if err != nil {
			return err
		}
	}

	hdr, err := FileInfoHeader(name, fi, link)
	if err != nil {
		return err
	}
	if err := ReadSecurityXattrToTarHeader(path, hdr); err != nil {
		return err
	}

	// if it's not a directory and has more than 1 link,
	// it's hard linked, so set the type flag accordingly
	if !fi.IsDir() && hasHardlinks(fi) {
		inode, err := getInodeFromStat(fi.Sys())
		if err != nil {
			return err
		}
		// a link should have a name that it links too
		// and that linked name should be first in the tar archive
		if oldpath, ok := ta.SeenFiles[inode]; ok {
			hdr.Typeflag = tar.TypeLink
			hdr.Linkname = oldpath
			hdr.Size = 0 // This Must be here for the writer math to add up!
		} else {
			ta.SeenFiles[inode] = name
		}
	}

	//check whether the file is overlayfs whiteout
	//if yes, skip re-mapping container ID mappings.
	isOverlayWhiteout := fi.Mode()&os.ModeCharDevice != 0 && hdr.Devmajor == 0 && hdr.Devminor == 0

	//handle re-mapping container ID mappings back to host ID mappings before
	//writing tar headers/files. We skip whiteout files because they were written
	//by the kernel and already have proper ownership relative to the host
	if !isOverlayWhiteout &&
		!strings.HasPrefix(filepath.Base(hdr.Name), WhiteoutPrefix) &&
		!ta.IDMappings.Empty() {
		fileIDPair, err := getFileUIDGID(fi.Sys())
		if err != nil {
			return err
		}
		hdr.Uid, hdr.Gid, err = ta.IDMappings.ToContainer(fileIDPair)
		if err != nil {
			return err
		}
	}

	// explicitly override with ChownOpts
	if ta.ChownOpts != nil {
		hdr.Uid = ta.ChownOpts.UID
		hdr.Gid = ta.ChownOpts.GID
	}

	if ta.WhiteoutConverter != nil {
		wo, err := ta.WhiteoutConverter.ConvertWrite(hdr, path, fi)
		if err != nil {
			return err
		}

		// If a new whiteout file exists, write original hdr, then
		// replace hdr with wo to be written after. Whiteouts should
		// always be written after the original. Note the original
		// hdr may have been updated to be a whiteout with returning
		// a whiteout header
		if wo != nil {
			if err := ta.TarWriter.WriteHeader(hdr); err != nil {
				return err
			}
			if hdr.Typeflag == tar.TypeReg && hdr.Size > 0 {
				return fmt.Errorf("tar: cannot use whiteout for non-empty file")
			}
			hdr = wo
		}
	}

	if err := ta.TarWriter.WriteHeader(hdr); err != nil {
		return err
	}

	if hdr.Typeflag == tar.TypeReg && hdr.Size > 0 {
		// We use system.OpenSequential to ensure we use sequential file
		// access on Windows to avoid depleting the standby list.
		// On Linux, this equates to a regular os.Open.
		file, err := system.OpenSequential(path)
		if err != nil {
			return err
		}

		ta.Buffer.Reset(ta.TarWriter)
		defer ta.Buffer.Reset(nil)
		_, err = io.Copy(ta.Buffer, file)
		file.Close()
		if err != nil {
			return err
		}
		err = ta.Buffer.Flush()
		if err != nil {
			return err
		}
	}

	return nil
}

// ReadSecurityXattrToTarHeader reads security.capability xattr from filesystem
// to a tar header
func ReadSecurityXattrToTarHeader(path string, hdr *tar.Header) error {
	capability, _ := system.Lgetxattr(path, "security.capability")
	if capability != nil {
		hdr.Xattrs = make(map[string]string)
		hdr.Xattrs["security.capability"] = string(capability)
	}
	return nil
}

// FileInfoHeader creates a populated Header from fi.
// Compared to archive pkg this function fills in more information.
// Also, regardless of Go version, this function fills file type bits (e.g. hdr.Mode |= modeISDIR),
// which have been deleted since Go 1.9 archive/tar.
func FileInfoHeader(name string, fi os.FileInfo, link string) (*tar.Header, error) {
	hdr, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return nil, err
	}
	hdr.Mode = fillGo18FileTypeBits(int64(chmodTarEntry(os.FileMode(hdr.Mode))), fi)
	name, err = canonicalTarName(name, fi.IsDir())
	if err != nil {
		return nil, fmt.Errorf("tar: cannot canonicalize path: %v", err)
	}
	hdr.Name = name
	if err := setHeaderForSpecialDevice(hdr, name, fi.Sys()); err != nil {
		return nil, err
	}
	return hdr, nil
}

// fillGo18FileTypeBits fills type bits which have been removed on Go 1.9 archive/tar
// https://github.com/golang/go/commit/66b5a2f
func fillGo18FileTypeBits(mode int64, fi os.FileInfo) int64 {
	fm := fi.Mode()
	switch {
	case fm.IsRegular():
		mode |= modeISREG
	case fi.IsDir():
		mode |= modeISDIR
	case fm&os.ModeSymlink != 0:
		mode |= modeISLNK
	case fm&os.ModeDevice != 0:
		if fm&os.ModeCharDevice != 0 {
			mode |= modeISCHR
		} else {
			mode |= modeISBLK
		}
	case fm&os.ModeNamedPipe != 0:
		mode |= modeISFIFO
	case fm&os.ModeSocket != 0:
		mode |= modeISSOCK
	}
	return mode
}

// canonicalTarName provides a platform-independent and consistent posix-style
//path for files and directories to be archived regardless of the platform.
func canonicalTarName(name string, isDir bool) (string, error) {
	name, err := CanonicalTarNameForPath(name)
	if err != nil {
		return "", err
	}

	// suffix with '/' for directories
	if isDir && !strings.HasSuffix(name, "/") {
		name += "/"
	}
	return name, nil
}
