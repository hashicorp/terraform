package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/kardianos/osext"

	"github.com/hashicorp/terraform/internal/logging"
	tfplugin "github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/terraform"
)

// NOTE WELL: The logic in this file is primarily about plugin types OTHER THAN
// providers, which use an older set of approaches implemented here.
//
// The provider-related functions live primarily in meta_providers.go, and
// lean on some different underlying mechanisms in order to support automatic
// installation and a hierarchical addressing namespace, neither of which
// are supported for other plugin types.

// store the user-supplied path for plugin discovery
func (m *Meta) storePluginPath(pluginPath []string) error {
	if len(pluginPath) == 0 {
		return nil
	}

	path := filepath.Join(m.DataDir(), PluginPathFile)

	// remove the plugin dir record if the path was set to an empty string
	if len(pluginPath) == 1 && (pluginPath[0] == "") {
		err := os.Remove(path)
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	js, err := json.MarshalIndent(pluginPath, "", "  ")
	if err != nil {
		return err
	}

	// if this fails, so will WriteFile
	os.MkdirAll(m.DataDir(), 0755)

	return ioutil.WriteFile(path, js, 0644)
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

func (m *Meta) provisionerFactories() map[string]terraform.ProvisionerFactory {
	dirs := m.pluginDirs(true)
	plugins := discovery.FindPlugins("provisioner", dirs)
	plugins, _ = plugins.ValidateVersions()

	// For now our goal is to just find the latest version of each plugin
	// we have on the system. All provisioners should be at version 0.0.0
	// currently, so there should actually only be one instance of each plugin
	// name here, even though the discovery interface forces us to pretend
	// that might not be true.

	factories := make(map[string]terraform.ProvisionerFactory)

	// Wire up the internal provisioners first. These might be overridden
	// by discovered provisioners below.
	for name := range InternalProvisioners {
		factories[name] = internalProvisionerFactory(discovery.PluginMeta{Name: name})
	}

	byName := plugins.ByName()
	for name, metas := range byName {
		// Since we validated versions above and we partitioned the sets
		// by name, we're guaranteed that the metas in our set all have
		// valid versions and that there's at least one meta.
		newest := metas.Newest()

		factories[name] = provisionerFactory(newest)
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
		Cmd:              exec.Command(cmdArgv[0], cmdArgv[1:]...),
		HandshakeConfig:  tfplugin.Handshake,
		Managed:          true,
		VersionedPlugins: tfplugin.VersionedPlugins,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		AutoMTLS:         enableProviderAutoMTLS,
		Logger:           logging.NewProviderLogger(),
	}

	return plugin.NewClient(cfg), nil
}

func provisionerFactory(meta discovery.PluginMeta) terraform.ProvisionerFactory {
	return func() (provisioners.Interface, error) {
		cfg := &plugin.ClientConfig{
			Cmd:              exec.Command(meta.Path),
			HandshakeConfig:  tfplugin.Handshake,
			VersionedPlugins: tfplugin.VersionedPlugins,
			Managed:          true,
			Logger:           logging.NewHCLogger("provisioner"),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			AutoMTLS:         enableProviderAutoMTLS,
		}
		client := plugin.NewClient(cfg)
		return newProvisionerClient(client)
	}
}

func internalProvisionerFactory(meta discovery.PluginMeta) terraform.ProvisionerFactory {
	return func() (provisioners.Interface, error) {
		client, err := internalPluginClient("provisioner", meta.Name)
		if err != nil {
			return nil, fmt.Errorf("[WARN] failed to build command line for internal plugin %q: %s", meta.Name, err)
		}
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
