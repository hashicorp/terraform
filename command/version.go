package command

import (
	"bytes"
	"fmt"
	"sort"
)

// VersionCommand is a Command implementation prints the version.
type VersionCommand struct {
	Meta

	Revision          string
	Version           string
	VersionPrerelease string
	CheckFunc         VersionCheckFunc
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
	return ""
}

func (c *VersionCommand) Run(args []string) int {
	var versionString bytes.Buffer
	args = c.Meta.process(args)
	fmt.Fprintf(&versionString, "Terraform v%s", c.Version)
	if c.VersionPrerelease != "" {
		fmt.Fprintf(&versionString, "-%s", c.VersionPrerelease)

		if c.Revision != "" {
			fmt.Fprintf(&versionString, " (%s)", c.Revision)
		}
	}

	c.Ui.Output(versionString.String())

	// We'll also attempt to print out the selected plugin versions. We can
	// do this only if "terraform init" was already run and thus we've committed
	// to a specific set of plugins. If not, the plugins lock will be empty
	// and so we'll show _no_ providers.
	//
	// Generally-speaking this is a best-effort thing that will give us a good
	// result in the usual case where the user successfully ran "terraform init"
	// and then hit a problem running _another_ command.
	providerInstaller := c.providerInstaller()
	providerSelections, err := providerInstaller.SelectedPackages()
	var pluginVersions []string
	if err != nil {
		// we'll just ignore it and show no plugins at all, then.
		providerSelections = nil
	}
	for providerAddr, cached := range providerSelections {
		version := cached.Version.String()
		if version == "0.0.0" {
			pluginVersions = append(pluginVersions, fmt.Sprintf("+ provider %s (unversioned)", providerAddr))
		} else {
			pluginVersions = append(pluginVersions, fmt.Sprintf("+ provider %s v%s", providerAddr, version))
		}
	}
	if len(pluginVersions) != 0 {
		sort.Strings(pluginVersions)
		for _, str := range pluginVersions {
			c.Ui.Output(str)
		}
	}

	// If we have a version check function, then let's check for
	// the latest version as well.
	if c.CheckFunc != nil {

		// Check the latest version
		info, err := c.CheckFunc()
		if err != nil {
			c.Ui.Error(fmt.Sprintf(
				"\nError checking latest version: %s", err))
		}
		if info.Outdated {
			c.Ui.Output(fmt.Sprintf(
				"\nYour version of Terraform is out of date! The latest version\n"+
					"is %s. You can update by downloading from https://www.terraform.io/downloads.html",
				info.Latest))
		}
	}

	return 0
}

func (c *VersionCommand) Synopsis() string {
	return "Prints the Terraform version"
}
