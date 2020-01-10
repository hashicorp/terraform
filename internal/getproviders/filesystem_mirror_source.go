package getproviders

import (
	"github.com/hashicorp/terraform/addrs"
)

// FilesystemMirrorSource is a source that reads providers and their metadata
// from a directory prefix in the local filesystem.
type FilesystemMirrorSource struct {
	baseDir string
}

var _ Source = (*FilesystemMirrorSource)(nil)

// NewFilesystemMirrorSource constructs and returns a new filesystem-based
// mirror source with the given base directory.
func NewFilesystemMirrorSource(baseDir string) *FilesystemMirrorSource {
	return &FilesystemMirrorSource{
		baseDir: baseDir,
	}
}

// AvailableVersions scans the directory structure under the source's base
// directory for locally-mirrored packages for the given provider, returning
// a list of version numbers for the providers it found.
func (s *FilesystemMirrorSource) AvailableVersions(provider addrs.Provider) (VersionList, error) {
	// TODO: Implement
	panic("FilesystemMirrorSource.AvailableVersions not yet implemented")
}

// PackageMeta checks to see if the source's base directory contains a
// local copy of the distribution package for the given provider version on
// the given target, and returns the metadata about it if so.
func (s *FilesystemMirrorSource) PackageMeta(provider addrs.Provider, version Version, target Platform) (PackageMeta, error) {
	// TODO: Implement
	panic("FilesystemMirrorSource.PackageMeta not yet implemented")
}
