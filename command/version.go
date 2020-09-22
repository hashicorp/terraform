package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// VersionCommand is a Command implementation prints the version.
type VersionCommand struct {
	Meta

	Revision          string
	Version           string
	VersionPrerelease string
	CheckFunc         VersionCheckFunc
}

type VersionOutput struct {
	Version            string            `json:"terraform_version"`
	Revision           string            `json:"terraform_revision"`
	ProviderSelections map[string]string `json:"provider_selections"`
	Outdated           bool              `json:"terraform_outdated"`
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
Usage: terraform version [options]

  Displays the version of Terraform and all installed plugins

Options:

  -json       Output the version information as a JSON object.
`
	return strings.TrimSpace(helpText)
}

func (c *VersionCommand) Run(args []string) int {
	var outdated bool
	var latest string
	var versionString bytes.Buffer
	args = c.Meta.process(args)
	var jsonOutput bool
	cmdFlags := c.Meta.defaultFlagSet("version")
	cmdFlags.BoolVar(&jsonOutput, "json", false, "json")
	// Enable but ignore the global version flags. In main.go, if any of the
	// arguments are -v, -version, or --version, this command will be called
	// with the rest of the arguments, so we need to be able to cope with
	// those.
	cmdFlags.Bool("v", true, "version")
	cmdFlags.Bool("version", true, "version")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	fmt.Fprintf(&versionString, "Terraform v%s", c.Version)
	if c.VersionPrerelease != "" {
		fmt.Fprintf(&versionString, "-%s", c.VersionPrerelease)

		if c.Revision != "" {
			fmt.Fprintf(&versionString, " (%s)", c.Revision)
		}
	}

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
	// If we have a version check function, then let's check for
	// the latest version as well.
	if c.CheckFunc != nil {
		// Check the latest version
		info, err := c.CheckFunc()
		if err != nil && !jsonOutput {
			c.Ui.Error(fmt.Sprintf(
				"\nError checking latest version: %s", err))
		}
		if info.Outdated {
			outdated = true
			latest = info.Latest
		}
	}

	if jsonOutput {
		selectionsOutput := make(map[string]string)
		for providerAddr, cached := range providerSelections {
			version := cached.Version.String()
			selectionsOutput[providerAddr.String()] = version
		}

		var versionOutput string
		if c.VersionPrerelease != "" {
			versionOutput = c.Version + "-" + c.VersionPrerelease
		} else {
			versionOutput = c.Version
		}

		output := VersionOutput{
			Version:            versionOutput,
			Revision:           c.Revision,
			ProviderSelections: selectionsOutput,
			Outdated:           outdated,
		}

		jsonOutput, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			c.Ui.Error(fmt.Sprintf("\nError marshalling JSON: %s", err))
			return 1
		}
		c.Ui.Output(string(jsonOutput))
		return 0
	} else {
		c.Ui.Output(versionString.String())
		if len(pluginVersions) != 0 {
			sort.Strings(pluginVersions)
			for _, str := range pluginVersions {
				c.Ui.Output(str)
			}
		}
		if outdated {
			c.Ui.Output(fmt.Sprintf(
				"\nYour version of Terraform is out of date! The latest version\n"+
					"is %s. You can update by downloading from https://www.terraform.io/downloads.html",
				latest))
		}

	}

	return 0
}

func (c *VersionCommand) Synopsis() string {
	return "Prints the Terraform version"
}
