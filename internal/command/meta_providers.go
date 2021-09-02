package command

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// providerInstaller returns an object that knows how to install providers and
// how to recover the selections from a prior installation process.
//
// The resulting provider installer is constructed from the results of
// the other methods providerLocalCacheDir, providerGlobalCacheDir, and
// providerInstallSource.
//
// Only one object returned from this method should be live at any time,
// because objects inside contain caches that must be maintained properly.
// Because this method wraps a result from providerLocalCacheDir, that
// limitation applies also to results from that method.
func (m *Meta) providerInstaller() *providercache.Installer {
	return m.providerInstallerCustomSource(m.providerInstallSource())
}

// providerInstallerCustomSource is a variant of providerInstaller that
// allows the caller to specify a different installation source than the one
// that would naturally be selected.
//
// The result of this method has the same dependencies and constraints as
// providerInstaller.
//
// The result of providerInstallerCustomSource differs from
// providerInstaller only in how it determines package installation locations
// during EnsureProviderVersions. A caller that doesn't call
// EnsureProviderVersions (anything other than "terraform init") can safely
// just use the providerInstaller method unconditionally.
func (m *Meta) providerInstallerCustomSource(source getproviders.Source) *providercache.Installer {
	targetDir := m.providerLocalCacheDir()
	globalCacheDir := m.providerGlobalCacheDir()
	inst := providercache.NewInstaller(targetDir, source)
	if globalCacheDir != nil {
		inst.SetGlobalCacheDir(globalCacheDir)
	}
	finder, moreDiags := m.pluginFinder()
	if moreDiags.HasErrors() {
		log.Printf("[WARN] Problems setting up our plugin finder: %s", moreDiags.Err().Error())
		// We'll make a best effort to use this even if there's an error setting
		// it up, because anything that'd make our finder fail to build here
		// will fail downstream during installation anyway.
	}
	builtinProviderTypes := finder.BuiltinProviderTypes()
	inst.SetBuiltInProviderTypes(builtinProviderTypes)
	unmanagedProviderAddrs := finder.UnmanagedProviderAddrs()
	// inst.SetUnmanagedProviderTypes wants a map representing a set, so
	// we need to project the result from upstream. :(
	unmanagedProviderAddrsSet := make(map[addrs.Provider]struct{}, len(unmanagedProviderAddrs))
	for _, addr := range unmanagedProviderAddrs {
		unmanagedProviderAddrsSet[addr] = struct{}{}
	}
	inst.SetUnmanagedProviderTypes(unmanagedProviderAddrsSet)
	return inst
}

// providerCustomLocalDirectorySource produces a provider source that consults
// only the given local filesystem directories for plugins to install.
//
// This is used to implement the -plugin-dir option for "terraform init", where
// the result of this method is used instead of what would've been returned
// from m.providerInstallSource.
//
// If the given list of directories is empty then the resulting source will
// have no providers available for installation at all.
func (m *Meta) providerCustomLocalDirectorySource(dirs []string) getproviders.Source {
	var ret getproviders.MultiSource
	for _, dir := range dirs {
		ret = append(ret, getproviders.MultiSourceSelector{
			Source: getproviders.NewFilesystemMirrorSource(dir),
		})
	}
	return ret
}

// providerLocalCacheDir returns an object representing the
// configuration-specific local cache directory. This is the
// only location consulted for provider plugin packages for Terraform
// operations other than provider installation.
//
// Only the provider installer (in "terraform init") is permitted to make
// modifications to this cache directory. All other commands must treat it
// as read-only.
//
// Only one object returned from this method should be live at any time,
// because objects inside contain caches that must be maintained properly.
func (m *Meta) providerLocalCacheDir() *providercache.Dir {
	m.fixupMissingWorkingDir()
	dir := m.WorkingDir.ProviderLocalCacheDir()
	return providercache.NewDir(dir)
}

// providerGlobalCacheDir returns an object representing the shared global
// provider cache directory, used as a read-through cache when installing
// new provider plugin packages.
//
// This function may return nil, in which case there is no global cache
// configured and new packages should be downloaded directly into individual
// configuration-specific cache directories.
//
// Only one object returned from this method should be live at any time,
// because objects inside contain caches that must be maintained properly.
func (m *Meta) providerGlobalCacheDir() *providercache.Dir {
	dir := m.PluginCacheDir
	if dir == "" {
		return nil // cache disabled
	}
	return providercache.NewDir(dir)
}

// providerInstallSource returns an object that knows how to consult one or
// more external sources to determine the availability of and package
// locations for versions of Terraform providers that are available for
// automatic installation.
//
// This returns the standard provider install source that consults a number
// of directories selected either automatically or via the CLI configuration.
// Users may choose to override this during a "terraform init" command by
// specifying one or more -plugin-dir options, in which case the installation
// process will construct its own source consulting only those directories
// and use that instead.
func (m *Meta) providerInstallSource() getproviders.Source {
	// A provider source should always be provided in normal use, but our
	// unit tests might not always populate Meta fully and so we'll be robust
	// by returning a non-nil source that just always answers that no plugins
	// are available.
	if m.ProviderSource == nil {
		// A multi-source with no underlying sources is effectively an
		// always-empty source.
		return getproviders.MultiSource(nil)
	}
	return m.ProviderSource
}

// providerDevOverrideInitWarnings returns a diagnostics that contains at
// least one warning if and only if there is at least one provider development
// override in effect. If not, the result is always empty. The result never
// contains error diagnostics.
//
// The init command can use this to include a warning that the results
// may differ from what's expected due to the development overrides. For
// other commands, providerDevOverrideRuntimeWarnings should be used.
func (m *Meta) providerDevOverrideInitWarnings() tfdiags.Diagnostics {
	pluginFinder, finderDiags := m.pluginFinder()
	if finderDiags.HasErrors() {
		// If the finder isn't buildable then we'll end up failing to do
		// anything with providers anyway, so we'll just bail here and let
		// the same errors surface via another codepath.
		return nil
	}
	devOverrides := pluginFinder.ProviderDevOverrides()
	if len(devOverrides) == 0 {
		return nil
	}
	var detailMsg strings.Builder
	detailMsg.WriteString("The following provider development overrides are set in the CLI configuration:\n")
	for addr, path := range devOverrides {
		detailMsg.WriteString(fmt.Sprintf(" - %s in %s\n", addr.ForDisplay(), path))
	}
	detailMsg.WriteString("\nSkip terraform init when using provider development overrides. It is not necessary and may error unexpectedly.")
	return tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Warning,
			"Provider development overrides are in effect",
			detailMsg.String(),
		),
	}
}

// providerDevOverrideRuntimeWarnings returns a diagnostics that contains at
// least one warning if and only if there is at least one provider development
// override in effect. If not, the result is always empty. The result never
// contains error diagnostics.
//
// Certain commands can use this to include a warning that their results
// may differ from what's expected due to the development overrides. It's
// not necessary to bother the user with this warning on every command, but
// it's helpful to return it on commands that have externally-visible side
// effects and on commands that are used to verify conformance to schemas.
//
// See providerDevOverrideInitWarnings for warnings specific to the init
// command.
func (m *Meta) providerDevOverrideRuntimeWarnings() tfdiags.Diagnostics {
	pluginFinder, finderDiags := m.pluginFinder()
	if finderDiags.HasErrors() {
		// If the finder isn't buildable then we'll end up failing to do
		// anything with providers anyway, so we'll just bail here and let
		// the same errors surface via another codepath.
		return nil
	}
	devOverrides := pluginFinder.ProviderDevOverrides()
	if len(devOverrides) == 0 {
		return nil
	}
	var detailMsg strings.Builder
	detailMsg.WriteString("The following provider development overrides are set in the CLI configuration:\n")
	for addr, path := range devOverrides {
		detailMsg.WriteString(fmt.Sprintf(" - %s in %s\n", addr.ForDisplay(), path))
	}
	detailMsg.WriteString("\nThe behavior may therefore not match any released version of the provider and applying changes may cause the state to become incompatible with published releases.")
	return tfdiags.Diagnostics{
		tfdiags.Sourceless(
			tfdiags.Warning,
			"Provider development overrides are in effect",
			detailMsg.String(),
		),
	}
}
