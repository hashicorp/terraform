package command

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	plugin "github.com/hashicorp/go-plugin"

	"github.com/hashicorp/terraform/internal/addrs"
	terraformProvider "github.com/hashicorp/terraform/internal/builtin/providers/terraform"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/moduletest"
	tfplugin "github.com/hashicorp/terraform/internal/plugin"
	tfplugin6 "github.com/hashicorp/terraform/internal/plugin6"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// The TF_DISABLE_PLUGIN_TLS environment variable is intended only for use by
// the plugin SDK test framework, to reduce startup overhead when rapidly
// launching and killing lots of instances of the same provider.
//
// This is not intended to be set by end-users.
var enableProviderAutoMTLS = os.Getenv("TF_DISABLE_PLUGIN_TLS") == ""

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
	var builtinProviderTypes []string
	for ty := range m.internalProviders() {
		builtinProviderTypes = append(builtinProviderTypes, ty)
	}
	inst.SetBuiltInProviderTypes(builtinProviderTypes)
	unmanagedProviderTypes := make(map[addrs.Provider]struct{}, len(m.UnmanagedProviders))
	for ty := range m.UnmanagedProviders {
		unmanagedProviderTypes[ty] = struct{}{}
	}
	inst.SetUnmanagedProviderTypes(unmanagedProviderTypes)
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
	dir := filepath.Join(m.DataDir(), "providers")
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
	if len(m.ProviderDevOverrides) == 0 {
		return nil
	}
	var detailMsg strings.Builder
	detailMsg.WriteString("The following provider development overrides are set in the CLI configuration:\n")
	for addr, path := range m.ProviderDevOverrides {
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
	if len(m.ProviderDevOverrides) == 0 {
		return nil
	}
	var detailMsg strings.Builder
	detailMsg.WriteString("The following provider development overrides are set in the CLI configuration:\n")
	for addr, path := range m.ProviderDevOverrides {
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

// providerFactories uses the selections made previously by an installer in
// the local cache directory (m.providerLocalCacheDir) to produce a map
// from provider addresses to factory functions to create instances of
// those providers.
//
// providerFactories will return an error if the installer's selections cannot
// be honored with what is currently in the cache, such as if a selected
// package has been removed from the cache or if the contents of a selected
// package have been modified outside of the installer. If it returns an error,
// the returned map may be incomplete or invalid, but will be as complete
// as possible given the cause of the error.
func (m *Meta) providerFactories() (map[addrs.Provider]providers.Factory, error) {
	locks, diags := m.lockedDependencies()
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to read dependency lock file: %s", diags.Err())
	}

	// We'll always run through all of our providers, even if one of them
	// encounters an error, so that we can potentially report multiple errors
	// where appropriate and so that callers can potentially make use of the
	// partial result we return if e.g. they want to enumerate which providers
	// are available, or call into one of the providers that didn't fail.
	var err error

	// For the providers from the lock file, we expect them to be already
	// available in the provider cache because "terraform init" should already
	// have put them there.
	providerLocks := locks.AllProviders()
	cacheDir := m.providerLocalCacheDir()

	// The internal providers are _always_ available, even if the configuration
	// doesn't request them, because they don't need any special installation
	// and they'll just be ignored if not used.
	internalFactories := m.internalProviders()

	// We have two different special cases aimed at provider development
	// use-cases, which are not for "production" use:
	// - The CLI config can specify that a particular provider should always
	// use a plugin from a particular local directory, ignoring anything the
	// lock file or cache directory might have to say about it. This is useful
	// for manual testing of local development builds.
	// - The Terraform SDK test harness (and possibly other callers in future)
	// can ask that we use its own already-started provider servers, which we
	// call "unmanaged" because Terraform isn't responsible for starting
	// and stopping them. This is intended for automated testing where a
	// calling harness is responsible both for starting the provider server
	// and orchestrating one or more non-interactive Terraform runs that then
	// exercise it.
	// Unmanaged providers take precedence over overridden providers because
	// overrides are typically a "session-level" setting while unmanaged
	// providers are typically scoped to a single unattended command.
	devOverrideProviders := m.ProviderDevOverrides
	unmanagedProviders := m.UnmanagedProviders

	factories := make(map[addrs.Provider]providers.Factory, len(providerLocks)+len(internalFactories)+len(unmanagedProviders))
	for name, factory := range internalFactories {
		factories[addrs.NewBuiltInProvider(name)] = factory
	}
	for provider, lock := range providerLocks {
		reportError := func(thisErr error) {
			err = multierror.Append(err, thisErr)
			// We'll populate a provider factory that just echoes our error
			// again if called, which allows us to still report a helpful
			// error even if it gets detected downstream somewhere from the
			// caller using our partial result.
			factories[provider] = providerFactoryError(thisErr)
		}

		version := lock.Version()
		cached := cacheDir.ProviderVersion(provider, version)
		if cached == nil {
			reportError(fmt.Errorf(
				"there is no package for %s %s cached in %s",
				provider, version, cacheDir.BasePath(),
			))
			continue
		}
		// The cached package must match one of the checksums recorded in
		// the lock file, if any.
		if allowedHashes := lock.PreferredHashes(); len(allowedHashes) != 0 {
			matched, err := cached.MatchesAnyHash(allowedHashes)
			if err != nil {
				reportError(fmt.Errorf(
					"failed to verify checksum of %s %s package cached in in %s: %s",
					provider, version, cacheDir.BasePath(), err,
				))
				continue
			}
			if !matched {
				reportError(fmt.Errorf(
					"the cached package for %s %s (in %s) does not match any of the checksums recorded in the dependency lock file",
					provider, version, cacheDir.BasePath(),
				))
				continue
			}
		}
		factories[provider] = providerFactory(cached)
	}
	for provider, localDir := range devOverrideProviders {
		// It's likely that providers in this map will conflict with providers
		// in providerLocks
		factories[provider] = devOverrideProviderFactory(provider, localDir)
	}
	for provider, reattach := range unmanagedProviders {
		factories[provider] = unmanagedProviderFactory(provider, reattach)
	}
	return factories, err
}

func (m *Meta) internalProviders() map[string]providers.Factory {
	return map[string]providers.Factory{
		"terraform": func() (providers.Interface, error) {
			return terraformProvider.NewProvider(), nil
		},
		"test": func() (providers.Interface, error) {
			return moduletest.NewProvider(), nil
		},
	}
}

// providerFactory produces a provider factory that runs up the executable
// file in the given cache package and uses go-plugin to implement
// providers.Interface against it.
func providerFactory(meta *providercache.CachedProvider) providers.Factory {
	return func() (providers.Interface, error) {
		execFile, err := meta.ExecutableFile()
		if err != nil {
			return nil, err
		}

		config := &plugin.ClientConfig{
			HandshakeConfig:  tfplugin.Handshake,
			Logger:           logging.NewProviderLogger(""),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			Managed:          true,
			Cmd:              exec.Command(execFile),
			AutoMTLS:         enableProviderAutoMTLS,
			VersionedPlugins: tfplugin.VersionedPlugins,
			SyncStdout:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stdout", meta.Provider)),
			SyncStderr:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stderr", meta.Provider)),
		}

		client := plugin.NewClient(config)
		rpcClient, err := client.Client()
		if err != nil {
			return nil, err
		}

		raw, err := rpcClient.Dispense(tfplugin.ProviderPluginName)
		if err != nil {
			return nil, err
		}

		// store the client so that the plugin can kill the child process
		protoVer := client.NegotiatedVersion()
		switch protoVer {
		case 5:
			p := raw.(*tfplugin.GRPCProvider)
			p.PluginClient = client
			return p, nil
		case 6:
			p := raw.(*tfplugin6.GRPCProvider)
			p.PluginClient = client
			return p, nil
		default:
			panic("unsupported protocol version")
		}
	}
}

func devOverrideProviderFactory(provider addrs.Provider, localDir getproviders.PackageLocalDir) providers.Factory {
	// A dev override is essentially a synthetic cache entry for our purposes
	// here, so that's how we'll construct it. The providerFactory function
	// doesn't actually care about the version, so we can leave it
	// unspecified: overridden providers are not explicitly versioned.
	log.Printf("[DEBUG] Provider %s is overridden to load from %s", provider, localDir)
	return providerFactory(&providercache.CachedProvider{
		Provider:   provider,
		Version:    getproviders.UnspecifiedVersion,
		PackageDir: string(localDir),
	})
}

// unmanagedProviderFactory produces a provider factory that uses the passed
// reattach information to connect to go-plugin processes that are already
// running, and implements providers.Interface against it.
func unmanagedProviderFactory(provider addrs.Provider, reattach *plugin.ReattachConfig) providers.Factory {
	return func() (providers.Interface, error) {
		config := &plugin.ClientConfig{
			HandshakeConfig:  tfplugin.Handshake,
			Logger:           logging.NewProviderLogger("unmanaged."),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			Managed:          false,
			Reattach:         reattach,
			SyncStdout:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stdout", provider)),
			SyncStderr:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stderr", provider)),
		}

		if reattach.ProtocolVersion == 0 {
			// As of the 0.15 release, sdk.v2 doesn't include the protocol
			// version in the ReattachConfig (only recently added to
			// go-plugin), so client.NegotiatedVersion() always returns 0. We
			// assume that an unmanaged provider reporting protocol version 0 is
			// actually using proto v5 for backwards compatibility.
			if defaultPlugins, ok := tfplugin.VersionedPlugins[5]; ok {
				config.Plugins = defaultPlugins
			} else {
				return nil, errors.New("no supported plugins for protocol 0")
			}
		} else if plugins, ok := tfplugin.VersionedPlugins[reattach.ProtocolVersion]; !ok {
			return nil, fmt.Errorf("no supported plugins for protocol %d", reattach.ProtocolVersion)
		} else {
			config.Plugins = plugins
		}

		client := plugin.NewClient(config)
		rpcClient, err := client.Client()
		if err != nil {
			return nil, err
		}

		raw, err := rpcClient.Dispense(tfplugin.ProviderPluginName)
		if err != nil {
			return nil, err
		}

		// store the client so that the plugin can kill the child process
		protoVer := client.NegotiatedVersion()
		switch protoVer {
		case 0, 5:
			// As of the 0.15 release, sdk.v2 doesn't include the protocol
			// version in the ReattachConfig (only recently added to
			// go-plugin), so client.NegotiatedVersion() always returns 0. We
			// assume that an unmanaged provider reporting protocol version 0 is
			// actually using proto v5 for backwards compatibility.
			p := raw.(*tfplugin.GRPCProvider)
			p.PluginClient = client
			return p, nil
		case 6:
			p := raw.(*tfplugin6.GRPCProvider)
			p.PluginClient = client
			return p, nil
		default:
			return nil, fmt.Errorf("unsupported protocol version %d", protoVer)
		}
	}
}

// providerFactoryError is a stub providers.Factory that returns an error
// when called. It's used to allow providerFactories to still produce a
// factory for each available provider in an error case, for situations
// where the caller can do something useful with that partial result.
func providerFactoryError(err error) providers.Factory {
	return func() (providers.Interface, error) {
		return nil, err
	}
}
