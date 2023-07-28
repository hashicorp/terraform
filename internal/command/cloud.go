package command

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/hashicorp/go-plugin"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/cloudplugin"
	"github.com/hashicorp/terraform/internal/cloudplugin/cloudplugin1"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/mitchellh/cli"
)

// CloudCommand is a Command implementation that interacts with Terraform
// Cloud for operations that are inherently planless. It delegates
// all execution to an internal plugin.
type CloudCommand struct {
	Meta
	pluginBinary string
}

const (
	// DefaultCloudPluginVersion is the implied protocol version, though all
	// historical versions are defined explicitly.
	DefaultCloudPluginVersion = 1

	// ExitRPCError is the exit code that is returned if an plugin
	// communication error occurred.
	ExitRPCError = 99

	// ExitPluginError is the exit code that is returned if the plugin
	// cannot be downloaded.
	ExitPluginError = 98
)

var (
	// Handshake is used to verify that the plugin is the appropriate plugin for
	// the client. This is not a security verification.
	Handshake = plugin.HandshakeConfig{
		MagicCookieKey:   "TF_CLOUDPLUGIN_MAGIC_COOKIE",
		MagicCookieValue: "721fca41431b780ff3ad2623838faaa178d74c65e1cfdfe19537c31656496bf9f82d6c6707f71d81c8eed0db9043f79e56ab4582d013bc08ead14f57961461dc",
		ProtocolVersion:  DefaultCloudPluginVersion,
	}
	// CloudPluginDataDir is the name of the directory within the data directory
	CloudPluginDataDir = "cloudplugin"
)

func (c *CloudCommand) realRun(args []string, stdout, stderr io.Writer) int {
	args = c.Meta.process(args)

	diags := c.initPlugin()
	if diags.HasErrors() {
		c.Ui.Warn(diags.ErrWithWarnings().Error())
		return ExitPluginError
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  Handshake,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Cmd:              exec.Command(c.pluginBinary),
		Logger:           logging.NewCloudLogger(),
		VersionedPlugins: map[int]plugin.PluginSet{
			1: {
				"cloud": &cloudplugin1.GRPCCloudPlugin{},
			},
		},
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		fmt.Fprintf(stderr, "Failed to create cloud plugin client: %s", err)
		return ExitRPCError
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("cloud")
	if err != nil {
		fmt.Fprintf(stderr, "Failed to request cloud plugin interface: %s", err)
		return ExitRPCError
	}

	// Proxy the request
	// Note: future changes will need to determine the type of raw when
	// multiple versions are possible.
	cloud1, ok := raw.(cloudplugin.Cloud1)
	if !ok {
		c.Ui.Error("If more than one cloudplugin versions are available, they need to be added to the cloud command. This is a bug in Terraform.")
		return ExitRPCError
	}
	return cloud1.Execute(args, stdout, stderr)
}

// discover the TFC/E API service URL and version constraints.
func (c *CloudCommand) discover(hostname string) (*url.URL, error) {
	hn, err := svchost.ForComparison(hostname)
	if err != nil {
		return nil, err
	}

	host, err := c.Services.Discover(hn)
	if err != nil {
		var serviceDiscoErr *disco.ErrServiceDiscoveryNetworkRequest

		switch {
		case errors.As(err, &serviceDiscoErr):
			err = fmt.Errorf("a network issue prevented cloud configuration; %w", err)
			return nil, err
		default:
			return nil, err
		}
	}

	service, err := host.ServiceURL("cloudplugin.v1")
	// Return the error, unless its a disco.ErrVersionNotSupported error.
	if _, ok := err.(*disco.ErrVersionNotSupported); !ok && err != nil {
		return nil, err
	}

	return service, err
}

func (c *CloudCommand) hostnameFromConfig() (string, error) {
	var diags tfdiags.Diagnostics

	backendConfig, backendDiags := c.loadBackendConfig(".")
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
		return "", diags.Err()
	}

	b, backendDiags := c.Backend(&BackendOpts{
		Config: backendConfig,
	})
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		return "", diags.Err()
	}

	cloudBackend, ok := b.(*cloud.Cloud)
	if !ok {
		return "", fmt.Errorf("cloud command requires that a cloud block be configured in the working directory")
	}

	return cloudBackend.Hostname, nil
}

func (c *CloudCommand) hostnameFromEnv() string {
	return os.Getenv("TF_CLOUD_HOSTNAME")
}

func (c *CloudCommand) initPlugin() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	var errorSummary = "Cloud plugin initialization error"

	// Initialization can be aborted by interruption signals
	ctx, done := c.InterruptibleContext(c.CommandContext())
	defer done()

	var hostname string
	if hostname = c.hostnameFromEnv(); hostname == "" {
		var err error
		hostname, err = c.hostnameFromConfig()
		if err != nil {
			return diags.Append(tfdiags.Sourceless(tfdiags.Error, errorSummary, err.Error()))
		}
	}

	serviceURL, err := c.discover(hostname)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, errorSummary, err.Error()))
	}

	packagesPath, err := c.initPackagesCache()
	if err != nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, errorSummary, err.Error()))
	}

	vm, err := cloudplugin.NewVersionManager(ctx, packagesPath, serviceURL, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, errorSummary, err.Error()))
	}

	version, err := vm.Resolve()
	if err != nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, "Cloud plugin download error", err.Error()))
	}

	var cacheTraceMsg = ""
	if version.ResolvedFromCache {
		cacheTraceMsg = " (resolved from cache)"
	}
	log.Printf("[TRACE] plugin %q binary located at %q%s", version.ProductVersion, version.BinaryLocation, cacheTraceMsg)
	c.pluginBinary = version.BinaryLocation
	return nil
}

func (c *CloudCommand) initPackagesCache() (string, error) {
	packagesPath := path.Join(c.WorkingDir.DataDir(), CloudPluginDataDir)

	if info, err := os.Stat(packagesPath); err != nil || !info.IsDir() {
		log.Printf("[TRACE] initialized cloudplugin cache directory at %q", packagesPath)
		err = os.MkdirAll(packagesPath, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to initialize cloudplugin cache directory: %w", err)
		}
	} else {
		log.Printf("[TRACE] cloudplugin cache directory found at %q", packagesPath)
	}

	return packagesPath, nil
}

type proxyOutput struct {
	io.Writer
	upstream cli.Ui
}

func (p proxyOutput) Write(data []byte) (int, error) {
	p.upstream.Output(strings.TrimSuffix(string(data), "\n"))
	return len(data), nil
}

type proxyError struct {
	io.Writer
	upstream cli.Ui
}

func (p proxyError) Write(data []byte) (int, error) {
	p.upstream.Error(strings.TrimSuffix(string(data), "\n"))
	return len(data), nil
}

// Run runs the cloud command with the given arguments.
func (c *CloudCommand) Run(args []string) int {
	args = c.Meta.process(args)
	return c.realRun(args, proxyOutput{upstream: c.Meta.Ui}, proxyError{upstream: c.Meta.Ui})
}

// Help returns help text for the cloud command.
func (c *CloudCommand) Help() string {
	helpText := new(bytes.Buffer)
	if exitCode := c.realRun([]string{}, helpText, io.Discard); exitCode != 0 {
		return ""
	}

	return helpText.String()
}

// Synopsis returns a short summary of the cloud command.
func (c *CloudCommand) Synopsis() string {
	return "Manage Terraform Cloud settings and metadata"
}
