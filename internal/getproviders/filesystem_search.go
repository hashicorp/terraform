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

// SearchLocalDirectory performs an immediate, one-off scan of the given base
// directory for provider plugins using the directory structure defined for
// FilesystemMirrorSource.
//
// This is separated to allow other callers, such as the provider plugin cache
// management in the "internal/providercache" package, to use the same
// directory structure conventions.
func SearchLocalDirectory(baseDir string) (map[addrs.Provider]PackageMetaList, error) {
	ret := make(map[addrs.Provider]PackageMetaList)

	// We don't support symlinks at intermediate points inside the directory
	// heirarchy because that could potentially cause our walk to get into
	// an infinite loop, but as a measure of pragmatism we'll allow the
	// top-level location itself to be a symlink, so that a user can
	// potentially keep their plugins in a non-standard location but use a
	// symlink to help Terraform find them anyway.
	originalBaseDir := baseDir
	if finalDir, err := filepath.EvalSymlinks(baseDir); err == nil {
		log.Printf("[TRACE] getproviders.SearchLocalDirectory: %s is a symlink to %s", baseDir, finalDir)
		baseDir = finalDir
	} else {
		// We'll eat this particular error because if we're somehow able to
		// find plugins via baseDir below anyway then we'd rather do that than
		// hard fail, but we'll log it in case it's useful for diagnosing why
		// discovery didn't produce the expected outcome.
		log.Printf("[TRACE] getproviders.SearchLocalDirectory: failed to resolve symlinks for %s: %s", baseDir, err)
	}

	err := filepath.Walk(baseDir, func(fullPath string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("cannot search %s: %s", fullPath, err)
		}

		// There are two valid directory structures that we support here...
		// Unpacked: registry.terraform.io/hashicorp/aws/2.0.0/linux_amd64 (a directory)
		// Packed:   registry.terraform.io/hashicorp/aws/terraform-provider-aws_2.0.0_linux_amd64.zip (a file)
		//
		// Both of these give us enough information to identify the package
		// metadata.
		fsPath, err := filepath.Rel(baseDir, fullPath)
		if err != nil {
			// This should never happen because the filepath.Walk contract is
			// for the paths to include the base path.
			log.Printf("[TRACE] getproviders.SearchLocalDirectory: ignoring malformed path %q during walk: %s", fullPath, err)
			return nil
		}
		relPath := filepath.ToSlash(fsPath)
		parts := strings.Split(relPath, "/")

		if len(parts) < 3 {
			// Likely a prefix of a valid path, so we'll ignore it and visit
			// the full valid path on a later call.

			if (info.Mode() & os.ModeSymlink) != 0 {
				// We don't allow symlinks for intermediate steps in the
				// heirarchy because otherwise this walk would risk getting
				// itself into an infinite loop, but if we do find one then
				// we'll warn about it to help with debugging.
				log.Printf("[WARN] Provider plugin search ignored symlink %s: only the base directory %s may be a symlink", fullPath, originalBaseDir)
			}

			return nil
		}

		hostnameGiven := parts[0]
		namespace := parts[1]
		typeName := parts[2]

		// validate each part
		// The legacy provider namespace is a special case.
		if namespace != addrs.LegacyProviderNamespace {
			_, err = addrs.ParseProviderPart(namespace)
			if err != nil {
				log.Printf("[WARN] local provider path %q contains invalid namespace %q; ignoring", fullPath, namespace)
				return nil
			}
		}

		_, err = addrs.ParseProviderPart(typeName)
		if err != nil {
			log.Printf("[WARN] local provider path %q contains invalid type %q; ignoring", fullPath, typeName)
			return nil
		}

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

		// The "info" passed to our function is an Lstat result, so it might
		// be referring to a symbolic link. We'll do a full "Stat" on it
		// now to make sure we're making tests against the real underlying
		// filesystem object below.
		info, err = os.Stat(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read metadata about %s: %s", fullPath, err)
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

			log.Printf("[TRACE] getproviders.SearchLocalDirectory: found %s v%s for %s at %s", providerAddr, version, platform, fullPath)

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
				log.Printf("[WARN] ignoring file %q as possible package for %s: filename lacks expected prefix %q", fsPath, providerAddr, prefix)
				return nil
			}
			if !strings.HasSuffix(normFilename, suffix) {
				log.Printf("[WARN] ignoring file %q as possible package for %s: filename lacks expected suffix %q", fsPath, providerAddr, suffix)
				return nil
			}

			// Extract the version and target part of the filename, which
			// will look like "2.1.0_linux_amd64"
			infoSlice := normFilename[len(prefix) : len(normFilename)-len(suffix)]
			infoParts := strings.Split(infoSlice, "_")
			if len(infoParts) < 3 {
				log.Printf("[WARN] ignoring file %q as possible package for %s: filename does not include version number, target OS, and target architecture", fsPath, providerAddr)
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

			log.Printf("[TRACE] getproviders.SearchLocalDirectory: found %s v%s for %s at %s", providerAddr, version, platform, fullPath)

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
		return nil, err
	}
	// Sort the results to be deterministic (aside from semver build metadata)
	// and consistent with ordering from other functions.
	for _, l := range ret {
		l.Sort()
	}
	return ret, nil
}

// UnpackedDirectoryPathForPackage is similar to
// PackageMeta.UnpackedDirectoryPath but makes its decision based on
// individually-passed provider address, version, and target platform so that
// it can be used by callers outside this package that may have other
// types that represent package identifiers.
func UnpackedDirectoryPathForPackage(baseDir string, provider addrs.Provider, version Version, platform Platform) string {
	return filepath.ToSlash(filepath.Join(
		baseDir,
		provider.Hostname.ForDisplay(), provider.Namespace, provider.Type,
		version.String(),
		platform.String(),
	))
}

// PackedFilePathForPackage is similar to
// PackageMeta.PackedFilePath but makes its decision based on
// individually-passed provider address, version, and target platform so that
// it can be used by callers outside this package that may have other
// types that represent package identifiers.
func PackedFilePathForPackage(baseDir string, provider addrs.Provider, version Version, platform Platform) string {
	return filepath.ToSlash(filepath.Join(
		baseDir,
		provider.Hostname.ForDisplay(), provider.Namespace, provider.Type,
		fmt.Sprintf("terraform-provider-%s_%s_%s.zip", provider.Type, version.String(), platform.String()),
	))
}
