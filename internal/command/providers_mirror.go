package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProvidersMirrorCommand is a Command implementation that implements the
// "terraform providers mirror" command, which populates a directory with
// local copies of provider plugins needed by the current configuration so
// that the mirror can be used to work offline, or similar.
type ProvidersMirrorCommand struct {
	Meta
}

func (c *ProvidersMirrorCommand) Synopsis() string {
	return "Save local copies of all required provider plugins"
}

func (c *ProvidersMirrorCommand) Run(args []string) int {
	args = c.Meta.process(args)
	cmdFlags := c.Meta.defaultFlagSet("providers mirror")
	var optPlatforms FlagStringSlice
	cmdFlags.Var(&optPlatforms, "platform", "target platform")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	var diags tfdiags.Diagnostics

	args = cmdFlags.Args()
	if len(args) != 1 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No output directory specified",
			"The providers mirror command requires an output directory as a command-line argument.",
		))
		c.showDiagnostics(diags)
		return 1
	}
	outputDir := args[0]

	var platforms []getproviders.Platform
	if len(optPlatforms) == 0 {
		platforms = []getproviders.Platform{getproviders.CurrentPlatform}
	} else {
		platforms = make([]getproviders.Platform, 0, len(optPlatforms))
		for _, platformStr := range optPlatforms {
			platform, err := getproviders.ParsePlatform(platformStr)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid target platform",
					fmt.Sprintf("The string %q given in the -platform option is not a valid target platform: %s.", platformStr, err),
				))
				continue
			}
			platforms = append(platforms, platform)
		}
	}

	config, confDiags := c.loadConfig(".")
	diags = diags.Append(confDiags)
	reqs, moreDiags := config.ProviderRequirements()
	diags = diags.Append(moreDiags)

	// If we have any error diagnostics already then we won't proceed further.
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Unlike other commands, this command always consults the origin registry
	// for every provider so that it can be used to update a local mirror
	// directory without needing to first disable that local mirror
	// in the CLI configuration.
	source := getproviders.NewMemoizeSource(
		getproviders.NewRegistrySource(c.Services),
	)

	// Providers from registries always use HTTP, so we don't need the full
	// generality of go-getter but it's still handy to use the HTTP getter
	// as an easy way to download over HTTP into a file on disk.
	httpGetter := getter.HttpGetter{
		Client: httpclient.New(),
		Netrc:  true,
	}

	// The following logic is similar to that used by the provider installer
	// in package providercache, but different in a few ways:
	// - It produces the packed directory layout rather than the unpacked
	//   layout we require in provider cache directories.
	// - It generates JSON index files that can be read by the
	//   getproviders.HTTPMirrorSource installation method if the result were
	//   copied into the docroot of an HTTP server.
	// - It can mirror packages for potentially many different target platforms,
	//   so that we can construct a multi-platform mirror regardless of which
	//   platform we run this command on.
	// - It ignores what's already present and just always downloads everything
	//   that the configuration requires. This is a command intended to be run
	//   infrequently to update a mirror, so it doesn't need to optimize away
	//   fetches of packages that might already be present.

	ctx, cancel := c.InterruptibleContext()
	defer cancel()
	for provider, constraints := range reqs {
		if provider.IsBuiltIn() {
			c.Ui.Output(fmt.Sprintf("- Skipping %s because it is built in to Terraform CLI", provider.ForDisplay()))
			continue
		}
		constraintsStr := getproviders.VersionConstraintsString(constraints)
		c.Ui.Output(fmt.Sprintf("- Mirroring %s...", provider.ForDisplay()))
		// First we'll look for the latest version that matches the given
		// constraint, which we'll then try to mirror for each target platform.
		acceptable := versions.MeetingConstraints(constraints)
		avail, _, err := source.AvailableVersions(ctx, provider)
		candidates := avail.Filter(acceptable)
		if err == nil && len(candidates) == 0 {
			err = fmt.Errorf("no releases match the given constraints %s", constraintsStr)
		}
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider not available",
				fmt.Sprintf("Failed to download %s from its origin registry: %s.", provider.String(), err),
			))
			continue
		}
		selected := candidates.Newest()
		if len(constraintsStr) > 0 {
			c.Ui.Output(fmt.Sprintf("  - Selected v%s to meet constraints %s", selected.String(), constraintsStr))
		} else {
			c.Ui.Output(fmt.Sprintf("  - Selected v%s with no constraints", selected.String()))
		}
		for _, platform := range platforms {
			c.Ui.Output(fmt.Sprintf("  - Downloading package for %s...", platform.String()))
			meta, err := source.PackageMeta(ctx, provider, selected, platform)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Provider release not available",
					fmt.Sprintf("Failed to download %s v%s for %s: %s.", provider.String(), selected.String(), platform.String(), err),
				))
				continue
			}
			urlStr, ok := meta.Location.(getproviders.PackageHTTPURL)
			if !ok {
				// We don't expect to get non-HTTP locations here because we're
				// using the registry source, so this seems like a bug in the
				// registry source.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Provider release not available",
					fmt.Sprintf("Failed to download %s v%s for %s: Terraform's provider registry client returned unexpected location type %T. This is a bug in Terraform.", provider.String(), selected.String(), platform.String(), meta.Location),
				))
				continue
			}
			urlObj, err := url.Parse(string(urlStr))
			if err != nil {
				// We don't expect to get non-HTTP locations here because we're
				// using the registry source, so this seems like a bug in the
				// registry source.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid URL for provider release",
					fmt.Sprintf("The origin registry for %s returned an invalid URL for v%s on %s: %s.", provider.String(), selected.String(), platform.String(), err),
				))
				continue
			}
			// targetPath is the path where we ultimately want to place the
			// downloaded archive, but we'll place it initially at stagingPath
			// so we can verify its checksums and signatures before making
			// it discoverable to mirror clients. (stagingPath intentionally
			// does not follow the filesystem mirror file naming convention.)
			targetPath := meta.PackedFilePath(outputDir)
			stagingPath := filepath.Join(filepath.Dir(targetPath), "."+filepath.Base(targetPath))
			err = httpGetter.GetFile(stagingPath, urlObj)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Cannot download provider release",
					fmt.Sprintf("Failed to download %s v%s for %s: %s.", provider.String(), selected.String(), platform.String(), err),
				))
				continue
			}
			if meta.Authentication != nil {
				result, err := meta.Authentication.AuthenticatePackage(getproviders.PackageLocalArchive(stagingPath))
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid provider package",
						fmt.Sprintf("Failed to authenticate %s v%s for %s: %s.", provider.String(), selected.String(), platform.String(), err),
					))
					continue
				}
				c.Ui.Output(fmt.Sprintf("  - Package authenticated: %s", result))
			}
			os.Remove(targetPath) // okay if it fails because we're going to try to rename over it next anyway
			err = os.Rename(stagingPath, targetPath)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Cannot download provider release",
					fmt.Sprintf("Failed to place %s package into mirror directory: %s.", provider.String(), err),
				))
				continue
			}
		}
	}

	// Now we'll generate or update the JSON index files in the directory.
	// We do this by scanning the directory to see what is present, rather than
	// by relying on the selections we made above, because we want to still
	// include in the indices any packages that were already present and
	// not affected by the changes we just made.
	available, err := getproviders.SearchLocalDirectory(outputDir)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to update indexes",
			fmt.Sprintf("Could not scan the output directory to get package metadata for the JSON indexes: %s.", err),
		))
		available = nil // the following loop will be a no-op
	}
	for provider, metas := range available {
		if len(metas) == 0 {
			continue // should never happen, but we'll be resilient
		}
		// The index files live in the same directory as the package files,
		// so to figure that out without duplicating the path-building logic
		// we'll ask the getproviders package to build an archive filename
		// for a fictitious package and then use the directory portion of it.
		indexDir := filepath.Dir(getproviders.PackedFilePathForPackage(
			outputDir, provider, versions.Unspecified, getproviders.CurrentPlatform,
		))
		indexVersions := map[string]interface{}{}
		indexArchives := map[getproviders.Version]map[string]interface{}{}
		for _, meta := range metas {
			archivePath, ok := meta.Location.(getproviders.PackageLocalArchive)
			if !ok {
				// only archive files are eligible to be included in JSON
				// indices for a network mirror.
				continue
			}
			archiveFilename := filepath.Base(string(archivePath))
			version := meta.Version
			platform := meta.TargetPlatform
			hash, err := meta.Hash()
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to update indexes",
					fmt.Sprintf("Failed to determine a hash value for %s v%s on %s: %s.", provider, version, platform, err),
				))
				continue
			}
			indexVersions[meta.Version.String()] = map[string]interface{}{}
			if _, ok := indexArchives[version]; !ok {
				indexArchives[version] = map[string]interface{}{}
			}
			indexArchives[version][platform.String()] = map[string]interface{}{
				"url":    archiveFilename,         // a relative URL from the index file's URL
				"hashes": []string{hash.String()}, // an array to allow for additional hash formats in future
			}
		}
		mainIndex := map[string]interface{}{
			"versions": indexVersions,
		}
		mainIndexJSON, err := json.MarshalIndent(mainIndex, "", "  ")
		if err != nil {
			// Should never happen because the input here is entirely under
			// our control.
			panic(fmt.Sprintf("failed to encode main index: %s", err))
		}
		// TODO: Ideally we would do these updates as atomic swap operations by
		// creating a new file and then renaming it over the old one, in case
		// this directory is the docroot of a live mirror. An atomic swap
		// requires platform-specific code though: os.Rename alone can't do it
		// when running on Windows as of Go 1.13. We should revisit this once
		// we're supporting network mirrors, to avoid having them briefly
		// become corrupted during updates.
		err = ioutil.WriteFile(filepath.Join(indexDir, "index.json"), mainIndexJSON, 0644)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to update indexes",
				fmt.Sprintf("Failed to write an updated JSON index for %s: %s.", provider, err),
			))
		}
		for version, archiveIndex := range indexArchives {
			versionIndex := map[string]interface{}{
				"archives": archiveIndex,
			}
			versionIndexJSON, err := json.MarshalIndent(versionIndex, "", "  ")
			if err != nil {
				// Should never happen because the input here is entirely under
				// our control.
				panic(fmt.Sprintf("failed to encode version index: %s", err))
			}
			err = ioutil.WriteFile(filepath.Join(indexDir, version.String()+".json"), versionIndexJSON, 0644)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to update indexes",
					fmt.Sprintf("Failed to write an updated JSON index for %s v%s: %s.", provider, version, err),
				))
			}
		}
	}

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}
	return 0
}

func (c *ProvidersMirrorCommand) Help() string {
	return `
Usage: terraform [global options] providers mirror [options] <target-dir>

  Populates a local directory with copies of the provider plugins needed for
  the current configuration, so that the directory can be used either directly
  as a filesystem mirror or as the basis for a network mirror and thus obtain
  those providers without access to their origin registries in future.

  The mirror directory will contain JSON index files that can be published
  along with the mirrored packages on a static HTTP file server to produce
  a network mirror. Those index files will be ignored if the directory is
  used instead as a local filesystem mirror.

Options:

  -platform=os_arch  Choose which target platform to build a mirror for.
                     By default Terraform will obtain plugin packages
                     suitable for the platform where you run this command.
                     Use this flag multiple times to include packages for
                     multiple target systems.

                     Target names consist of an operating system and a CPU
                     architecture. For example, "linux_amd64" selects the
                     Linux operating system running on an AMD64 or x86_64
                     CPU. Each provider is available only for a limited
                     set of target platforms.
`
}
