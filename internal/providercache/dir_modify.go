package providercache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/internal/copydir"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// InstallPackage takes a metadata object describing a package available for
// installation, retrieves that package, and installs it into the receiving
// cache directory.
func (d *Dir) InstallPackage(ctx context.Context, meta getproviders.PackageMeta) error {
	if meta.TargetPlatform != d.targetPlatform {
		return fmt.Errorf("can't install %s package into cache directory expecting %s", meta.TargetPlatform, d.targetPlatform)
	}
	newPath := getproviders.UnpackedDirectoryPathForPackage(
		d.baseDir, meta.Provider, meta.Version, d.targetPlatform,
	)

	switch location := meta.Location.(type) {
	case getproviders.PackageHTTPURL:
		return installFromHTTPURL(ctx, string(location), newPath)
	case getproviders.PackageLocalArchive:
		return installFromLocalArchive(ctx, string(location), newPath)
	case getproviders.PackageLocalDir:
		return installFromLocalDir(ctx, string(location), newPath)
	default:
		// Should not get here, because the above should be exhaustive for
		// all implementations of getproviders.Location.
		return fmt.Errorf("don't know how to install from a %T location", location)
	}
}

// LinkFromOtherCache takes a CachedProvider value produced from another Dir
// and links it into the cache represented by the receiver Dir.
//
// This is used to implement tiered caching, where new providers are first
// populated into a system-wide shared cache and then linked from there into
// a configuration-specific local cache.
//
// It's invalid to link a CachedProvider from a particular Dir into that same
// Dir, because that would otherwise potentially replace a real package
// directory with a circular link back to itself.
func (d *Dir) LinkFromOtherCache(entry *CachedProvider) error {
	newPath := getproviders.UnpackedDirectoryPathForPackage(
		d.baseDir, entry.Provider, entry.Version, d.targetPlatform,
	)
	currentPath := entry.PackageDir

	absNew, err := filepath.Abs(newPath)
	if err != nil {
		return fmt.Errorf("failed to make new path %s absolute: %s", newPath, err)
	}
	absCurrent, err := filepath.Abs(currentPath)
	if err != nil {
		return fmt.Errorf("failed to make existing cache path %s absolute: %s", currentPath, err)
	}

	// Before we do anything else, we'll do a quick check to make sure that
	// these two paths are not pointing at the same physical directory on
	// disk. This compares the files by their OS-level device and directory
	// entry identifiers, not by their virtual filesystem paths.
	if same, err := copydir.SameFile(absNew, absCurrent); same {
		return fmt.Errorf("cannot link existing cache path %s to itself", newPath)
	} else if err != nil {
		return fmt.Errorf("failed to determine if %s and %s are the same: %s", currentPath, newPath, err)
	}

	// Invalidate our metaCache so that subsequent read calls will re-scan to
	// incorporate any changes we make here.
	d.metaCache = nil

	// Delete anything that's already present at this path first.
	err = os.RemoveAll(currentPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing %s before linking it to %s: %s", currentPath, newPath, err)
	}

	// We'll prefer to create a symlink if possible, but we'll fall back to
	// a recursive copy if symlink creation fails. It could fail for a number
	// of reasons, including being on Windows 8 without administrator
	// privileges or being on a legacy filesystem like FAT that has no way
	// to represent a symlink. (Generalized symlink support for Windows was
	// introduced in a Windows 10 minor update.)
	//
	// We'd prefer to use a relative path for the symlink to reduce the risk
	// of it being broken by moving things around later, but we'll fall back
	// on the absolute path we already calculated if that isn't possible
	// (e.g. because the two paths are on different "volumes" on an OS with
	// that concept, like Windows with drive letters and UNC host/share names.)
	linkTarget, err := filepath.Rel(newPath, absCurrent)
	if err != nil {
		linkTarget = absCurrent
	}

	parentDir := filepath.Dir(absNew)
	err = os.MkdirAll(parentDir, 0755)
	if err != nil && os.IsExist(err) {
		return fmt.Errorf("failed to create parent directories leading to %s: %s", newPath, err)
	}

	err = os.Symlink(linkTarget, absNew)
	if err == nil {
		// Success, then!
		return nil
	}

	// If we get down here then symlinking failed and we need a deep copy
	// instead.
	err = copydir.CopyDir(absNew, absCurrent)
	if err != nil {
		return fmt.Errorf("failed to either symlink or copy %s to %s: %s", absCurrent, absNew, err)
	}

	// If we got here then apparently our copy succeeded, so we're done.
	return nil
}
