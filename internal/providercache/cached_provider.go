package providercache

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// CachedProvider represents a provider package in a cache directory.
type CachedProvider struct {
	// Provider and Version together identify the specific provider version
	// this cache entry represents.
	Provider addrs.Provider
	Version  getproviders.Version

	// PackageDir is the local filesystem path to the root directory where
	// the provider's distribution archive was unpacked.
	//
	// The path always uses slashes as path separators, even on Windows, so
	// that the results are consistent between platforms. Windows accepts
	// both slashes and backslashes as long as the separators are consistent
	// within a particular path string.
	PackageDir string

	// ExecutableFile is the local filesystem path to the main plugin executable
	// for the provider, which is always a file within the directory given
	// in PackageDir.
	//
	// The path always uses slashes as path separators, even on Windows, so
	// that the results are consistent between platforms. Windows accepts
	// both slashes and backslashes as long as the separators are consistent
	// within a particular path string.
	ExecutableFile string
}
