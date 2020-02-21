package getproviders

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	svchost "github.com/hashicorp/terraform-svchost"
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
func (s *FilesystemMirrorSource) AvailableVersions(provider addrs.Provider) (VersionList, error) {
	// s.allPackages is populated if scanAllVersions succeeds
	err := s.scanAllVersions()
	if err != nil {
		return nil, err
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
	return ret, nil
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
	ret := make(map[addrs.Provider]PackageMetaList)
	err := filepath.Walk(s.baseDir, func(fullPath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("cannot search %s: %s", fullPath, err)
		}

		// There are two valid directory structures that we support here...
		// Unpacked: registry.terraform.io/hashicorp/aws/2.0.0/linux_amd64 (a directory)
		// Packed:   registry.terraform.io/hashicorp/aws/terraform-provider-aws_2.0.0_linux_amd64.zip (a file)
		//
		// Both of these give us enough information to identify the package
		// metadata.
		fsPath, err := filepath.Rel(s.baseDir, fullPath)
		if err != nil {
			// This should never happen because the filepath.Walk contract is
			// for the paths to include the base path.
			log.Printf("[TRACE] FilesystemMirrorSource: ignoring malformed path %q during walk: %s", fullPath, err)
			return nil
		}
		relPath := filepath.ToSlash(fsPath)
		parts := strings.Split(relPath, "/")

		if len(parts) < 3 {
			// Likely a prefix of a valid path, so we'll ignore it and visit
			// the full valid path on a later call.
			return nil
		}

		hostnameGiven := parts[0]
		namespace := parts[1]
		typeName := parts[2]

		hostname, err := svchost.ForComparison(hostnameGiven)
		if err != nil {
			log.Printf("[WARN] local provider path %q contains invalid hostname %q; ignoring", fullPath, hostnameGiven)
			return nil
		}
		var providerAddr addrs.Provider
		if namespace == addrs.LegacyProviderNamespace {
			if hostname != addrs.DefaultRegistryHost {
				log.Printf("[WARN] local provider path %q indicates a legacy provider not on the default registry host; ignoring", fullPath)
				return nil
			}
			providerAddr = addrs.NewLegacyProvider(typeName)
		} else {
			providerAddr = addrs.NewProvider(hostname, namespace, typeName)
		}

		switch len(parts) {
		case 5: // Might be unpacked layout
			if !info.IsDir() {
				return nil // packed layout requires a directory
			}

			versionStr := parts[3]
			version, err := ParseVersion(versionStr)
			if err != nil {
				log.Printf("[WARN] ignoring local provider path %q with invalid version %q: %s", fullPath, versionStr, err)
				return nil
			}

			platformStr := parts[4]
			platform, err := ParsePlatform(platformStr)
			if err != nil {
				log.Printf("[WARN] ignoring local provider path %q with invalid platform %q: %s", fullPath, platformStr, err)
				return nil
			}

			log.Printf("[TRACE] FilesystemMirrorSource: found %s v%s for %s at %s", providerAddr, version, platform, fullPath)

			meta := PackageMeta{
				Provider: providerAddr,
				Version:  version,

				// FIXME: How do we populate this?
				ProtocolVersions: nil,
				TargetPlatform:   platform,

				// Because this is already unpacked, the filename is synthetic
				// based on the standard naming scheme.
				Filename: fmt.Sprintf("terraform-provider-%s_%s_%s.zip", providerAddr.Type, version, platform),
				Location: PackageLocalDir(fullPath),

				// FIXME: What about the SHA256Sum field? As currently specified
				// it's a hash of the zip file, but this thing is already
				// unpacked and so we don't have the zip file to hash.
			}
			ret[providerAddr] = append(ret[providerAddr], meta)

		case 4: // Might be packed layout
			if info.IsDir() {
				return nil // packed layout requires a file
			}

			filename := filepath.Base(fsPath)
			// the filename components are matched case-insensitively, and
			// the normalized form of them is in lowercase so we'll convert
			// to lowercase for comparison here. (This normalizes only for case,
			// because that is the primary constraint affecting compatibility
			// between filesystem implementations on different platforms;
			// filenames are expected to be pre-normalized and valid in other
			// regards.)
			normFilename := strings.ToLower(filename)

			// In the packed layout, the version number and target platform
			// are derived from the package filename, but only if the
			// filename has the expected prefix identifying it as a package
			// for the provider in question, and the suffix identifying it
			// as a zip file.
			prefix := "terraform-provider-" + providerAddr.Type + "_"
			const suffix = ".zip"
			if !strings.HasPrefix(normFilename, prefix) {
				log.Printf("[WARN] ignoring file %q as possible package for %s: lacks expected prefix %q", filename, providerAddr, prefix)
				return nil
			}
			if !strings.HasSuffix(normFilename, suffix) {
				log.Printf("[WARN] ignoring file %q as possible package for %s: lacks expected suffix %q", filename, providerAddr, suffix)
				return nil
			}

			// Extract the version and target part of the filename, which
			// will look like "2.1.0_linux_amd64"
			infoSlice := normFilename[len(prefix) : len(normFilename)-len(suffix)]
			infoParts := strings.Split(infoSlice, "_")
			if len(infoParts) < 3 {
				log.Printf("[WARN] ignoring file %q as possible package for %s: filename does not include version number, target OS, and target architecture", filename, providerAddr)
				return nil
			}

			versionStr := infoParts[0]
			version, err := ParseVersion(versionStr)
			if err != nil {
				log.Printf("[WARN] ignoring local provider path %q with invalid version %q: %s", fullPath, versionStr, err)
				return nil
			}

			// We'll reassemble this back into a single string just so we can
			// easily re-use our existing parser and its normalization rules.
			platformStr := infoParts[1] + "_" + infoParts[2]
			platform, err := ParsePlatform(platformStr)
			if err != nil {
				log.Printf("[WARN] ignoring local provider path %q with invalid platform %q: %s", fullPath, platformStr, err)
				return nil
			}

			log.Printf("[TRACE] FilesystemMirrorSource: found %s v%s for %s at %s", providerAddr, version, platform, fullPath)

			meta := PackageMeta{
				Provider: providerAddr,
				Version:  version,

				// FIXME: How do we populate this?
				ProtocolVersions: nil,
				TargetPlatform:   platform,

				// Because this is already unpacked, the filename is synthetic
				// based on the standard naming scheme.
				Filename: normFilename,                  // normalized filename, because this field says what it _should_ be called, not what it _is_ called
				Location: PackageLocalArchive(fullPath), // non-normalized here, because this is the actual physical location

				// TODO: Also populate the SHA256Sum field. Skipping that
				// for now because our initial uses of this result --
				// scanning already-installed providers in local directories,
				// rather than explicit filesystem mirrors -- doesn't do
				// any hash verification anyway, and this is consistent with
				// the FIXME in the unpacked case above even though technically
				// we _could_ populate SHA256Sum here right now.
			}
			ret[providerAddr] = append(ret[providerAddr], meta)

		}

		return nil
	})
	if err != nil {
		return err
	}
	// Sort the results to be deterministic (aside from semver build metadata)
	// and consistent with ordering from other functions.
	for _, l := range ret {
		l.Sort()
	}
	s.allPackages = ret
	return nil
}
