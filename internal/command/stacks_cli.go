// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

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

	"google.golang.org/grpc/metadata"

	"github.com/hashicorp/go-plugin"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/stackscliplugin"
	"github.com/hashicorp/terraform/internal/stackscliplugin/stackscliplugin1"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// StacksCLICommand is a Command implementation that interacts with Terraform
// Cloud for operations that relate to stacks. It delegates
// all execution to an internal plugin.
type StacksCLICommand struct {
	Meta
	// Path to the plugin server executable
	pluginBinary string
	// Service URL we can download plugin release binaries from
	pluginService *url.URL
	// Everything the plugin needs to build a client and Do Things
	pluginConfig StacksCLIPluginConfig
}

const (
	// DefaultStacksCLIVersion is the implied protocol version, though all
	// historical versions are defined explicitly.
	DefaultStacksCLIVersion = 1

	// // ExitRPCError is the exit code that is returned if an plugin
	// // communication error occurred.
	// ExitRPCError = 99

	// // ExitPluginError is the exit code that is returned if the plugin
	// // cannot be downloaded.
	// ExitPluginError = 98

	// // The regular HCP Terraform API service that the go-tfe client relies on.
	// tfeServiceID = "tfe.v2"
	// The stacks CLI release download service that the BinaryManager relies
	// on to fetch the plugin.
	stacksclipluginServiceID = "stackscliplugin.v1"
)

var (
	// StacksCLIHandshake is used to verify that the plugin is the appropriate plugin for
	// the client. This is not a security verification.
	StacksCLIHandshake = plugin.HandshakeConfig{
		MagicCookieKey:   "TF_STACKSCLIPLUGIN_MAGIC_COOKIE",
		MagicCookieValue: "123", // TODO: generate a value
		ProtocolVersion:  DefaultStacksCLIVersion,
	}
	// StacksCLIDataDir is the name of the directory within the data directory
	StacksCLIDataDir = "stackscliplugin"
)

func (c *StacksCLICommand) realRun(args []string, stdout, stderr io.Writer) int {
	args = c.Meta.process(args)
	fmt.Fprintf(stdout, "!!!terraform stacks cli command with args: %#v", args)

	diags := c.initPlugin()
	if diags.HasWarnings() || diags.HasErrors() {
		c.View.Diagnostics(diags)
	}
	if diags.HasErrors() {
		return ExitPluginError
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  StacksCLIHandshake,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Cmd:              exec.Command(c.pluginBinary),
		Logger:           logging.NewStacksCLILogger(),
		VersionedPlugins: map[int]plugin.PluginSet{
			1: {
				"stacks": &stackscliplugin1.GRPCStacksCLIPlugin{
					Metadata: c.pluginConfig.ToMetadata(),
				},
			},
		},
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		fmt.Fprintf(stderr, "Failed to create stacks CLI client: %s", err)
		return ExitRPCError
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("stacks")
	if err != nil {
		fmt.Fprintf(stderr, "Failed to request stacks CLI interface: %s", err)
		return ExitRPCError
	}

	// Proxy the request
	// Note: future changes will need to determine the type of raw when
	// multiple versions are possible.
	stacksCLI1, ok := raw.(stackscliplugin.StacksCLI1)
	if !ok {
		c.Ui.Error("If more than one stackscliplugin versions are available, they need to be added to the stacks cli command. This is a bug in Terraform.")
		return ExitRPCError
	}
	return stacksCLI1.Execute(args, stdout, stderr)
}

// discoverAndConfigure is an implementation detail of initPlugin. It fills in the
// pluginService and pluginConfig fields on a StacksCLICommand struct.
func (c *StacksCLICommand) discoverAndConfigure() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// stacks cli requires a cloud backend in order to work,
	// however `cloud` block in not yet allowed in the stacks working directory
	// initialize an empty cloud backend
	bf := backendInit.Backend("cloud")
	if bf == nil {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"`cloud` backend not found, this should not happen",
			"`cloud` backend is a valid backend type, yet it was not found, this is could be a bug, report it.",
		))
	}
	b := bf()
	cb, ok := b.(*cloud.Cloud)
	if !ok {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"`cloud` backend could not be initialized",
			"Could not initialize a `cloud` backend, this is could be a bug, report it.",
		))
		return diags
	}

	displayHostname := os.Getenv("TF_STACKS_CLI_HOSTNAME")
	if strings.TrimSpace(displayHostname) == "" {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"TF_STACKS_CLI_HOSTNAME is not set",
			"TF_STACKS_CLI_HOSTNAME must be set to the hostname of the HCP Terraform instance",
		))
	}

	token := os.Getenv("TF_STACKS_CLI_TOKEN")
	if strings.TrimSpace(token) == "" {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"TF_STACKS_CLI_TOKEN is not set",
			"TF_STACKS_CLI_TOKEN must be set to the token of the HCP Terraform instance",
		))
	}

	hostname, err := svchost.ForComparison(displayHostname)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Hostname string cannot be parsed into a svc.Hostname",
			err.Error(),
		))
	}

	host, err := cb.Services().Discover(hostname)
	if err != nil {
		// Network errors from Discover() can read like non-sequiters, so we wrap em.
		var serviceDiscoErr *disco.ErrServiceDiscoveryNetworkRequest
		if errors.As(err, &serviceDiscoErr) {
			err = fmt.Errorf("a network issue prevented cloud configuration; %w", err)
		}

		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Hostname discovery failed",
			err.Error(),
		))
	}

	// The discovery request worked, so cache the full results.
	cb.ServicesHost = host

	// re-use the cached service discovery info for this TFC
	// instance to find our plugin service and TFE API URLs:
	pluginService, err := cb.ServicesHost.ServiceURL(stacksclipluginServiceID)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Stacks CLI plugin service not found",
			err.Error(),
		))
	}
	c.pluginService = pluginService

	tfeService, err := cb.ServicesHost.ServiceURL(tfeServiceID)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"HCP Terraform API service not found",
			err.Error(),
		))
	}

	// Now just steal everything we need so we can pass it to the plugin later.
	c.pluginConfig = StacksCLIPluginConfig{
		Address:         tfeService.String(),
		BasePath:        tfeService.Path,
		DisplayHostname: displayHostname,
		Token:           token,
	}

	return diags
}

func (c *StacksCLICommand) initPlugin() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	var errorSummary = "Stacks CLI plugin initialization error"

	// Initialization can be aborted by interruption signals
	ctx, done := c.InterruptibleContext(c.CommandContext())
	defer done()

	// Discover service URLs, and build out the plugin config
	diags = diags.Append(c.discoverAndConfigure())
	if diags.HasErrors() {
		return diags
	}

	packagesPath, err := c.initPackagesCache()
	if err != nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, errorSummary, err.Error()))
	}

	overridePath := os.Getenv("TF_STACKS_CLI_PLUGIN_DEV_OVERRIDE")

	bm, err := stackscliplugin.NewStacksCLIBinaryManager(ctx, packagesPath, overridePath, c.pluginService, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, errorSummary, err.Error()))
	}

	version, err := bm.Resolve()
	if err != nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, "Stacks CLI plugin download error", err.Error()))
	}

	var cacheTraceMsg = ""
	if version.ResolvedFromCache {
		cacheTraceMsg = " (resolved from cache)"
	}
	if version.ResolvedFromDevOverride {
		cacheTraceMsg = " (resolved from dev override)"
		detailMsg := fmt.Sprintf("Instead of using the current released version, Terraform is loading the stacks CLI from the following location:\n\n - %s\n\nOverriding the stacks CLI location can cause unexpected behavior, and is only intended for use when developing new versions of the plugin.", version.Path)
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Stacks CLI plugin development overrides are in effect",
			detailMsg,
		))
	}
	log.Printf("[TRACE] plugin %q binary located at %q%s", version.ProductVersion, version.Path, cacheTraceMsg)
	c.pluginBinary = version.Path
	return diags
}

func (c *StacksCLICommand) initPackagesCache() (string, error) {
	packagesPath := path.Join(c.WorkingDir.DataDir(), StacksCLIDataDir)

	if info, err := os.Stat(packagesPath); err != nil || !info.IsDir() {
		log.Printf("[TRACE] initialized stackscliplugin cache directory at %q", packagesPath)
		err = os.MkdirAll(packagesPath, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to initialize stackscliplugin cache directory: %w", err)
		}
	} else {
		log.Printf("[TRACE] stackscliplugin cache directory found at %q", packagesPath)
	}

	return packagesPath, nil
}

// Run runs the stacks cli command with the given arguments.
func (c *StacksCLICommand) Run(args []string) int {
	args = c.Meta.process(args)
	return c.realRun(args, c.Meta.Streams.Stdout.File, c.Meta.Streams.Stderr.File)
}

// Help returns help text for the stacks cli command.
func (c *StacksCLICommand) Help() string {
	helpText := new(bytes.Buffer)
	if exitCode := c.realRun([]string{}, helpText, io.Discard); exitCode != 0 {
		return ""
	}

	return helpText.String()
}

// Synopsis returns a short summary of the stacks cli command.
func (c *StacksCLICommand) Synopsis() string {
	return "Manage HCP Terraform settings and metadata"
}

// StacksCLIPluginConfig is everything the plugin needs to know to configure a
// client and talk to HCP Terraform.
type StacksCLIPluginConfig struct {
	// Maybe someday we can use struct tags to automate grabbing these out of
	// the metadata headers! And verify client-side that we're sending the right
	// stuff, instead of having it all be a stringly-typed mystery ball! I want
	// to believe in that distant shining day! 🌻 Meantime, these struct tags
	// serve purely as docs.
	Address         string `md:"tfc-address"`
	BasePath        string `md:"tfc-base-path"`
	DisplayHostname string `md:"tfc-display-hostname"`
	Token           string `md:"tfc-token"`
	// TODO: how to read relevant env vars and pass it to the stacks-cli plugin
}

func (c StacksCLIPluginConfig) ToMetadata() metadata.MD {
	md := metadata.Pairs(
		"tfc-address", c.Address,
		"tfc-base-path", c.BasePath,
		"tfc-display-hostname", c.DisplayHostname,
		"tfc-token", c.Token,
	)
	return md
}
