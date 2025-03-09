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
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/cloudplugin"
	"github.com/hashicorp/terraform/internal/cloudplugin/cloudplugin1"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// CloudCommand is a Command implementation that interacts with Terraform
// Cloud for operations that are inherently planless. It delegates
// all execution to an internal plugin.
type CloudCommand struct {
	Meta
	// Path to the plugin server executable
	pluginBinary string
	// Service URL we can download plugin release binaries from
	pluginService *url.URL
	// Everything the plugin needs to build a client and Do Things
	pluginConfig CloudPluginConfig
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

	// The regular HCP Terraform API service that the go-tfe client relies on.
	tfeServiceID = "tfe.v2"
	// The cloud plugin release download service that the BinaryManager relies
	// on to fetch the plugin.
	cloudpluginServiceID = "cloudplugin.v1"
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
	if diags.HasWarnings() || diags.HasErrors() {
		c.View.Diagnostics(diags)
	}
	if diags.HasErrors() {
		return ExitPluginError
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  Handshake,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Cmd:              exec.Command(c.pluginBinary),
		Logger:           logging.NewCloudLogger(),
		VersionedPlugins: map[int]plugin.PluginSet{
			1: {
				"cloud": &cloudplugin1.GRPCCloudPlugin{
					Metadata: c.pluginConfig.ToMetadata(),
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

// discoverAndConfigure is an implementation detail of initPlugin. It fills in the
// pluginService and pluginConfig fields on a CloudCommand struct.
func (c *CloudCommand) discoverAndConfigure() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// First, spin up a Cloud backend. (Why? bc finding the info the plugin
	// needs is hard, and the Cloud backend already knows how to do it all.)
	backendConfig, bConfigDiags := c.loadBackendConfig(".")
	diags = diags.Append(bConfigDiags)
	if diags.HasErrors() {
		return diags
	}
	b, backendDiags := c.Backend(&BackendOpts{
		Config: backendConfig,
	})
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
		return diags
	}
	cb, ok := b.(*cloud.Cloud)
	if !ok {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No `cloud` block found",
			"Cloud command requires that a `cloud` block be configured in the working directory",
		))
		return diags
	}

	// Ok sweet. First, re-use the cached service discovery info for this TFC
	// instance to find our plugin service and TFE API URLs:
	pluginService, err := cb.ServicesHost.ServiceURL(cloudpluginServiceID)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cloud plugin service not found",
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

	currentWorkspace, err := c.Workspace()
	if err != nil {
		// The only possible error here is "you set TF_WORKSPACE badly"
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Bad current workspace",
			err.Error(),
		))
	}

	// Now just steal everything we need so we can pass it to the plugin later.
	c.pluginConfig = CloudPluginConfig{
		Address:            tfeService.String(),
		BasePath:           tfeService.Path,
		DisplayHostname:    cb.Hostname,
		Token:              cb.Token,
		Organization:       cb.Organization,
		CurrentWorkspace:   currentWorkspace,
		WorkspaceName:      cb.WorkspaceMapping.Name,
		WorkspaceTags:      cb.WorkspaceMapping.TagsAsSet,
		DefaultProjectName: cb.WorkspaceMapping.Project,
	}

	return diags
}

func (c *CloudCommand) initPlugin() tfdiags.Diagnostics {
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

	overridePath := os.Getenv("TF_CLOUD_PLUGIN_DEV_OVERRIDE")

	bm, err := cloudplugin.NewBinaryManager(ctx, packagesPath, overridePath, c.pluginService, runtime.GOOS, runtime.GOARCH)
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
		detailMsg := fmt.Sprintf("Instead of using the current released version, Terraform is loading the cloud plugin from the following location:\n\n - %s\n\nOverriding the cloud plugin location can cause unexpected behavior, and is only intended for use when developing new versions of the plugin.", version.Path)
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Cloud plugin development overrides are in effect",
			detailMsg,
		))
	}
	log.Printf("[TRACE] plugin %q binary located at %q%s", version.ProductVersion, version.Path, cacheTraceMsg)
	c.pluginBinary = version.Path
	return diags
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

// Run runs the cloud command with the given arguments.
func (c *CloudCommand) Run(args []string) int {
	args = c.Meta.process(args)
	return c.realRun(args, c.Meta.Streams.Stdout.File, c.Meta.Streams.Stderr.File)
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
	return "Manage HCP Terraform settings and metadata"
}

// CloudPluginConfig is everything the cloud plugin needs to know to configure a
// client and talk to HCP Terraform.
type CloudPluginConfig struct {
	// Maybe someday we can use struct tags to automate grabbing these out of
	// the metadata headers! And verify client-side that we're sending the right
	// stuff, instead of having it all be a stringly-typed mystery ball! I want
	// to believe in that distant shining day! 🌻 Meantime, these struct tags
	// serve purely as docs.
	Address         string `md:"tfc-address"`
	BasePath        string `md:"tfc-base-path"`
	DisplayHostname string `md:"tfc-display-hostname"`
	Token           string `md:"tfc-token"`
	Organization    string `md:"tfc-organization"`
	// The actual selected workspace
	CurrentWorkspace string `md:"tfc-current-workspace"`

	// The raw "WorkspaceMapping" attributes, which determine the workspaces
	// that could be selected. Generally you want CurrentWorkspace instead, but
	// these can potentially be useful for niche use cases.
	WorkspaceName      string   `md:"tfc-workspace-name"`
	WorkspaceTags      []string `md:"tfc-workspace-tags"`
	DefaultProjectName string   `md:"tfc-default-project-name"`
}

func (c CloudPluginConfig) ToMetadata() metadata.MD {
	// First, do everything except tags the easy way
	md := metadata.Pairs(
		"tfc-address", c.Address,
		"tfc-base-path", c.BasePath,
		"tfc-display-hostname", c.DisplayHostname,
		"tfc-token", c.Token,
		"tfc-organization", c.Organization,
		"tfc-current-workspace", c.CurrentWorkspace,
		"tfc-workspace-name", c.WorkspaceName,
		"tfc-default-project-name", c.DefaultProjectName,
	)
	// Then the straggler
	md["tfc-workspace-tags"] = c.WorkspaceTags
	return md
}
