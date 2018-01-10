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
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

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
	providerPlugins := c.providerPluginSet()
	pluginsLockFile := c.providerPluginsLock()
	pluginsLock := pluginsLockFile.Read()
	var pluginVersions []string
	for meta := range providerPlugins {
		name := meta.Name
		wantHash, wanted := pluginsLock[name]
		if !wanted {
			// Ignore providers that aren't used by the current config at all
			continue
		}
		gotHash, err := meta.SHA256()
		if err != nil {
			// if we can't read the file to hash it, ignore it.
			continue
		}
		if !bytes.Equal(gotHash, wantHash) {
			// Not the plugin we've locked, so ignore it.
			continue
		}

		// If we get here then we've found a selected plugin, so we'll print
		// out its details.
		if meta.Version == "0.0.0" {
			pluginVersions = append(pluginVersions, fmt.Sprintf("+ provider.%s (unversioned)", name))
		} else {
			pluginVersions = append(pluginVersions, fmt.Sprintf("+ provider.%s v%s", name, meta.Version))
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
		// Separate the prior output with a newline
		c.Ui.Output("")

		// Check the latest version
		info, err := c.CheckFunc()
		if err != nil {
			c.Ui.Error(fmt.Sprintf(
				"Error checking latest version: %s", err))
		}
		if info.Outdated {
			c.Ui.Output(fmt.Sprintf(
				"Your version of Terraform is out of date! The latest version\n"+
					"is %s. You can update by downloading from www.terraform.io/downloads.html",
				info.Latest))
		}
	}

	return 0
}

func (c *VersionCommand) Synopsis() string {
	return "Prints the Terraform version"
}
