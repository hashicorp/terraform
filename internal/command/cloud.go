// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/cloudplugin"
	"github.com/hashicorp/terraform/internal/cloudplugin/cloudplugin1"
	"github.com/hashicorp/terraform/internal/logging"
)

// CloudCommand is a Command implementation that interacts with Terraform
// Cloud for operations that are inherently planless. It delegates
// all execution to an internal plugin.
type CloudCommand struct {
	Meta
}

const (
	// DefaultCloudPluginVersion is the implied protocol version, though all
	// historical versions are defined explicitly.
	DefaultCloudPluginVersion = 1

	// ExitRPCError is the exit code that is returned if an plugin
	// communication error occurred.
	ExitRPCError = 99
)

var (
	// Handshake is used to verify that the plugin is the appropriate plugin for
	// the client. This is not a security verification.
	Handshake = plugin.HandshakeConfig{
		MagicCookieKey:   "TF_CLOUDPLUGIN_MAGIC_COOKIE",
		MagicCookieValue: "721fca41431b780ff3ad2623838faaa178d74c65e1cfdfe19537c31656496bf9f82d6c6707f71d81c8eed0db9043f79e56ab4582d013bc08ead14f57961461dc",
		ProtocolVersion:  DefaultCloudPluginVersion,
	}
)

func (c *CloudCommand) proxy(args []string, stdout, stderr io.Writer) int {
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  Handshake,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Cmd:              exec.Command("./terraform-cloudplugin"),
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

// Run runs the cloud command with the given arguments.
func (c *CloudCommand) Run(args []string) int {
	args = c.Meta.process(args)

	// TODO: Download and verify the signing of the terraform-cloudplugin
	// release that is appropriate for this OS/Arch
	if _, err := os.Stat("./terraform-cloudplugin"); err != nil {
		c.Ui.Warn("terraform-cloudplugin not found. This plugin does not have an official release yet.")
		return 1
	}

	// TODO: Need to use some type of c.Meta handle here
	return c.proxy(args, os.Stdout, os.Stderr)
}

// Help returns help text for the cloud command.
func (c *CloudCommand) Help() string {
	helpText := new(bytes.Buffer)
	c.proxy([]string{}, helpText, io.Discard)

	return helpText.String()
}

// Synopsis returns a short summary of the cloud command.
func (c *CloudCommand) Synopsis() string {
	return "Manage Terraform Cloud settings and metadata"
}
