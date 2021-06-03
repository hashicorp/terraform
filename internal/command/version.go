package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// VersionCommand is a Command implementation prints the version.
type VersionCommand struct {
	Meta

	Version           string
	VersionPrerelease string
	CheckFunc         VersionCheckFunc
	Platform          getproviders.Platform
}

type VersionOutput struct {
	Version            string            `json:"terraform_version"`
	Platform           string            `json:"platform"`
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
Usage: terraform [global options] version [options]

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
	}

	// We'll also attempt to print out the selected plugin versions. We do
	// this based on the dependency lock file, and so the result might be
	// empty or incomplete if the user hasn't successfully run "terraform init"
	// since the most recent change to dependencies.
	//
	// Generally-speaking this is a best-effort thing that will give us a good
	// result in the usual case where the user successfully ran "terraform init"
	// and then hit a problem running _another_ command.
	var providerVersions []string
	var providerLocks map[addrs.Provider]*depsfile.ProviderLock
	if locks, err := c.lockedDependencies(); err == nil {
		providerLocks = locks.AllProviders()
		for providerAddr, lock := range providerLocks {
			version := lock.Version().String()
			if version == "0.0.0" {
				providerVersions = append(providerVersions, fmt.Sprintf("+ provider %s (unversioned)", providerAddr))
			} else {
				providerVersions = append(providerVersions, fmt.Sprintf("+ provider %s v%s", providerAddr, version))
			}
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
		for providerAddr, lock := range providerLocks {
			version := lock.Version().String()
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
			Platform:           c.Platform.String(),
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
		c.Ui.Output(fmt.Sprintf("on %s", c.Platform))

		if len(providerVersions) != 0 {
			sort.Strings(providerVersions)
			for _, str := range providerVersions {
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
	return "Show the current Terraform version"
}
