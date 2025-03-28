// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"

	"google.golang.org/grpc/metadata"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/stacksplugin"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksplugin1"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// CloudCommand is a Command implementation that interacts with Terraform
// Cloud for operations that are inherently planless. It delegates
// all execution to an internal plugin.
type StacksCommand struct {
	Meta
	// Path to the plugin server executable
	pluginBinary string
	// Service URL we can download plugin release binaries from
	pluginService *url.URL
	// Everything the plugin needs to build a client and Do Things
	pluginConfig StacksPluginConfig
}

const (
	// DefaultCloudPluginVersion is the implied protocol version, though all
	// historical versions are defined explicitly.
	DefaultStacksPluginVersion = 1

	// ExitRPCError is the exit code that is returned if an plugin
	// communication error occurred.
	ExitStacksRPCError = 99

	// ExitPluginError is the exit code that is returned if the plugin
	// cannot be downloaded.
	ExitStacksPluginError = 98

	// The regular HCP Terraform API service that the go-tfe client relies on.
	tfeStacksServiceID = "tfe.v2"
	// The cloud plugin release download service that the BinaryManager relies
	// on to fetch the plugin.
	stackspluginServiceID = "stacksplugin.v1"
)

var (
	// Handshake is used to verify that the plugin is the appropriate plugin for
	// the client. This is not a security verification.
	StacksHandshake = plugin.HandshakeConfig{
		MagicCookieKey:   "TF_STACKSPLUGIN_MAGIC_COOKIE",
		MagicCookieValue: "721fca41431b780ff3ad2623838faaa178d74c65e1cfdfe19537c31656496bf9f82d6c6707f71d81c8eed0db9043f79e56ab4582d013bc08ead14f57961461dc",
		ProtocolVersion:  DefaultStacksPluginVersion,
	}
	// CloudPluginDataDir is the name of the directory within the data directory
	StacksPluginDataDir = "stacksplugin"
)

func (c *StacksCommand) realRun(args []string, stdout, stderr io.Writer) int {
	args = c.Meta.process(args)

	diags := c.initPlugin()
	if diags.HasWarnings() || diags.HasErrors() {
		c.View.Diagnostics(diags)
	}
	if diags.HasErrors() {
		return ExitPluginError
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  StacksHandshake,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Cmd:              exec.Command(c.pluginBinary),
		Logger:           logging.NewCloudLogger(),
		VersionedPlugins: map[int]plugin.PluginSet{
			1: {
				"stacks": &stacksplugin1.GRPCStacksPlugin{
					Metadata: c.pluginConfig.ToMetadata(),
					Services: c.Meta.Services,
				},
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
	raw, err := rpcClient.Dispense("stacks")
	if err != nil {
		fmt.Fprintf(stderr, "Failed to request cloud plugin interface: %s", err)
		return ExitRPCError
	}

	// Proxy the request
	// Note: future changes will need to determine the type of raw when
	// multiple versions are possible.
	stacks1, ok := raw.(stacksplugin.Stacks1)
	if !ok {
		c.Ui.Error("If more than one cloudplugin versions are available, they need to be added to the cloud command. This is a bug in Terraform.")
		return ExitRPCError
	}

	return stacks1.Execute(args, stdout, stderr)
}

// discoverAndConfigure is an implementation detail of initPlugin. It fills in the
// pluginService and pluginConfig fields on a CloudCommand struct.
func (c *StacksCommand) discoverAndConfigure() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	tfBinaryPath, err := os.Executable()
	if err != nil {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Terraform binary path not found",
			"Terraform binary path not found: "+err.Error(),
		))
	}
	// just a dummy value to avoid nil pointer dereference, otherwise are just testing the dev plugin override
	c.pluginService = &url.URL{
		Scheme: "https",
		Host:   "api.stacks.hashicorp.com",
	}

	// Now just steal everything we need so we can pass it to the plugin later.
	c.pluginConfig = StacksPluginConfig{
		TerraformBinaryPath: tfBinaryPath,
	}

	return diags
}

func (c *StacksCommand) initPlugin() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	var errorSummary = "Cloud plugin initialization error"

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

	overridePath := os.Getenv("TF_STACKS_PLUGIN_DEV_OVERRIDE")

	bm, err := stacksplugin.NewBinaryManager(ctx, packagesPath, overridePath, c.pluginService, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, errorSummary, err.Error()))
	}

	version, err := bm.Resolve()
	if err != nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, "Cloud plugin download error", err.Error()))
	}

	var cacheTraceMsg = ""
	if version.ResolvedFromCache {
		cacheTraceMsg = " (resolved from cache)"
	}
	if version.ResolvedFromDevOverride {
		cacheTraceMsg = " (resolved from dev override)"
		detailMsg := fmt.Sprintf("Instead of using the current released version, Terraform is loading the stacks plugin from the following location:\n\n - %s\n\nOverriding the stacks plugin location can cause unexpected behavior, and is only intended for use when developing new versions of the plugin.", version.Path)
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Stacks plugin development overrides are in effect",
			detailMsg,
		))
	}
	log.Printf("[TRACE] plugin %q binary located at %q%s", version.ProductVersion, version.Path, cacheTraceMsg)
	c.pluginBinary = version.Path
	return diags
}

func (c *StacksCommand) initPackagesCache() (string, error) {
	packagesPath := path.Join(c.WorkingDir.DataDir(), StacksPluginDataDir)

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

// Run runs the cloud command with the given arguments.
func (c *StacksCommand) Run(args []string) int {
	args = c.Meta.process(args)
	return c.realRun(args, c.Meta.Streams.Stdout.File, c.Meta.Streams.Stderr.File)
}

// Help returns help text for the cloud command.
func (c *StacksCommand) Help() string {
	helpText := new(bytes.Buffer)
	if exitCode := c.realRun([]string{}, helpText, io.Discard); exitCode != 0 {
		return ""
	}

	return helpText.String()
}

// Synopsis returns a short summary of the cloud command.
func (c *StacksCommand) Synopsis() string {
	return "Manage HCP Terraform settings and metadata"
}

// CloudPluginConfig is everything the cloud plugin needs to know to configure a
// client and talk to HCP Terraform.
type StacksPluginConfig struct {
	// Maybe someday we can use struct tags to automate grabbing these out of
	// the metadata headers! And verify client-side that we're sending the right
	// stuff, instead of having it all be a stringly-typed mystery ball! I want
	// to believe in that distant shining day! 🌻 Meantime, these struct tags
	// serve purely as docs.
	Address             string `md:"tfc-address"`
	BasePath            string `md:"tfc-base-path"`
	DisplayHostname     string `md:"tfc-display-hostname"`
	Token               string `md:"tfc-token"`
	Organization        string `md:"tfc-organization"`
	TerraformBinaryPath string `md:"terraform-binary-path"`
}

func (c StacksPluginConfig) ToMetadata() metadata.MD {
	// First, do everything except tags the easy way
	md := metadata.Pairs(
		"tfc-address", c.Address,
		"tfc-base-path", c.BasePath,
		"tfc-display-hostname", c.DisplayHostname,
		"tfc-token", c.Token,
		"tfc-organization", c.Organization,
		"terraform-binary-path", c.TerraformBinaryPath,
	)
	return md
}
