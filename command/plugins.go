package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	plugin "github.com/hashicorp/go-plugin"
	terraformProvider "github.com/hashicorp/terraform/builtin/providers/terraform"
	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/terraform"
	"github.com/kardianos/osext"
)

// multiVersionProviderResolver is an implementation of
// terraform.ResourceProviderResolver that matches the given version constraints
// against a set of versioned provider plugins to find the newest version of
// each that satisfies the given constraints.
type multiVersionProviderResolver struct {
	Available discovery.PluginMetaSet

	// Internal is a map that overrides the usual plugin selection process
	// for internal plugins. These plugins do not support version constraints
	// (will produce an error if one is set). This should be used only in
	// exceptional circumstances since it forces the provider's release
	// schedule to be tied to that of Terraform Core.
	Internal map[string]terraform.ResourceProviderFactory
}

func choosePlugins(avail discovery.PluginMetaSet, internal map[string]terraform.ResourceProviderFactory, reqd discovery.PluginRequirements) map[string]discovery.PluginMeta {
	candidates := avail.ConstrainVersions(reqd)
	ret := map[string]discovery.PluginMeta{}
	for name, metas := range candidates {
		// If the provider is in our internal map then we ignore any
		// discovered plugins for it since these are dealt with separately.
		if _, isInternal := internal[name]; isInternal {
			continue
		}

		if len(metas) == 0 {
			continue
		}
		ret[name] = metas.Newest()
	}
	return ret
}

func (r *multiVersionProviderResolver) ResolveProviders(
	reqd discovery.PluginRequirements,
) (map[string]terraform.ResourceProviderFactory, []error) {
	factories := make(map[string]terraform.ResourceProviderFactory, len(reqd))
	var errs []error

	chosen := choosePlugins(r.Available, r.Internal, reqd)
	for name, req := range reqd {
		if factory, isInternal := r.Internal[name]; isInternal {
			if !req.Versions.Unconstrained() {
				errs = append(errs, fmt.Errorf("provider.%s: this provider is built in to Terraform and so it does not support version constraints", name))
				continue
			}
			factories[name] = factory
			continue
		}

		if newest, available := chosen[name]; available {
			digest, err := newest.SHA256()
			if err != nil {
				errs = append(errs, fmt.Errorf("provider.%s: failed to load plugin to verify its signature: %s", name, err))
				continue
			}
			if !reqd[name].AcceptsSHA256(digest) {
				errs = append(errs, fmt.Errorf("provider.%s: new or changed plugin executable", name))
				continue
			}

			client := tfplugin.Client(newest)
			factories[name] = providerFactory(client)
		} else {
			msg := fmt.Sprintf("provider.%s: no suitable version installed", name)

			required := req.Versions.String()
			// no version is unconstrained
			if required == "" {
				required = "(any version)"
			}

			foundVersions := []string{}
			for meta := range r.Available.WithName(name) {
				foundVersions = append(foundVersions, fmt.Sprintf("%q", meta.Version))
			}

			found := "none"
			if len(foundVersions) > 0 {
				found = strings.Join(foundVersions, ", ")
			}

			msg += fmt.Sprintf("\n  version requirements: %q\n  versions installed: %s", required, found)

			errs = append(errs, errors.New(msg))
		}
	}

	return factories, errs
}

// store the user-supplied path for plugin discovery
func (m *Meta) storePluginPath(pluginPath []string) error {
	if len(pluginPath) == 0 {
		return nil
	}

	js, err := json.MarshalIndent(pluginPath, "", "  ")
	if err != nil {
		return err
	}

	// if this fails, so will WriteFile
	os.MkdirAll(m.DataDir(), 0755)

	return ioutil.WriteFile(filepath.Join(m.DataDir(), PluginPathFile), js, 0644)
}

// Load the user-defined plugin search path into Meta.pluginPath if the file
// exists.
func (m *Meta) loadPluginPath() ([]string, error) {
	js, err := ioutil.ReadFile(filepath.Join(m.DataDir(), PluginPathFile))
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	var pluginPath []string
	if err := json.Unmarshal(js, &pluginPath); err != nil {
		return nil, err
	}

	return pluginPath, nil
}

// the default location for automatically installed plugins
func (m *Meta) pluginDir() string {
	return filepath.Join(m.DataDir(), "plugins", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
}

// pluginDirs return a list of directories to search for plugins.
//
// Earlier entries in this slice get priority over later when multiple copies
// of the same plugin version are found, but newer versions always override
// older versions where both satisfy the provider version constraints.
func (m *Meta) pluginDirs(includeAutoInstalled bool) []string {
	// user defined paths take precedence
	if len(m.pluginPath) > 0 {
		return m.pluginPath
	}

	// When searching the following directories, earlier entries get precedence
	// if the same plugin version is found twice, but newer versions will
	// always get preference below regardless of where they are coming from.
	// TODO: Add auto-install dir, default vendor dir and optional override
	// vendor dir(s).
	dirs := []string{"."}

	// Look in the same directory as the Terraform executable.
	// If found, this replaces what we found in the config path.
	exePath, err := osext.Executable()
	if err != nil {
		log.Printf("[ERROR] Error discovering exe directory: %s", err)
	} else {
		dirs = append(dirs, filepath.Dir(exePath))
	}

	// add the user vendor directory
	dirs = append(dirs, DefaultPluginVendorDir)

	if includeAutoInstalled {
		dirs = append(dirs, m.pluginDir())
	}
	dirs = append(dirs, m.GlobalPluginDirs...)

	return dirs
}

func (m *Meta) pluginCache() discovery.PluginCache {
	dir := m.PluginCacheDir
	if dir == "" {
		return nil // cache disabled
	}

	dir = filepath.Join(dir, pluginMachineName)

	return discovery.NewLocalPluginCache(dir)
}

// providerPluginSet returns the set of valid providers that were discovered in
// the defined search paths.
func (m *Meta) providerPluginSet() discovery.PluginMetaSet {
	plugins := discovery.FindPlugins("provider", m.pluginDirs(true))

	// Add providers defined in the legacy .terraformrc,
	if m.PluginOverrides != nil {
		plugins = plugins.OverridePaths(m.PluginOverrides.Providers)
	}

	plugins, _ = plugins.ValidateVersions()

	for p := range plugins {
		log.Printf("[DEBUG] found valid plugin: %q", p.Name)
	}

	return plugins
}

// providerPluginAutoInstalledSet returns the set of providers that exist
// within the auto-install directory.
func (m *Meta) providerPluginAutoInstalledSet() discovery.PluginMetaSet {
	plugins := discovery.FindPlugins("provider", []string{m.pluginDir()})
	plugins, _ = plugins.ValidateVersions()

	for p := range plugins {
		log.Printf("[DEBUG] found valid plugin: %q", p.Name)
	}

	return plugins
}

// providerPluginManuallyInstalledSet returns the set of providers that exist
// in all locations *except* the auto-install directory.
func (m *Meta) providerPluginManuallyInstalledSet() discovery.PluginMetaSet {
	plugins := discovery.FindPlugins("provider", m.pluginDirs(false))

	// Add providers defined in the legacy .terraformrc,
	if m.PluginOverrides != nil {
		plugins = plugins.OverridePaths(m.PluginOverrides.Providers)
	}

	plugins, _ = plugins.ValidateVersions()

	for p := range plugins {
		log.Printf("[DEBUG] found valid plugin: %q", p.Name)
	}

	return plugins
}

func (m *Meta) providerResolver() terraform.ResourceProviderResolver {
	return &multiVersionProviderResolver{
		Available: m.providerPluginSet(),
		Internal:  m.internalProviders(),
	}
}

func (m *Meta) internalProviders() map[string]terraform.ResourceProviderFactory {
	return map[string]terraform.ResourceProviderFactory{
		"terraform": func() (terraform.ResourceProvider, error) {
			return terraformProvider.Provider(), nil
		},
	}
}

// filter the requirements returning only the providers that we can't resolve
func (m *Meta) missingPlugins(avail discovery.PluginMetaSet, reqd discovery.PluginRequirements) discovery.PluginRequirements {
	missing := make(discovery.PluginRequirements)

	candidates := avail.ConstrainVersions(reqd)
	internal := m.internalProviders()

	for name, versionSet := range reqd {
		// internal providers can't be missing
		if _, ok := internal[name]; ok {
			continue
		}

		log.Printf("[DEBUG] plugin requirements: %q=%q", name, versionSet.Versions)
		if metas := candidates[name]; metas.Count() == 0 {
			missing[name] = versionSet
		}
	}

	return missing
}

func (m *Meta) provisionerFactories() map[string]terraform.ResourceProvisionerFactory {
	dirs := m.pluginDirs(true)
	plugins := discovery.FindPlugins("provisioner", dirs)
	plugins, _ = plugins.ValidateVersions()

	// For now our goal is to just find the latest version of each plugin
	// we have on the system. All provisioners should be at version 0.0.0
	// currently, so there should actually only be one instance of each plugin
	// name here, even though the discovery interface forces us to pretend
	// that might not be true.

	factories := make(map[string]terraform.ResourceProvisionerFactory)

	// Wire up the internal provisioners first. These might be overridden
	// by discovered provisioners below.
	for name := range InternalProvisioners {
		client, err := internalPluginClient("provisioner", name)
		if err != nil {
			log.Printf("[WARN] failed to build command line for internal plugin %q: %s", name, err)
			continue
		}
		factories[name] = provisionerFactory(client)
	}

	byName := plugins.ByName()
	for name, metas := range byName {
		// Since we validated versions above and we partitioned the sets
		// by name, we're guaranteed that the metas in our set all have
		// valid versions and that there's at least one meta.
		newest := metas.Newest()
		client := tfplugin.Client(newest)
		factories[name] = provisionerFactory(client)
	}

	return factories
}

func internalPluginClient(kind, name string) (*plugin.Client, error) {
	cmdLine, err := BuildPluginCommandString(kind, name)
	if err != nil {
		return nil, err
	}

	// See the docstring for BuildPluginCommandString for why we need to do
	// this split here.
	cmdArgv := strings.Split(cmdLine, TFSPACE)

	cfg := &plugin.ClientConfig{
		Cmd:             exec.Command(cmdArgv[0], cmdArgv[1:]...),
		HandshakeConfig: tfplugin.Handshake,
		Managed:         true,
		Plugins:         tfplugin.PluginMap,
	}

	return plugin.NewClient(cfg), nil
}

func providerFactory(client *plugin.Client) terraform.ResourceProviderFactory {
	return func() (terraform.ResourceProvider, error) {
		// Request the RPC client so we can get the provider
		// so we can build the actual RPC-implemented provider.
		rpcClient, err := client.Client()
		if err != nil {
			return nil, err
		}

		raw, err := rpcClient.Dispense(tfplugin.ProviderPluginName)
		if err != nil {
			return nil, err
		}

		return raw.(terraform.ResourceProvider), nil
	}
}

func provisionerFactory(client *plugin.Client) terraform.ResourceProvisionerFactory {
	return func() (terraform.ResourceProvisioner, error) {
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

		return raw.(terraform.ResourceProvisioner), nil
	}
}
