package command

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	plugin "github.com/hashicorp/go-plugin"
	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/terraform"
)

// multiVersionProviderResolver is an implementation of
// terraform.ResourceProviderResolver that matches the given version constraints
// against a set of versioned provider plugins to find the newest version of
// each that satisfies the given constraints.
type multiVersionProviderResolver struct {
	Available discovery.PluginMetaSet
}

func (r *multiVersionProviderResolver) ResolveProviders(
	reqd discovery.PluginRequirements,
) (map[string]terraform.ResourceProviderFactory, []error) {
	factories := make(map[string]terraform.ResourceProviderFactory, len(reqd))
	var errs []error

	candidates := r.Available.ConstrainVersions(reqd)
	for name := range reqd {
		if metas := candidates[name]; metas != nil {
			newest := metas.Newest()
			client := tfplugin.Client(newest)
			factories[name] = providerFactory(client)
		} else {
			errs = append(errs, fmt.Errorf("provider.%s: no suitable version installed", name))
		}
	}

	return factories, errs
}

// providerPluginSet returns the set of valid providers that were discovered in
// the defined search paths.
func (m *Meta) providerPluginSet() discovery.PluginMetaSet {
	var dirs []string

	// When searching the following directories, earlier entries get precedence
	// if the same plugin version is found twice, but newer versions will
	// always get preference below regardless of where they are coming from.
	// TODO: Add auto-install dir, default vendor dir and optional override
	// vendor dir(s).
	dirs = append(dirs, ".")
	dirs = append(dirs, m.GlobalPluginDirs...)

	plugins := discovery.FindPlugins("provider", dirs)
	plugins, _ = plugins.ValidateVersions()

	return plugins
}

func (m *Meta) providerResolver() terraform.ResourceProviderResolver {
	return &multiVersionProviderResolver{
		Available: m.providerPluginSet(),
	}
}

// filter the requirements returning only the providers that we can't resolve
func (m *Meta) missingProviders(reqd discovery.PluginRequirements) discovery.PluginRequirements {
	missing := make(discovery.PluginRequirements)

	candidates := m.providerPluginSet().ConstrainVersions(reqd)

	for name, versionSet := range reqd {
		if metas := candidates[name]; metas == nil {
			missing[name] = versionSet
		}
	}

	return missing
}

func (m *Meta) provisionerFactories() map[string]terraform.ResourceProvisionerFactory {
	var dirs []string

	// When searching the following directories, earlier entries get precedence
	// if the same plugin version is found twice, but newer versions will
	// always get preference below regardless of where they are coming from.
	//
	// NOTE: Currently we don't use versioning for provisioners, so the
	// version handling here is just the minimum required to be able to use
	// the plugin discovery package. All provisioner plugins should always
	// be versionless, which we treat as version 0.0.0 here.
	dirs = append(dirs, ".")
	dirs = append(dirs, m.GlobalPluginDirs...)

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
