package plugins

import (
	"errors"
	"fmt"
	"log"
	"os/exec"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-plugin"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/logging"
	tfplugin "github.com/hashicorp/terraform/internal/plugin"
	"github.com/hashicorp/terraform/internal/plugin/discovery"
	tfplugin6 "github.com/hashicorp/terraform/internal/plugin6"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
)

// Selections represents a specific set of plugins chosen by a Finder, based
// on its settings for where to look and other constraints that must be
// applied.
type Selections struct {
	// providerPlugins retains the cache entry objects for the subset of
	// selected providers that are backed by real external plugins, as opposed
	// to the ones with special treatment such as builtins, dev overrides, etc.
	providerPlugins map[addrs.Provider]*providercache.CachedProvider

	// providerFactories represents the final set of available providers,
	// each one having a function to instantiate it which will vary depending
	// on whether it's a normal external plugin, a builtin, etc.
	//
	// Presence in this map doesn't guarantee it's actually instantiable. It
	// might fail to start up, or we might find when we run it that it doesn't
	// meet some constraint imposed by the context we're running in.
	providerFactories map[addrs.Provider]providers.Factory

	// provisionerFactories represents the final set of available provisioners,
	// each one having a function to instantiate it.
	provisionerFactories map[string]provisioners.Factory
}

// FindPlugins uses the settings in the receiver to select a specific set of
// provider plugins that will either meet the configured constraints or
// generate errors explaining why they don't.
func (f Finder) FindPlugins() (*Selections, error) {
	if f.testingOverrides != nil {
		log.Printf(
			"[TRACE] command/plugins: FindPlugins with test overrides: %d normal providers, %d built-in providers, and %d provisioners",
			len(f.testingOverrides.Providers), len(f.providerBuiltins), len(f.testingOverrides.Provisioners),
		)
		// Testing overrides usurp all of our usual work and just force
		// returning exactly what the overrider set. However, we do make an
		// exception for the built-in providers because some tests do still
		// expect those to behave properly even when overridden.
		providerFactories := make(map[addrs.Provider]providers.Factory, len(f.testingOverrides.Providers)+len(f.providerBuiltins))
		for addr, f := range f.testingOverrides.Providers {
			providerFactories[addr] = f
		}
		for typeName, f := range f.providerBuiltins {
			providerFactories[addrs.NewBuiltInProvider(typeName)] = f
		}
		return &Selections{
			providerFactories:    providerFactories,
			provisionerFactories: f.testingOverrides.Provisioners,
		}, nil
	}

	// NOTE: The reciever here makes one final copy of the Finder which
	// we'll use as the basis for our selections, so from now on we're using
	// pointers to that copy to ensure that things stay coherent for
	// each individual selections object.
	ret := &Selections{}
	err := ret.initProviders(&f)
	if err != nil {
		return ret, err
	}
	err = ret.initProvisioners(&f)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func (s *Selections) initProviders(f *Finder) error {
	// Ideally we want to defer single-provider-specific errors until
	// we're inside the factory function for that provider whereever that's
	// reasonable, because then we can return an error in a better context
	// and not need to aggregate together various different provider errors
	// all at once into our single error result here. Returning a non-nil
	// error from this method usually means that our search process failed
	// catastrophically, e.g. due to a misconfigured finder.

	var locks map[addrs.Provider]*depsfile.ProviderLock
	if f.dependencyLocks != nil {
		locks = f.dependencyLocks.AllProviders()
	}

	cacheDir := f.providerDir
	builtinFactories := f.providerBuiltins
	devOverrideProviders := f.providerDevOverrides
	unmanagedProviders := f.providersUnmanaged
	requiredProviders := f.providerRequirements

	// All of the providers that our Finder is configured to require must
	// have an entry in the lock file. This catches situations where we
	// wouldn't typically create a factory at all, and so we wouldn't catch
	// these otherwise.
	var err error
	for provider := range requiredProviders {
		if provider.IsBuiltIn() {
			continue
		}
		if _, exists := devOverrideProviders[provider]; exists {
			continue
		}
		if _, exists := unmanagedProviders[provider]; exists {
			continue
		}

		if _, exists := locks[provider]; !exists {
			err = multierror.Append(err, fmt.Errorf(
				"no dependency lock file entry for required provider %q",
				provider,
			))
		}
	}
	if err != nil {
		return err
	}

	plugins := make(map[addrs.Provider]*providercache.CachedProvider, len(locks))

	factories := make(map[addrs.Provider]providers.Factory, len(locks)+len(builtinFactories)+len(unmanagedProviders))
	for name, factory := range builtinFactories {
		factories[addrs.NewBuiltInProvider(name)] = factory
	}

	// The main factories are those which launch true external plugins from
	// our local cache directory.
	for provider, lock := range locks {
		plugins[provider], factories[provider] = providerFactory(f, s, lock, cacheDir)
	}

	// It's likely and expected that providers in these extra maps will
	// conflict with providers in providerLocks, because the installer would've
	// installed a published release of each provider which is what we're
	// overriding here.
	for provider, localDir := range devOverrideProviders {
		factories[provider] = devOverrideProviderFactory(provider, localDir, !f.providerDisableAutoMTLS)
	}
	for provider, reattach := range unmanagedProviders {
		factories[provider] = unmanagedProviderFactory(provider, reattach, !f.providerDisableAutoMTLS)
	}

	s.providerFactories = factories
	s.providerPlugins = plugins
	return nil
}

func (s *Selections) initProvisioners(f *Finder) error {
	dirs := f.provisionerSearchDirs
	plugins := discovery.FindPlugins("provisioner", dirs)
	plugins, _ = plugins.ValidateVersions()

	// For now our goal is to just find the latest version of each plugin
	// we have on the system. All provisioners should be at version 0.0.0
	// currently, so there should actually only be one instance of each plugin
	// name here, even though the discovery interface forces us to pretend
	// that might not be true.

	factories := make(map[string]provisioners.Factory)

	// Wire up the internal provisioners first. These might be overridden
	// by discovered provisioners below.
	for name, factory := range f.provisionerBuiltins {
		factories[name] = factory
	}

	byName := plugins.ByName()
	for name, metas := range byName {
		// Since we validated versions above and we partitioned the sets
		// by name, we're guaranteed that the metas in our set all have
		// valid versions and that there's at least one meta.
		newest := metas.Newest()

		// NOTE: In the original implementation of this "providerDisableAutoMTLS"
		// idea in the command package it was called "provider" but yet still
		// used for provisioners too. We've preserved that here for
		// compatibility, but it seems confusing to use a setting with
		// "provider" in it for provisioners too, so in future we should either
		// make this actually apply only to providers or rename it to be more
		// clearly general across all plugin types, once we've done some
		// research to see if there are any requirements that call for
		// provisioner plugins not doing this check.

		factories[name] = provisionerFactory(newest, !f.providerDisableAutoMTLS)
	}

	s.provisionerFactories = factories
	return nil
}

// ProviderFactories builds a map of provider factory functions based on the
// recieving sections.
//
// Terraform Core (terraform.NewContext, specifically) interacts with
// providers by calling a factory function each time it needs a new provider
// instance, and so the result of this method serves as the interface between
// Terraform Core and a plugin finder for the purpose of launching providers
// in particular. (Other plugin types have similar Finder methods, which
// callers must use separately.)
//
// This function always succeeds itself, because verification of the available
// providers is deferred until Terraform Core attempts to instantiate one.
// Some or all of the factory functions might therefore fail with an error
// in the event that the Finder is misconfigured or that the working directory
// is in an inconsistent state.
//
// The result only contains factories for providers included in the requirements
// set previously passed to WithProviderRequirements.
func (s *Selections) ProviderFactories() map[addrs.Provider]providers.Factory {
	if s == nil {
		return nil
	}
	return s.providerFactories
}

// ProvisionerFactories builds a map of provisioner factory functions based on
// the recieving selections.
//
// Terraform Core (terraform.NewContext, specifically) interacts with
// provisioners by calling a factory function each time it needs a new
// provisioner instance, and so the result of this method serves as the
// interface between Terraform Core and a plugin finder for the purpose of
// launching provisioners in particular. (Other plugin types have similar
// Finder methods, which callers must use separately.)
//
// This function always succeeds itself, because verification of the available
// provisioners is deferred until Terraform Core attempts to instantiate one.
// Some or all of the factory functions might therefore fail with an error
// in the event that the Finder is misconfigured or that the working directory
// is in an inconsistent state.
func (s *Selections) ProvisionerFactories() map[string]provisioners.Factory {
	if s == nil {
		return nil
	}
	return s.provisionerFactories
}

// ProviderChecksums returns the checksums for the exact provider plugin
// packages we've selected, using whatever hash format is the default for
// the current Terraform version.
//
// This is a suitable table to record in a saved plan file and then provide
// to a future Finder's WithForcedProviderChecksums to make sure that we can
// only apply a plan with exactly identical providers than what created it.
func (s *Selections) ProviderChecksums() map[addrs.Provider]getproviders.Hash {
	// We wait until we're asked for ProviderChecksums to do the work for
	// this, because we only actually need this answer if we're saving a
	// plan file.
	// In principle we could memoize our hash lookups from our provider
	// factory functions to avoid re-reading in here, but the getproviders
	// API for verifying checksums makes that awkward to achieve and so
	// we'll accept just recalculating these hashes so we can be sure to
	// always use the most current hash format version.

	if len(s.providerPlugins) == 0 {
		return nil
	}
	ret := make(map[addrs.Provider]getproviders.Hash, len(s.providerPlugins))
	for addr, cached := range s.providerPlugins {
		checksum, err := cached.Hash()
		if err != nil {
			// It'd be weird to get here because we should've been able to
			// launch all of these providers earlier anyway, and this isn't
			// an appropriate place to catch this error, so we'll just ignore
			// it. This does mean that if we _do_ get into this edge case then
			// we'll end up generating an un-applyable plan (because it'll lack
			// one of the needed checksums), which is unfortunate but okay
			// because of how unlikely we are to get here.
			continue
		}
		ret[addr] = checksum
	}
	return ret
}

func providerFactory(finder *Finder, selections *Selections, lock *depsfile.ProviderLock, cacheDir *providercache.Dir) (*providercache.CachedProvider, providers.Factory) {
	provider := lock.Provider()
	version := lock.Version()
	cached := cacheDir.ProviderVersion(provider, version)

	launchFactory := externalPluginLaunchFactory(cached, !finder.providerDisableAutoMTLS)

	return cached, func() (providers.Interface, error) {
		if cached == nil {
			return nil, fmt.Errorf(
				"there is no package for %s %s cached in %s",
				provider, version, cacheDir.BasePath(),
			)
		}

		// The selected provider must be one that matches the version
		// constraints in the configuration, and must be a provider that
		// the configuration or state actually refer to. The provider installer
		// is typically the one that catches disagreements between version
		// constraints and the lock file, but we can get here if someone
		// edits the lock file or configuration to create a disagreement and
		// then runs a command other than "terraform init", which therefore
		// isn't empowered to change the locked version selections.
		reqs := finder.providerRequirements
		if reqd, exists := reqs[provider]; exists {
			acceptable := versions.MeetingConstraints(reqd)
			if !acceptable.Has(version) {
				return nil, fmt.Errorf(
					"locked provider %s %s doesn't match configured version constraint %q",
					provider, version, getproviders.VersionConstraintsString(reqd),
				)
			}
		} else {
			// It would be very odd to get to this message, suggesting that
			// a Terraform Core caller passed a different configuration or
			// state to a graph walk operation than it previously used to
			// configure the finder.
			return nil, fmt.Errorf(
				"can't use provider %s which isn't required for the current configuration or state (this is a bug in Terraform)",
				provider,
			)
		}

		// The cached package must match one of the checksums recorded in
		// the lock file, if any.
		if allowedHashes := lock.PreferredHashes(); len(allowedHashes) != 0 {
			matched, err := cached.MatchesAnyHash(allowedHashes)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to verify checksum of %s %s package cached in in %s: %s",
					provider, version, cacheDir.BasePath(), err,
				)
			}
			if !matched {
				return nil, fmt.Errorf(
					"the cached package for %s %s (in %s) does not match any of the checksums recorded in the dependency lock file",
					provider, version, cacheDir.BasePath(),
				)
			}
		}

		// If we have forced exact checksums then we further require that
		// the package matches exactly what we recorded, and that we did
		// indeed record a checksum for this particular provider.
		//
		// Our error messages here assume that providerForceChecksums is
		// only used for saved plan files, which isn't really an assumption
		// this package's abstraction is supposed to make, but we can always
		// tweak these error messages without changing the API if the situation
		// changes in future.
		//
		// These are all "shouldn't really ever happen" cases, which indicate
		// either that there's a bug in Terraform, that someone tampered with a
		// saved plan file, or that someone tried to apply a saved plan file
		// in a different working directory than the one which created it.
		if finder.providerForceChecksums != nil {
			requiredHash, exists := finder.providerForceChecksums[provider]
			if !exists {
				return nil, fmt.Errorf(
					"unexpected use of provider %s which didn't participate in creating the plan",
					provider,
				)
			}
			matched, err := cached.MatchesHash(requiredHash)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to verify checksum of %s %s package cached in in %s: %s",
					provider, version, cacheDir.BasePath(), err,
				)
			}
			if !matched {
				return nil, fmt.Errorf(
					"the cached package for %s doesn't match the one which created the plan",
					provider,
				)
			}
		}

		// If all of the checks above succeeded then it's time to actually
		// try to launch the plugin's child process.
		return launchFactory()
	}
}

func devOverrideProviderFactory(provider addrs.Provider, localDir getproviders.PackageLocalDir, enableAutoMTLS bool) providers.Factory {
	// A dev override is essentially a synthetic cache entry for our purposes
	// here, so that's how we'll construct it. The providerFactory function
	// doesn't actually care about the version, so we can leave it
	// unspecified: overridden providers are not explicitly versioned.
	log.Printf("[DEBUG] Provider %s is overridden to load from %s", provider, localDir)
	return externalPluginLaunchFactory(&providercache.CachedProvider{
		Provider:   provider,
		Version:    getproviders.UnspecifiedVersion,
		PackageDir: string(localDir),
	}, enableAutoMTLS)
}

// externalPluginLaunchFactory produces a factory function that just launches
// the cached provider it's given, without doing any prior checks that it's
// a sensible cached plugin to be launching.
func externalPluginLaunchFactory(cached *providercache.CachedProvider, enableAutoMTLS bool) providers.Factory {
	return func() (providers.Interface, error) {
		execFile, err := cached.ExecutableFile()
		if err != nil {
			return nil, err
		}
		config := &plugin.ClientConfig{
			HandshakeConfig:  tfplugin.Handshake,
			Logger:           logging.NewProviderLogger(""),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			Managed:          true,
			Cmd:              exec.Command(execFile),
			AutoMTLS:         enableAutoMTLS,
			VersionedPlugins: tfplugin.VersionedPlugins,
			SyncStdout:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stdout", cached.Provider)),
			SyncStderr:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stderr", cached.Provider)),
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

// unmanagedProviderFactory produces a provider factory that uses the passed
// reattach information to connect to go-plugin processes that are already
// running, and implements providers.Interface against it.
func unmanagedProviderFactory(provider addrs.Provider, reattach *plugin.ReattachConfig, enableAutoMTLS bool) providers.Factory {
	return func() (providers.Interface, error) {
		config := &plugin.ClientConfig{
			HandshakeConfig:  tfplugin.Handshake,
			Logger:           logging.NewProviderLogger("unmanaged."),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			Managed:          false,
			Reattach:         reattach,
			AutoMTLS:         enableAutoMTLS,
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

func provisionerFactory(meta discovery.PluginMeta, enableAutoMTLS bool) provisioners.Factory {
	return func() (provisioners.Interface, error) {
		cfg := &plugin.ClientConfig{
			Cmd:              exec.Command(meta.Path),
			HandshakeConfig:  tfplugin.Handshake,
			VersionedPlugins: tfplugin.VersionedPlugins,
			Managed:          true,
			Logger:           logging.NewLogger("provisioner"),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			AutoMTLS:         enableAutoMTLS,
			SyncStdout:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stdout", meta.Name)),
			SyncStderr:       logging.PluginOutputMonitor(fmt.Sprintf("%s:stderr", meta.Name)),
		}
		client := plugin.NewClient(cfg)
		return newProvisionerClient(client)
	}
}

func newProvisionerClient(client *plugin.Client) (provisioners.Interface, error) {
	// Request the RPC client so we can get the provisioner
	// so we can build the actual RPC-implemented provisioner.
	rpcClient, err := client.Client()
	if err != nil {
		return nil, err
	}

	raw, err := rpcClient.Dispense(tfplugin.ProvisionerPluginName)
	if err != nil {
		return nil, err
	}

	// store the client so that the plugin can kill the child process
	p := raw.(*tfplugin.GRPCProvisioner)
	p.PluginClient = client
	return p, nil
}
