package main

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"flag"

	"io"

	getter "github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/mitchellh/cli"
)

type PackageCommand struct {
	ui cli.Ui
}

func (c *PackageCommand) Run(args []string) int {
	flags := flag.NewFlagSet("package", flag.ExitOnError)
	osPtr := flags.String("os", "", "Target operating system")
	archPtr := flags.String("arch", "", "Target CPU architecture")
	err := flags.Parse(args)
	if err != nil {
		c.ui.Error(err.Error())
		return 1
	}

	osName := runtime.GOOS
	archName := runtime.GOARCH
	if *osPtr != "" {
		osName = *osPtr
	}
	if *archPtr != "" {
		archName = *archPtr
	}

	if flags.NArg() != 1 {
		c.ui.Error("Configuration filename is required")
		return 1
	}
	configFn := flags.Arg(0)

	config, err := LoadConfigFile(configFn)
	if err != nil {
		c.ui.Error(fmt.Sprintf("Failed to read config: %s", err))
		return 1
	}

	if discovery.ConstraintStr("< 0.10.0-beta1").MustParse().Allows(config.Terraform.Version.MustParse()) {
		c.ui.Error("Bundles can be created only for Terraform 0.10 or newer")
		return 1
	}

	workDir, err := ioutil.TempDir("", "terraform-bundle")
	if err != nil {
		c.ui.Error(fmt.Sprintf("Could not create temporary dir: %s", err))
		return 1
	}
	defer os.RemoveAll(workDir)

	c.ui.Info(fmt.Sprintf("Fetching Terraform %s core package...", config.Terraform.Version))

	coreZipURL := c.coreURL(config.Terraform.Version, osName, archName)
	err = getter.Get(workDir, coreZipURL)
	if err != nil {
		c.ui.Error(fmt.Sprintf("Failed to fetch core package from %s: %s", coreZipURL, err))
	}

	installer := &discovery.ProviderInstaller{
		Dir: workDir,

		// FIXME: This is incorrect because it uses the protocol version of
		// this tool, rather than of the Terraform binary we just downloaded.
		// But we can't get this information from a Terraform binary, so
		// we'll just ignore this for now as we only have one protocol version
		// in play anyway. If a new protocol version shows up later we will
		// probably deal with this by just matching version ranges and
		// hard-coding the knowledge of which Terraform version uses which
		// protocol version.
		PluginProtocolVersion: plugin.Handshake.ProtocolVersion,

		OS:   osName,
		Arch: archName,
		Ui:   c.ui,
	}

	if len(config.Providers) > 0 {
		c.ui.Output(fmt.Sprintf("Checking for available provider plugins on %s...",
			discovery.GetReleaseHost()))
	}

	for name, constraints := range config.Providers {
		for _, constraint := range constraints {
			c.ui.Output(fmt.Sprintf("- Resolving %q provider (%s)...",
				name, constraint))
			_, err := installer.Get(name, constraint.MustParse())
			if err != nil {
				c.ui.Error(fmt.Sprintf("- Failed to resolve %s provider %s: %s", name, constraint, err))
				return 1
			}
		}
	}

	files, err := ioutil.ReadDir(workDir)
	if err != nil {
		c.ui.Error(fmt.Sprintf("Failed to read work directory %s: %s", workDir, err))
		return 1
	}

	// If we get this far then our workDir now contains the union of the
	// contents of all the zip files we downloaded above. We can now create
	// our output file.
	outFn := c.bundleFilename(config.Terraform.Version, time.Now(), osName, archName)
	c.ui.Info(fmt.Sprintf("Creating %s ...", outFn))
	outF, err := os.OpenFile(outFn, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		c.ui.Error(fmt.Sprintf("Failed to create %s: %s", outFn, err))
		return 1
	}
	outZ := zip.NewWriter(outF)
	defer func() {
		err := outZ.Close()
		if err != nil {
			c.ui.Error(fmt.Sprintf("Failed to close %s: %s", outFn, err))
			os.Exit(1)
		}
		err = outF.Close()
		if err != nil {
			c.ui.Error(fmt.Sprintf("Failed to close %s: %s", outFn, err))
			os.Exit(1)
		}
	}()

	for _, file := range files {
		if file.IsDir() {
			// should never happen unless something tampers with our tmpdir
			continue
		}

		fn := filepath.Join(workDir, file.Name())
		r, err := os.Open(fn)
		if err != nil {
			c.ui.Error(fmt.Sprintf("Failed to open %s: %s", fn, err))
			return 1
		}
		hdr, err := zip.FileInfoHeader(file)
		if err != nil {
			c.ui.Error(fmt.Sprintf("Failed to add zip entry for %s: %s", fn, err))
			return 1
		}
		w, err := outZ.CreateHeader(hdr)
		if err != nil {
			c.ui.Error(fmt.Sprintf("Failed to add zip entry for %s: %s", fn, err))
			return 1
		}
		_, err = io.Copy(w, r)
		if err != nil {
			c.ui.Error(fmt.Sprintf("Failed to write %s to bundle: %s", fn, err))
			return 1
		}
	}

	c.ui.Info("All done!")

	return 0
}

func (c *PackageCommand) bundleFilename(version discovery.VersionStr, time time.Time, osName, archName string) string {
	time = time.UTC()
	return fmt.Sprintf(
		"terraform_%s-bundle%04d%02d%02d%02d_%s_%s.zip",
		version,
		time.Year(), time.Month(), time.Day(), time.Hour(),
		osName, archName,
	)
}

func (c *PackageCommand) coreURL(version discovery.VersionStr, osName, archName string) string {
	return fmt.Sprintf(
		"%s/terraform/%s/terraform_%s_%s_%s.zip",
		discovery.GetReleaseHost(), version, version, osName, archName,
	)
}

func (c *PackageCommand) Synopsis() string {
	return "Produces a bundle archive"
}

func (c *PackageCommand) Help() string {
	return `Usage: terraform-bundle package [options] <config-file>

Uses the given bundle configuration file to produce a zip file in the
current working directory containing a Terraform binary along with zero or
more provider plugin binaries.

Options:
  -os=name    Target operating system the archive will be built for. Defaults
              to that of the system where the command is being run.

  -arch=name  Target CPU architecture the archive will be built for. Defaults
              to that of the system where the command is being run.

The resulting zip file can be used to more easily install Terraform and
a fixed set of providers together on a server, so that Terraform's provider
auto-installation mechanism can be avoided.

To build an archive for Terraform Enterprise, use:
  -os=linux -arch=amd64

Note that the given configuration file is a format specific to this command,
not a normal Terraform configuration file. The file format looks like this:

  terraform {
    # Version of Terraform to include in the bundle. An exact version number
	# is required.
    version = "0.10.0"
  }

  # Define which provider plugins are to be included
  providers {
    # Include the newest "aws" provider version in the 1.0 series.
    aws = ["~> 1.0"]

    # Include both the newest 1.0 and 2.0 versions of the "google" provider.
    # Each item in these lists allows a distinct version to be added. If the
	# two expressions match different versions then _both_ are included in
	# the bundle archive.
    google = ["~> 1.0", "~> 2.0"]
  }

`
}
