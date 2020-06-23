package getproviders

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	svchost "github.com/hashicorp/terraform-svchost"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

// SearchLocalDirectory performs an immediate, one-off scan of the given base
// directory for provider plugins using the directory structure defined for
// FilesystemMirrorSource.
//
// This is separated to allow other callers, such as the provider plugin cache
// management in the "internal/providercache" package, to use the same
// directory structure conventions.
func SearchLocalDirectory(baseDir string) (map[addrs.Provider]PackageMetaList, error) {
	available, diags := SearchLocalDirectoryDiags(baseDir)

	// Filter out error diagnostics to build a return error, and log warning diagnostics here
	var errDiags tfdiags.Diagnostics
	for _, diag := range diags {
		if diag.Severity() == tfdiags.Error {
			errDiags = errDiags.Append(diag)
		} else if diag.Severity() == tfdiags.Warning {
			desc := diag.Description()
			err := desc.Summary
			if desc.Detail == "" {
				err = fmt.Sprintf("%s: %s", desc.Summary, desc.Detail)
			}
			log.Printf("[WARN] SearchLocalDirectory: %s", err)
		}
	}

	return available, errDiags.Err()
}

// SearchLocalDirectoryDiags implements the logic for SearchLocalDirectory, but
// returning detailed diagnostic information instead of logging warnings and
// returning only errors.
func SearchLocalDirectoryDiags(baseDir string) (map[addrs.Provider]PackageMetaList, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := make(map[addrs.Provider]PackageMetaList)
	filepath.Walk(baseDir, func(fullPath string, info os.FileInfo, err error) error {
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Cannot search %s", fullPath),
				err.Error(),
			))
			return diags.Err()
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
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				fmt.Sprintf("Ignoring malformed path %q", fullPath),
				err.Error(),
			))
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

		// validate each part
		// The legacy provider namespace is a special case.
		if namespace != addrs.LegacyProviderNamespace {
			_, err = addrs.ParseProviderPart(namespace)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					fmt.Sprintf("Ignoring local provider at %s", relPath),
					fmt.Sprintf("Provider contains invalid namespace %s: %s", namespace, err),
				))
				return nil
			}
		}

		_, err = addrs.ParseProviderPart(typeName)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				fmt.Sprintf("Ignoring local provider at %s", relPath),
				fmt.Sprintf("Provider contains invalid type %s: %s", typeName, err),
			))
			return nil
		}

		hostname, err := svchost.ForComparison(hostnameGiven)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				fmt.Sprintf("Ignoring local provider at %s", relPath),
				fmt.Sprintf("Provider contains invalid hostname %s: %s", hostnameGiven, err),
			))
			return nil
		}
		var providerAddr addrs.Provider
		if namespace == addrs.LegacyProviderNamespace {
			if hostname != addrs.DefaultRegistryHost {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					fmt.Sprintf("Ignoring local provider at %s", relPath),
					fmt.Sprintf("Provider indicates a legacy provider not on the default registry host: %s", err),
				))
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
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Failed to read metadata about %s", fullPath),
				err.Error(),
			))
			return diags.Err()
		}

		switch len(parts) {
		case 5: // Might be unpacked layout
			if !info.IsDir() {
				return nil // packed layout requires a directory
			}

			versionStr := parts[3]
			version, err := ParseVersion(versionStr)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					fmt.Sprintf("Ignoring local provider %s", providerAddr),
					fmt.Sprintf("Provider has invalid version %q: %s. Path: %q", versionStr, err, fullPath),
				))
				return nil
			}

			platformStr := parts[4]
			platform, err := ParsePlatform(platformStr)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					fmt.Sprintf("Ignoring local provider %s", providerAddr),
					fmt.Sprintf("Provider has invalid platform %q: %s. Path: %q", platformStr, err, fullPath),
				))
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
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					fmt.Sprintf("Ignoring file %q as possible package for %s", filename, providerAddr),
					fmt.Sprintf("Filename %s lacks expected prefix %q", fsPath, prefix),
				))
				return nil
			}
			if !strings.HasSuffix(normFilename, suffix) {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					fmt.Sprintf("Ignoring file %q as possible package for %s", filename, providerAddr),
					fmt.Sprintf("Filename %s lacks expected suffix %q", fsPath, suffix),
				))
				return nil
			}

			// Extract the version and target part of the filename, which
			// will look like "2.1.0_linux_amd64"
			infoSlice := normFilename[len(prefix) : len(normFilename)-len(suffix)]
			infoParts := strings.Split(infoSlice, "_")
			if len(infoParts) < 3 {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					fmt.Sprintf("Ignoring file %q as possible package for %s", filename, providerAddr),
					fmt.Sprintf("Filename %s does not include version number, target OS, and target architecture", fsPath),
				))
				return nil
			}

			versionStr := infoParts[0]
			version, err := ParseVersion(versionStr)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					fmt.Sprintf("Ignoring local provider %s", providerAddr),
					fmt.Sprintf("Provider has invalid version %q: %s", versionStr, err),
				))
				return nil
			}

			// We'll reassemble this back into a single string just so we can
			// easily re-use our existing parser and its normalization rules.
			platformStr := infoParts[1] + "_" + infoParts[2]
			platform, err := ParsePlatform(platformStr)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					fmt.Sprintf("Ignoring local provider %s", providerAddr),
					fmt.Sprintf("Provider has invalid platform %q: %s", platformStr, err),
				))
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
	if diags.HasErrors() {
		return nil, diags
	}
	// Sort the results to be deterministic (aside from semver build metadata)
	// and consistent with ordering from other functions.
	for _, l := range ret {
		l.Sort()
	}
	return ret, diags
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
