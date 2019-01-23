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
	discovery "github.com/hashicorp/terraform/plugin/discovery"
	"github.com/mitchellh/cli"
)

var releaseHost = "https://releases.hashicorp.com"

type PackageCommand struct {
	ui cli.Ui
}

// shameless stackoverflow copy + pasta https://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	os.Chmod(dst, sfi.Mode())
	return
}

// see above
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

func (c *PackageCommand) Run(args []string) int {
	flags := flag.NewFlagSet("package", flag.ExitOnError)
	osPtr := flags.String("os", "", "Target operating system")
	archPtr := flags.String("arch", "", "Target CPU architecture")
	pluginDirPtr := flags.String("plugin-dir", "", "Path to custom plugins directory")
	err := flags.Parse(args)
	if err != nil {
		c.ui.Error(err.Error())
		return 1
	}

	osName := runtime.GOOS
	archName := runtime.GOARCH
	pluginDir := "./plugins"
	if *osPtr != "" {
		osName = *osPtr
	}
	if *archPtr != "" {
		archName = *archPtr
	}
	if *pluginDirPtr != "" {
		pluginDir = *pluginDirPtr
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
		return 1
	}

	c.ui.Info(fmt.Sprintf("Fetching 3rd party plugins in directory: %s", pluginDir))
	dirs := []string{pluginDir} //FindPlugins requires an array
	localPlugins := discovery.FindPlugins("provider", dirs)
	for k, _ := range localPlugins {
		c.ui.Info(fmt.Sprintf("plugin: %s (%s)", k.Name, k.Version))
	}
	installer := &discovery.ProviderInstaller{
		Dir: workDir,

		// FIXME: This is incorrect because it uses the protocol version of
		// this tool, rather than of the Terraform binary we just downloaded.
		// But we can't get this information from a Terraform binary, so
		// we'll just ignore this for now and use the same plugin installer
		// protocol version for terraform-bundle as the terraform shipped
		// with this release.
		//
		// NOTE: To target older versions of terraform, use the terraform-bundle
		// from the same tag.
		PluginProtocolVersion: discovery.PluginInstallProtocolVersion,

		OS:   osName,
		Arch: archName,
		Ui:   c.ui,
	}

	for name, constraintStrs := range config.Providers {
		for _, constraintStr := range constraintStrs {
			c.ui.Output(fmt.Sprintf("- Resolving %q provider (%s)...",
				name, constraintStr))
			foundPlugins := discovery.PluginMetaSet{}
			constraint := constraintStr.MustParse()
			for plugin, _ := range localPlugins {
				if plugin.Name == name && constraint.Allows(plugin.Version.MustParse()) {
					foundPlugins.Add(plugin)
				}
			}

			if len(foundPlugins) > 0 {
				plugin := foundPlugins.Newest()
				CopyFile(plugin.Path, workDir+"/terraform-provider-"+plugin.Name+"_v"+plugin.Version.MustParse().String()) //put into temp dir
			} else { //attempt to get from the public registry if not found locally
				c.ui.Output(fmt.Sprintf("- Checking for provider plugin on %s...",
					releaseHost))
				_, err := installer.Get(name, constraint)
				if err != nil {
					c.ui.Error(fmt.Sprintf("- Failed to resolve %s provider %s: %s", name, constraint, err))
					return 1
				}
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
		hdr.Method = zip.Deflate // be sure to compress files
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
		releaseHost, version, version, osName, archName,
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
  -os=name    		Target operating system the archive will be built for. Defaults
              		to that of the system where the command is being run.

  -arch=name  		Target CPU architecture the archive will be built for. Defaults
					to that of the system where the command is being run.
					  
  -plugin-dir=path 	The path to the custom plugins directory. Defaults to "./plugins".

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
	
	#Include a custom plugin to the bundle. Will search for the plugin in the 
	#plugins directory, and package it with the bundle archive. Plugin must have
	#a name of the form: terraform-provider-*-v*, and must be built with the operating
	#system and architecture that terraform enterprise is running, e.g. linux and amd64
	customplugin = ["0.1"]
  }

`
}
