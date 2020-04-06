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
func (d *Dir) InstallPackage(ctx context.Context, meta getproviders.PackageMeta) error {
	if meta.TargetPlatform != d.targetPlatform {
		return fmt.Errorf("can't install %s package into cache directory expecting %s", meta.TargetPlatform, d.targetPlatform)
	}
	newPath := getproviders.UnpackedDirectoryPathForPackage(
		d.baseDir, meta.Provider, meta.Version, d.targetPlatform,
	)

	// Invalidate our metaCache so that subsequent read calls will re-scan to
	// incorporate any changes we make here.
	d.metaCache = nil

	log.Printf("[TRACE] providercache.Dir.InstallPackage: installing %s v%s from %s", meta.Provider, meta.Version, meta.Location)
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
	log.Printf("[TRACE] providercache.Dir.LinkFromOtherCache: linking %s v%s from existing cache %s to %s", entry.Provider, entry.Version, currentPath, newPath)

	// Invalidate our metaCache so that subsequent read calls will re-scan to
	// incorporate any changes we make here.
	d.metaCache = nil

	// We re-use the process of installing from a local directory here, because
	// the two operations are fundamentally the same: symlink if possible,
	// deep-copy otherwise.
	return installFromLocalDir(context.TODO(), currentPath, newPath)
}
