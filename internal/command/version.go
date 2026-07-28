// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// VersionCommand is a Command implementation prints the version.
type VersionCommand struct {
	Meta

	Version           string
	VersionPrerelease string
	CheckFunc         VersionCheckFunc
	Platform          getproviders.Platform
}

// VersionCheckFunc is the callback called by the Version command to
// check if there is a new version of Terraform.
type VersionCheckFunc func() (VersionCheckInfo, error)

// VersionCheckInfo is the return value for the VersionCheckFunc callback
// and tells the Version command information about the latest version
// of Terraform.
type VersionCheckInfo struct {
	Outdated bool
	Latest   string
	Alerts   []string
}

func (c *VersionCommand) Help() string {
	helpText := `
Usage: terraform [global options] version [options]

  Displays the version of Terraform and all installed plugins

Options:

  -json       Output the version information as a JSON object.
`
	return strings.TrimSpace(helpText)
}

func (c *VersionCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Parse command-specific arguments.
	args, argDiags := arguments.ParseVersion(rawArgs)
	diags = diags.Append(argDiags)

	// Prepare the view
	view := views.NewVersion(args.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Collect version information
	var version string
	if c.VersionPrerelease != "" {
		version = fmt.Sprintf("%s-%s", c.Version, c.VersionPrerelease)
	} else {
		version = c.Version
	}
	platform := c.Platform.String()

	// We attempt to print out the selected plugin versions. We do
	// this based on the dependency lock file, and so the result might be
	// empty or incomplete if the user hasn't successfully run "terraform init"
	// since the most recent change to dependencies.
	//
	// Generally-speaking this is a best-effort thing that will give us a good
	// result in the usual case where the user successfully ran "terraform init"
	// and then hit a problem running _another_ command.
	var providerLocks map[addrs.Provider]*depsfile.ProviderLock
	if locks, err := c.lockedDependencies(); err == nil {
		providerLocks = locks.AllProviders()
	}

	// If we have a version check function, then let's check for
	// the latest version as well.
	var latest string
	var outdated bool
	if c.CheckFunc != nil {
		// Check the latest version
		info, err := c.CheckFunc()
		if err != nil {
			diags = diags.Append(fmt.Errorf(
				"\nError checking latest version: %s", err))
		}
		if info.Outdated {
			latest = info.Latest
			outdated = true
		}
	}

	// Format and print output
	view.LogVersion(version, platform, providerLocks, outdated, latest, diags)

	return 0
}

func (c *VersionCommand) Synopsis() string {
	return "Show the current Terraform version"
}
