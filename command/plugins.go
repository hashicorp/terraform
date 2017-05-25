package command

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	plugin "github.com/hashicorp/go-plugin"
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
}

func choosePlugins(avail discovery.PluginMetaSet, reqd discovery.PluginRequirements) map[string]discovery.PluginMeta {
	candidates := avail.ConstrainVersions(reqd)
	ret := map[string]discovery.PluginMeta{}
	for name, metas := range candidates {
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

	chosen := choosePlugins(r.Available, reqd)
	for name := range reqd {
		if newest, available := chosen[name]; available {
			digest, err := newest.SHA256()
			if err != nil {
				errs = append(errs, fmt.Errorf("provider.%s: failed to load plugin to verify its signature: %s", name, err))
				continue
			}
			if !reqd[name].AcceptsSHA256(digest) {
				// This generic error message is intended to avoid troubling
				// users with implementation details. The main useful point
				// here is that they need to run "terraform init" to
				// fix this, which is covered by the UI code reporting these
				// error messages.
				errs = append(errs, fmt.Errorf("provider.%s: installed but not yet initialized", name))
				continue
			}

			client := tfplugin.Client(newest)
			factories[name] = providerFactory(client)
		} else {
			errs = append(errs, fmt.Errorf("provider.%s: no suitable version installed", name))
		}
	}

	return factories, errs
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
func (m *Meta) pluginDirs() []string {

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

	dirs = append(dirs, m.pluginDir())
	dirs = append(dirs, m.GlobalPluginDirs...)
	return dirs
}

// providerPluginSet returns the set of valid providers that were discovered in
// the defined search paths.
func (m *Meta) providerPluginSet() discovery.PluginMetaSet {
	plugins := discovery.FindPlugins("provider", m.pluginDirs())
	plugins, _ = plugins.ValidateVersions()

	return plugins
}

func (m *Meta) providerResolver() terraform.ResourceProviderResolver {
	return &multiVersionProviderResolver{
		Available: m.providerPluginSet(),
	}
}

// filter the requirements returning only the providers that we can't resolve
func (m *Meta) missingPlugins(avail discovery.PluginMetaSet, reqd discovery.PluginRequirements) discovery.PluginRequirements {
	missing := make(discovery.PluginRequirements)

	candidates := avail.ConstrainVersions(reqd)

	for name, versionSet := range reqd {
		if metas := candidates[name]; metas == nil {
			missing[name] = versionSet
		}
	}

	return missing
}

func (m *Meta) provisionerFactories() map[string]terraform.ResourceProvisionerFactory {
	dirs := m.pluginDirs()
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
