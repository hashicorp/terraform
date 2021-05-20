package providercache

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	getter "github.com/hashicorp/go-getter"

	"github.com/hashicorp/terraform/internal/copy"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/httpclient"
)

// We borrow the "unpack a zip file into a target directory" logic from
// go-getter, even though we're not otherwise using go-getter here.
// (We don't need the same flexibility as we have for modules, because
// providers _always_ come from provider registries, which have a very
// specific protocol and set of expectations.)
var unzip = getter.ZipDecompressor{}

func installFromHTTPURL(ctx context.Context, meta getproviders.PackageMeta, targetDir string, allowedHashes []getproviders.Hash) (*getproviders.PackageAuthenticationResult, error) {
	url := meta.Location.String()

	// When we're installing from an HTTP URL we expect the URL to refer to
	// a zip file. We'll fetch that into a temporary file here and then
	// delegate to installFromLocalArchive below to actually extract it.
	// (We're not using go-getter here because its HTTP getter has a bunch
	// of extraneous functionality we don't need or want, like indirection
	// through X-Terraform-Get header, attempting partial fetches for
	// files that already exist, etc.)

	httpClient := httpclient.New()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid provider download request: %s", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			// "context canceled" is not a user-friendly error message,
			// so we'll return a more appropriate one here.
			return nil, fmt.Errorf("provider download was interrupted")
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unsuccessful request to %s: %s", url, resp.Status)
	}

	f, err := ioutil.TempFile("", "terraform-provider")
	if err != nil {
		return nil, fmt.Errorf("failed to open temporary file to download from %s", url)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	// We'll borrow go-getter's "cancelable copy" implementation here so that
	// the download can potentially be interrupted partway through.
	n, err := getter.Copy(ctx, f, resp.Body)
	if err == nil && n < resp.ContentLength {
		err = fmt.Errorf("incorrect response size: expected %d bytes, but got %d bytes", resp.ContentLength, n)
	}
	if err != nil {
		return nil, err
	}

	archiveFilename := f.Name()
	localLocation := getproviders.PackageLocalArchive(archiveFilename)

	var authResult *getproviders.PackageAuthenticationResult
	if meta.Authentication != nil {
		if authResult, err = meta.Authentication.AuthenticatePackage(localLocation); err != nil {
			return authResult, err
		}
	}

	// We can now delegate to installFromLocalArchive for extraction. To do so,
	// we construct a new package meta description using the local archive
	// path as the location, and skipping authentication. installFromLocalMeta
	// is responsible for verifying that the archive matches the allowedHashes,
	// though.
	localMeta := getproviders.PackageMeta{
		Provider:         meta.Provider,
		Version:          meta.Version,
		ProtocolVersions: meta.ProtocolVersions,
		TargetPlatform:   meta.TargetPlatform,
		Filename:         meta.Filename,
		Location:         localLocation,
		Authentication:   nil,
	}
	if _, err := installFromLocalArchive(ctx, localMeta, targetDir, allowedHashes); err != nil {
		return nil, err
	}
	return authResult, nil
}

func installFromLocalArchive(ctx context.Context, meta getproviders.PackageMeta, targetDir string, allowedHashes []getproviders.Hash) (*getproviders.PackageAuthenticationResult, error) {
	var authResult *getproviders.PackageAuthenticationResult
	if meta.Authentication != nil {
		var err error
		if authResult, err = meta.Authentication.AuthenticatePackage(meta.Location); err != nil {
			return nil, err
		}
	}

	if len(allowedHashes) > 0 {
		if matches, err := meta.MatchesAnyHash(allowedHashes); err != nil {
			return authResult, fmt.Errorf(
				"failed to calculate checksum for %s %s package at %s: %s",
				meta.Provider, meta.Version, meta.Location, err,
			)
		} else if !matches {
			return authResult, fmt.Errorf(
				"the current package for %s %s doesn't match any of the checksums previously recorded in the dependency lock file",
				meta.Provider, meta.Version,
			)
		}
	}

	filename := meta.Location.String()

	err := unzip.Decompress(targetDir, filename, true, 0000)
	if err != nil {
		return authResult, err
	}

	return authResult, nil
}

// installFromLocalDir is the implementation of both installing a package from
// a local directory source _and_ of linking a package from another cache
// in LinkFromOtherCache, because they both do fundamentally the same
// operation: symlink if possible, or deep-copy otherwise.
func installFromLocalDir(ctx context.Context, meta getproviders.PackageMeta, targetDir string, allowedHashes []getproviders.Hash) (*getproviders.PackageAuthenticationResult, error) {
	sourceDir := meta.Location.String()

	absNew, err := filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to make target path %s absolute: %s", targetDir, err)
	}
	absCurrent, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to make source path %s absolute: %s", sourceDir, err)
	}

	// Before we do anything else, we'll do a quick check to make sure that
	// these two paths are not pointing at the same physical directory on
	// disk. This compares the files by their OS-level device and directory
	// entry identifiers, not by their virtual filesystem paths.
	if same, err := copy.SameFile(absNew, absCurrent); same {
		return nil, fmt.Errorf("cannot install existing provider directory %s to itself", targetDir)
	} else if err != nil {
		return nil, fmt.Errorf("failed to determine if %s and %s are the same: %s", sourceDir, targetDir, err)
	}

	var authResult *getproviders.PackageAuthenticationResult
	if meta.Authentication != nil {
		// (we have this here for completeness but note that local filesystem
		// mirrors typically don't include enough information for package
		// authentication and so we'll rarely get in here in practice.)
		var err error
		if authResult, err = meta.Authentication.AuthenticatePackage(meta.Location); err != nil {
			return nil, err
		}
	}

	// If the caller provided at least one hash in allowedHashes then at
	// least one of those hashes ought to match. However, for local directories
	// in particular we can't actually verify the legacy "zh:" hash scheme
	// because it requires access to the original .zip archive, and so as a
	// measure of pragmatism we'll treat a set of hashes where all are "zh:"
	// the same as no hashes at all, and let anything pass. This is definitely
	// non-ideal but accepted for two reasons:
	// - Packages we find on local disk can be considered a little more trusted
	//   than packages coming from over the network, because we assume that
	//   they were either placed intentionally by an operator or they were
	//   automatically installed by a previous network operation that would've
	//   itself verified the hashes.
	// - Our installer makes a concerted effort to record at least one new-style
	//   hash for each lock entry, so we should very rarely end up in this
	//   situation anyway.
	suitableHashCount := 0
	for _, hash := range allowedHashes {
		if !hash.HasScheme(getproviders.HashSchemeZip) {
			suitableHashCount++
		}
	}
	if suitableHashCount > 0 {
		if matches, err := meta.MatchesAnyHash(allowedHashes); err != nil {
			return authResult, fmt.Errorf(
				"failed to calculate checksum for %s %s package at %s: %s",
				meta.Provider, meta.Version, meta.Location, err,
			)
		} else if !matches {
			return authResult, fmt.Errorf(
				"the local package for %s %s doesn't match any of the checksums previously recorded in the dependency lock file (this might be because the available checksums are for packages targeting different platforms)",
				meta.Provider, meta.Version,
			)
		}
	}

	// Delete anything that's already present at this path first.
	err = os.RemoveAll(targetDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to remove existing %s before linking it to %s: %s", sourceDir, targetDir, err)
	}

	// We'll prefer to create a symlink if possible, but we'll fall back to
	// a recursive copy if symlink creation fails. It could fail for a number
	// of reasons, including being on Windows 8 without administrator
	// privileges or being on a legacy filesystem like FAT that has no way
	// to represent a symlink. (Generalized symlink support for Windows was
	// introduced in a Windows 10 minor update.)
	//
	// We use an absolute path for the symlink to reduce the risk of it being
	// broken by moving things around later, since the source directory is
	// likely to be a shared directory independent on any particular target
	// and thus we can't assume that they will move around together.
	linkTarget := absCurrent

	parentDir := filepath.Dir(absNew)
	err = os.MkdirAll(parentDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create parent directories leading to %s: %s", targetDir, err)
	}

	err = os.Symlink(linkTarget, absNew)
	if err == nil {
		// Success, then!
		return nil, nil
	}

	// If we get down here then symlinking failed and we need a deep copy
	// instead. To make a copy, we first need to create the target directory,
	// which would otherwise be a symlink.
	err = os.Mkdir(absNew, 0755)
	if err != nil && os.IsExist(err) {
		return nil, fmt.Errorf("failed to create directory %s: %s", absNew, err)
	}
	err = copy.CopyDir(absNew, absCurrent)
	if err != nil {
		return nil, fmt.Errorf("failed to either symlink or copy %s to %s: %s", absCurrent, absNew, err)
	}

	// If we got here then apparently our copy succeeded, so we're done.
	return nil, nil
}
