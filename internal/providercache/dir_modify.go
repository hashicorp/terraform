package providercache

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/getproviders"
)

// InstallPackage takes a metadata object describing a package available for
// installation, retrieves that package, and installs it into the receiving
// cache directory.
func (d *Dir) InstallPackage(ctx context.Context, meta getproviders.PackageMeta) (*getproviders.PackageAuthenticationResult, error) {
	if meta.TargetPlatform != d.targetPlatform {
		return nil, fmt.Errorf("can't install %s package into cache directory expecting %s", meta.TargetPlatform, d.targetPlatform)
	}
	newPath := getproviders.UnpackedDirectoryPathForPackage(
		d.baseDir, meta.Provider, meta.Version, d.targetPlatform,
	)

	// Invalidate our metaCache so that subsequent read calls will re-scan to
	// incorporate any changes we make here.
	d.metaCache = nil

	log.Printf("[TRACE] providercache.Dir.InstallPackage: installing %s v%s from %s", meta.Provider, meta.Version, meta.Location)
	switch meta.Location.(type) {
	case getproviders.PackageHTTPURL:
		return installFromHTTPURL(ctx, meta, newPath)
	case getproviders.PackageLocalArchive:
		return installFromLocalArchive(ctx, meta, newPath)
	case getproviders.PackageLocalDir:
		return installFromLocalDir(ctx, meta, newPath)
	default:
		// Should not get here, because the above should be exhaustive for
		// all implementations of getproviders.Location.
		return nil, fmt.Errorf("don't know how to install from a %T location", meta.Location)
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
	log.Printf("[TRACE] providercache.Dir.LinkFromOtherCache: linking %s v%s from existing cache %s to %s", entry.Provider, entry.Version, currentPath, newPath)

	// Invalidate our metaCache so that subsequent read calls will re-scan to
	// incorporate any changes we make here.
	d.metaCache = nil

	// We re-use the process of installing from a local directory here, because
	// the two operations are fundamentally the same: symlink if possible,
	// deep-copy otherwise.
	meta := getproviders.PackageMeta{
		Provider: entry.Provider,
		Version:  entry.Version,

		// FIXME: How do we populate this?
		ProtocolVersions: nil,
		TargetPlatform:   d.targetPlatform,

		// Because this is already unpacked, the filename is synthetic
		// based on the standard naming scheme.
		Filename: fmt.Sprintf("terraform-provider-%s_%s_%s.zip",
			entry.Provider.Type, entry.Version, d.targetPlatform),
		Location: getproviders.PackageLocalDir(currentPath),
	}
	_, err := installFromLocalDir(context.TODO(), meta, newPath)
	return err
}
