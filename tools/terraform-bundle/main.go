// terraform-bundle is a tool to create "bundle archives" that contain both
// a particular version of Terraform and a set of providers for use with it.
//
// Such bundles are useful for distributing a Terraform version and a set
// of providers to a system out-of-band, in situations where Terraform's
// auto-installer cannot be used due to firewall rules, "air-gapped" systems,
// etc.
//
// When using bundle archives, it's suggested to use a version numbering
// scheme that adds a suffix that identifies the archive as being a bundle,
// to make it easier to distinguish bundle archives from the normal separated
// release archives. This tool by default produces files with the following
// naming scheme:
//
//    terraform_0.10.0-bundle2017070302_linux_amd64.zip
//
// The user is free to rename these files, since the archive filename has
// no significance to Terraform itself and the generated pseudo-version number
// is not referenced within the archive contents.
//
// If using such a bundle with an on-premises Terraform Enterprise installation,
// it's recommended to use the generated version number (or a modification
// thereof) as the tool version within Terraform Enterprise, so that
// bundle archives can be distinguished from official releases and from
// each other even if the same core Terraform version is used.
//
// Terraform providers in general release more often than core, so it is
// intended that this tool can be used to periodically upgrade providers
// within certain constraints and produce a new bundle containing these
// upgraded provider versions. A bundle archive can include multiple versions
// of the same provider, allowing configurations containing provider version
// constrants to be gradually migrated to newer versions.
package main

import (
	"io/ioutil"
	"log"
	"os"

	tfversion "github.com/hashicorp/terraform/version"
	"github.com/mitchellh/cli"
)

func main() {
	ui := &cli.ColoredUi{
		OutputColor: cli.UiColorNone,
		InfoColor:   cli.UiColorNone,
		ErrorColor:  cli.UiColorRed,
		WarnColor:   cli.UiColorYellow,

		Ui: &cli.BasicUi{
			Reader:      os.Stdin,
			Writer:      os.Stdout,
			ErrorWriter: os.Stderr,
		},
	}

	// Terraform's code tends to produce noisy logs, since Terraform itself
	// suppresses them by default. To avoid polluting our console, we'll do
	// the same.
	if os.Getenv("TF_LOG") == "" {
		log.SetOutput(ioutil.Discard)
	}

	c := cli.NewCLI("terraform-bundle", tfversion.Version)
	c.Args = os.Args[1:]
	c.Commands = map[string]cli.CommandFactory{
		"package": func() (cli.Command, error) {
			return &PackageCommand{
				ui: ui,
			}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		ui.Error(err.Error())
	}

	os.Exit(exitStatus)
}
