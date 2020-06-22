package getproviders

import (
	"github.com/hashicorp/terraform/addrs"
)

// FilesystemMirrorSource is a source that reads providers and their metadata
// from a directory prefix in the local filesystem.
type FilesystemMirrorSource struct {
	baseDir string

	// allPackages caches the result of scanning the baseDir for all available
	// packages on the first call that needs package availability information,
	// to avoid re-scanning the filesystem on subsequent operations.
	allPackages map[addrs.Provider]PackageMetaList
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
func (s *FilesystemMirrorSource) AvailableVersions(provider addrs.Provider) (VersionList, Warnings, error) {
	// s.allPackages is populated if scanAllVersions succeeds
	err := s.scanAllVersions()
	if err != nil {
		return nil, nil, err
	}

	// There might be multiple packages for a given version in the filesystem,
	// but the contract here is to return distinct versions so we'll dedupe
	// them first, then sort them, and then return them.
	versionsMap := make(map[Version]struct{})
	for _, m := range s.allPackages[provider] {
		versionsMap[m.Version] = struct{}{}
	}
	ret := make(VersionList, 0, len(versionsMap))
	for v := range versionsMap {
		ret = append(ret, v)
	}
	ret.Sort()
	return ret, nil, nil
}

// PackageMeta checks to see if the source's base directory contains a
// local copy of the distribution package for the given provider version on
// the given target, and returns the metadata about it if so.
func (s *FilesystemMirrorSource) PackageMeta(provider addrs.Provider, version Version, target Platform) (PackageMeta, error) {
	// s.allPackages is populated if scanAllVersions succeeds
	err := s.scanAllVersions()
	if err != nil {
		return PackageMeta{}, err
	}

	relevantPkgs := s.allPackages[provider].FilterProviderPlatformExactVersion(provider, target, version)
	if len(relevantPkgs) == 0 {
		// This is the local equivalent of a "404 Not Found" when retrieving
		// a particular version from a registry or network mirror. Because
		// the caller should've selected a version already found by
		// AvailableVersions, the only discriminator that should fail here
		// is the target platform, and so our error result assumes that,
		// causing the caller to return an error like "This provider version is
		// not compatible with aros_riscv".
		return PackageMeta{}, ErrPlatformNotSupported{
			Provider: provider,
			Version:  version,
			Platform: target,
		}
	}

	// It's possible that there could be multiple copies of the same package
	// available in the filesystem, if e.g. there's both a packed and an
	// unpacked variant. For now we assume that the decision between them
	// is arbitrary and just take the first one in the result.
	return relevantPkgs[0], nil
}

// AllAvailablePackages scans the directory structure under the source's base
// directory for locally-mirrored packages for all providers, returning a map
// of the discovered packages with the fully-qualified provider names as
// keys.
//
// This is not an operation generally supported by all Source implementations,
// but the filesystem implementation offers it because we also use the
// filesystem mirror source directly to scan our auto-install plugin directory
// and in other automatic discovery situations.
func (s *FilesystemMirrorSource) AllAvailablePackages() (map[addrs.Provider]PackageMetaList, error) {
	// s.allPackages is populated if scanAllVersions succeeds
	err := s.scanAllVersions()
	return s.allPackages, err
}

func (s *FilesystemMirrorSource) scanAllVersions() error {
	if s.allPackages != nil {
		// we're distinguishing nil-ness from emptiness here so we can
		// recognize when we've scanned the directory without errors, even
		// if we found nothing during the scan.
		return nil
	}

	ret, err := SearchLocalDirectory(s.baseDir)
	if err != nil {
		return err
	}

	// As noted above, we use an explicit empty map so we can distinguish a
	// successful-but-empty result from a failure on future calls, so we'll
	// make sure that's what we have before we assign it here.
	if ret == nil {
		ret = make(map[addrs.Provider]PackageMetaList)
	}
	s.allPackages = ret
	return nil
}

func (s *FilesystemMirrorSource) ForDisplay(provider addrs.Provider) string {
	return s.baseDir
}
