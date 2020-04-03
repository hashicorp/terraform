package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/apparentlymart/go-userdirs/userdirs"

	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/command/cliconfig"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// providerSource constructs a provider source based on a combination of the
// CLI configuration and some default search locations. This will be the
// provider source used for provider installation in the "terraform init"
// command, unless overridden by the special -plugin-dir option.
func providerSource(services *disco.Disco) getproviders.Source {
	// We're not yet using the CLI config here because we've not implemented
	// yet the new configuration constructs to customize provider search
	// locations. That'll come later.
	// For now, we have a fixed set of search directories:
	// - The "terraform.d/plugins" directory in the current working directory,
	//   which we've historically documented as a place to put plugins as a
	//   way to include them in bundles uploaded to Terraform Cloud, where
	//   there has historically otherwise been no way to use custom providers.
	// - The "plugins" subdirectory of the CLI config search directory.
	//   (thats ~/.terraform.d/plugins on Unix systems, equivalents elsewhere)
	// - The "plugins" subdirectory of any platform-specific search paths,
	//   following e.g. the XDG base directory specification on Unix systems,
	//   Apple's guidelines on OS X, and "known folders" on Windows.
	//
	// Those directories are checked in addition to the direct upstream
	// registry specified in the provider's address.
	var searchRules []getproviders.MultiSourceSelector

	addLocalDir := func(dir string) {
		// We'll make sure the directory actually exists before we add it,
		// because otherwise installation would always fail trying to look
		// in non-existent directories. (This is done here rather than in
		// the source itself because explicitly-selected directories via the
		// CLI config, once we have them, _should_ produce an error if they
		// don't exist to help users get their configurations right.)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			log.Printf("[DEBUG] will search for provider plugins in %s", dir)
			searchRules = append(searchRules, getproviders.MultiSourceSelector{
				Source: getproviders.NewFilesystemMirrorSource(dir),
			})
		} else {
			log.Printf("[DEBUG] ignoring non-existing provider search directory %s", dir)
		}
	}

	addLocalDir("terraform.d/plugins") // our "vendor" directory
	cliConfigDir, err := cliconfig.ConfigDir()
	if err != nil {
		addLocalDir(filepath.Join(cliConfigDir, "plugins"))
	}

	// This "userdirs" library implements an appropriate user-specific and
	// app-specific directory layout for the current platform, such as XDG Base
	// Directory on Unix, using the following name strings to construct a
	// suitable application-specific subdirectory name following the
	// conventions for each platform:
	//
	//   XDG (Unix): lowercase of the first string, "terraform"
	//   Windows:    two-level heirarchy of first two strings, "HashiCorp\Terraform"
	//   OS X:       reverse-DNS unique identifier, "io.terraform".
	sysSpecificDirs := userdirs.ForApp("Terraform", "HashiCorp", "io.terraform")
	for _, dir := range sysSpecificDirs.DataSearchPaths("plugins") {
		addLocalDir(dir)
	}

	// Last but not least, the main registry source! We'll wrap a caching
	// layer around this one to help optimize the several network requests
	// we'll end up making to it while treating it as one of several sources
	// in a MultiSource (as recommended in the MultiSource docs).
	// This one is listed last so that if a particular version is available
	// both in one of the above directories _and_ in a remote registry, the
	// local copy will take precedence.
	searchRules = append(searchRules, getproviders.MultiSourceSelector{
		Source: getproviders.NewMemoizeSource(
			getproviders.NewRegistrySource(services),
		),
	})

	return getproviders.MultiSource(searchRules)
}
