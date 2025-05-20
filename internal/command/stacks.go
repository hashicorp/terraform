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

	"github.com/hashicorp/go-plugin"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
	backendInit "github.com/hashicorp/terraform/internal/backend/init"
	"github.com/hashicorp/terraform/internal/cloud"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/pluginshared"
	"github.com/hashicorp/terraform/internal/stacksplugin/stacksplugin1"

	"github.com/hashicorp/terraform/internal/tfdiags"
	"google.golang.org/grpc/metadata"
)

// StacksCommand is a Command implementation that interacts with Terraform
// Cloud for stack operations. It delegates all execution to an internal plugin.
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
	// DefaultStacksPluginVersion is the implied protocol version, though all
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
	// The stacks plugin release download service that the BinaryManager relies
	// on to fetch the plugin.
	stackspluginServiceID = "stacksplugin.v1"

	defaultHostname = "app.terraform.io"
)

var (
	// Handshake is used to verify that the plugin is the appropriate plugin for
	// the client. This is not a security verification.
	StacksHandshake = plugin.HandshakeConfig{
		MagicCookieKey:   "TF_STACKS_MAGIC_COOKIE",
		MagicCookieValue: "c9183f264a1db49ef2cbcc7b74f508a7bba9c3704c47cde3d130ae7f3b7a59c8f97a1e43d9e17ec0ac43a57fd250f373b2a8d991431d9fb1ea7bc48c8e7696fd",
		ProtocolVersion:  DefaultStacksPluginVersion,
	}
	// StacksPluginDataDir is the name of the directory within the data directory
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
		Logger:           logging.NewStacksLogger(),
		VersionedPlugins: map[int]plugin.PluginSet{
			1: {
				"stacks": &stacksplugin1.GRPCStacksPlugin{
					Metadata:   c.pluginConfig.ToMetadata(),
					Services:   c.Meta.Services,
					ShutdownCh: c.Meta.ShutdownCh,
				},
			},
		},
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		fmt.Fprintf(stderr, "Failed to create stacks plugin client: %s", err)
		return ExitRPCError
	}

	// Request the plugin
	raw, err := rpcClient.Dispense("stacks")
	if err != nil {
		fmt.Fprintf(stderr, "Failed to request stacks plugin interface: %s", err)
		return ExitRPCError
	}

	// Proxy the request
	// Note: future changes will need to determine the type of raw when
	// multiple versions are possible.
	stacks1, ok := raw.(pluginshared.CustomPluginClient)
	if !ok {
		c.Ui.Error("If more than one stacksplugin versions are available, they need to be added to the stacks command. This is a bug in Terraform.")
		return ExitRPCError
	}

	return stacks1.Execute(args, stdout, stderr)
}

// discoverAndConfigure is an implementation detail of initPlugin. It fills in the
// pluginService and pluginConfig fields on a StacksCommand struct.
func (c *StacksCommand) discoverAndConfigure() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// using the current terraform path for the plugin binary path
	tfBinaryPath, err := os.Executable()
	if err != nil {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Terraform binary path not found",
			"Terraform binary path not found: "+err.Error(),
		))
	}

	// stacks plugin requires a cloud backend in order to work,
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

	displayHostname := os.Getenv("TF_STACKS_HOSTNAME")
	if strings.TrimSpace(displayHostname) == "" {
		log.Printf("[TRACE] stacksplugin hostname not set, falling back to %q", defaultHostname)
		displayHostname = defaultHostname
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

	token := os.Getenv("TF_STACKS_TOKEN")
	if strings.TrimSpace(token) == "" {
		// attempt to read from the credentials file
		token, err = cloud.CliConfigToken(hostname, cb.Services())
		if err != nil {
			// some commands like stacks init and validate could be run without a token so allow it without errors
			diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Could not read token from credentials file, proceeding without a token",
				err.Error(),
			))
		}
	}

	// re-use the cached service discovery info for this TFC
	// instance to find our plugin service and TFE API URLs:
	pluginService, err := cb.ServicesHost.ServiceURL(stackspluginServiceID)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Stacks plugin service not found",
			err.Error(),
		))
	}
	c.pluginService = pluginService

	tfeService, err := cb.ServicesHost.ServiceURL(tfeStacksServiceID)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"HCP Terraform API service not found",
			err.Error(),
		))
	}

	// optional env values
	orgName := os.Getenv("TF_STACKS_ORGANIZATION_NAME")
	projectName := os.Getenv("TF_STACKS_PROJECT_NAME")
	stackName := os.Getenv("TF_STACKS_STACK_NAME")

	// config to be passed to the plugin later.
	c.pluginConfig = StacksPluginConfig{
		Address:             tfeService.String(),
		BasePath:            tfeService.Path,
		DisplayHostname:     displayHostname,
		Token:               token,
		TerraformBinaryPath: tfBinaryPath,
		OrganizationName:    orgName,
		ProjectName:         projectName,
		StackName:           stackName,
		TerminalWidth:       c.Meta.Streams.Stdout.Columns(),
	}

	return diags
}

func (c *StacksCommand) initPlugin() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	var errorSummary = "Stacks plugin initialization error"

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

	bm, err := pluginshared.NewStacksBinaryManager(ctx, packagesPath, overridePath, c.pluginService, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, errorSummary, err.Error()))
	}

	version, err := bm.Resolve()
	if err != nil {
		return diags.Append(tfdiags.Sourceless(tfdiags.Error, "Stacks plugin download error", err.Error()))
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
		log.Printf("[TRACE] initialized stacksplugin cache directory at %q", packagesPath)
		err = os.MkdirAll(packagesPath, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to initialize stacksplugin cache directory: %w", err)
		}
	} else {
		log.Printf("[TRACE] stacksplugin cache directory found at %q", packagesPath)
	}

	return packagesPath, nil
}

// Run runs the stacks command with the given arguments.
func (c *StacksCommand) Run(args []string) int {
	args = c.Meta.process(args)
	return c.realRun(args, c.Meta.Streams.Stdout.File, c.Meta.Streams.Stderr.File)
}

// Help returns help text for the stacks command.
func (c *StacksCommand) Help() string {
	helpText := new(bytes.Buffer)
	if exitCode := c.realRun([]string{}, helpText, io.Discard); exitCode != 0 {
		return ""
	}

	return helpText.String()
}

// Synopsis returns a short summary of the stacks command.
func (c *StacksCommand) Synopsis() string {
	return "Manage HCP Terraform stack operations"
}

// StacksPluginConfig is everything the stacks plugin needs to know to configure a
// client and talk to HCP Terraform.
type StacksPluginConfig struct {
	Address             string `md:"tfc-address"`
	BasePath            string `md:"tfc-base-path"`
	DisplayHostname     string `md:"tfc-display-hostname"`
	Token               string `md:"tfc-token"`
	TerraformBinaryPath string `md:"terraform-binary-path"`
	OrganizationName    string `md:"tfc-organization"`
	ProjectName         string `md:"tfc-project"`
	StackName           string `md:"tfc-stack"`
	TerminalWidth       int    `md:"terminal-width"`
}

func (c StacksPluginConfig) ToMetadata() metadata.MD {
	md := metadata.Pairs(
		"tfc-address", c.Address,
		"tfc-base-path", c.BasePath,
		"tfc-display-hostname", c.DisplayHostname,
		"tfc-token", c.Token,
		"terraform-binary-path", c.TerraformBinaryPath,
		"tfc-organization", c.OrganizationName,
		"tfc-project", c.ProjectName,
		"tfc-stack", c.StackName,
		"terminal-width", fmt.Sprintf("%d", c.TerminalWidth),
	)
	return md
}
