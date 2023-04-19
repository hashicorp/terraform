package sourcebundle

import (
	"fmt"

	"github.com/apparentlymart/go-versions/versions"

	"github.com/hashicorp/terraform/internal/addrs"
)

// Manifest is a table of contents for a source bundle, which can be used to
// translate module source addresses into local paths relative to the root
// of the corresponding source bundle.
//
// If a caller first extracts a bundle to a local filesystem directory and then
// passes its path to [os.DirFS], or otherwise makes its content available as
// an [io/fs.FS], then the local paths returned from the manifest can be used
// with the filesystem API to retrieve files and list directories.
type Manifest struct {
	// modulePackageDirs remembers the first level of subdirectory inside
	// the source bundle for each distinct module package.
	//
	// Subdirectory names are not necessarily unique: a source bundle builder
	// will reuse a source directory if two module packages have identical
	// content, as can sometimes happen if the user specified the same real
	// module package using two non-equal addresses, such as specifying the
	// same Git commit both as a tag and as a full commit id.
	modulePackageDirs map[addrs.ModulePackage]string

	// registryPackageSources remembers the relationships between module
	// registry addresses and the real storage locations, since we use the
	// real storage locations as the key to the main lookup table above.
	registryPackageSources map[registryModuleVersion]addrs.ModuleSourceRemote
}

func newManifest() *Manifest {
	return &Manifest{
		modulePackageDirs:      make(map[addrs.ModulePackage]string),
		registryPackageSources: make(map[registryModuleVersion]addrs.ModuleSourceRemote),
	}
}

// GetRemoteSourcePath returns the local path, using [io/fs.FS] virtual path
// syntax, corresponding with the given remote module source source address.
//
// The resulting path is relative to the root of the source bundle's virtual
// filesystem.
func (m *Manifest) GetRemoteSourcePath(srcAddr addrs.ModuleSourceRemote) (string, error) {
	panic("not yet implemented")
}

// GetRemoteSourcePath returns the local path, using [io/fs.FS] virtual path
// syntax, corresponding with the given registry source source address and
// version.
//
// The resulting path is relative to the root of the source bundle's virtual
// filesystem.
func (m *Manifest) GetRegistrySourcePath(srcAddr addrs.ModuleSourceRegistry, version versions.Version) (string, error) {
	panic("not yet implemented")
}

// GetSourceLocationForDisplay walks the lookup table backwards to translate
// a path relative to the bundle root back into a remote package source address,
// so that callers can avoid reporting the internal package prefixes to
// end-users when describing errors, etc.
//
// If the given path doesn't correspond to any packages in the source bundle,
// the final boolean return value is false and the returned address is invalid.
func (m *Manifest) GetSourceLocationForDisplay(localPath string) (addrs.ModuleSourceRemote, bool) {
	panic("not yet implemented")
}

func (m *Manifest) saveModulePackageBundleDir(pkgAddr addrs.ModulePackage, dirName string) {
	if _, exists := m.modulePackageDirs[pkgAddr]; exists {
		panic(fmt.Sprintf("duplicate saveModulePackageBundleDir call for %s", pkgAddr))
	}
	m.modulePackageDirs[pkgAddr] = dirName
}

func (m *Manifest) getRegistryPackageSource(pkgAddr addrs.ModuleRegistryPackage, version versions.Version) (realAddr addrs.ModuleSourceRemote, exists bool) {
	k := registryModuleVersion{
		module:  pkgAddr,
		version: version,
	}
	ret, ok := m.registryPackageSources[k]
	return ret, ok
}

func (m *Manifest) saveRegistryPackageSource(pkgAddr addrs.ModuleRegistryPackage, version versions.Version, realAddr addrs.ModuleSourceRemote) {
	k := registryModuleVersion{
		module:  pkgAddr,
		version: version,
	}
	if _, exists := m.registryPackageSources[k]; exists {
		panic(fmt.Sprintf("duplicate saveRegistryPackageSource call for %s %s", pkgAddr, version))
	}
	m.registryPackageSources[k] = realAddr
}

type registryModuleVersion struct {
	module  addrs.ModuleRegistryPackage
	version versions.Version
}
