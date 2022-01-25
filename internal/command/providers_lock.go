package command

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProvidersLockCommand is a Command implementation that implements the
// "terraform providers lock" command, which creates or updates the current
// configuration's dependency lock file using information from upstream
// registries, regardless of the provider installation configuration that
// is configured for normal provider installation.
type ProvidersLockCommand struct {
	Meta
}

func (c *ProvidersLockCommand) Synopsis() string {
	return "Write out dependency locks for the configured providers"
}

func (c *ProvidersLockCommand) Run(args []string) int {
	args = c.Meta.process(args)
	cmdFlags := c.Meta.defaultFlagSet("providers lock")
	var optPlatforms FlagStringSlice
	var fsMirrorDir string
	var netMirrorURL string
	cmdFlags.Var(&optPlatforms, "platform", "target platform")
	cmdFlags.StringVar(&fsMirrorDir, "fs-mirror", "", "filesystem mirror directory")
	cmdFlags.StringVar(&netMirrorURL, "net-mirror", "", "network mirror base URL")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	var diags tfdiags.Diagnostics

	if fsMirrorDir != "" && netMirrorURL != "" {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid installation method options",
			"The -fs-mirror and -net-mirror command line options are mutually-exclusive.",
		))
		c.showDiagnostics(diags)
		return 1
	}

	providerStrs := cmdFlags.Args()

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

	// Unlike other commands, this command ignores the installation methods
	// selected in the CLI configuration and instead chooses an installation
	// method based on CLI options.
	//
	// This is so that folks who use a local mirror for everyday use can
	// use this command to populate their lock files from upstream so
	// subsequent "terraform init" calls can then verify the local mirror
	// against the upstream checksums.
	var source getproviders.Source
	switch {
	case fsMirrorDir != "":
		source = getproviders.NewFilesystemMirrorSource(fsMirrorDir)
	case netMirrorURL != "":
		u, err := url.Parse(netMirrorURL)
		if err != nil || u.Scheme != "https" {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid network mirror URL",
				"The -net-mirror option requires a valid https: URL as the mirror base URL.",
			))
			c.showDiagnostics(diags)
			return 1
		}
		source = getproviders.NewHTTPMirrorSource(u, c.Services.CredentialsSource())
	default:
		// With no special options we consult upstream registries directly,
		// because that gives us the most information to produce as complete
		// and portable as possible a lock entry.
		source = getproviders.NewRegistrySource(c.Services)
	}

	config, confDiags := c.loadConfig(".")
	diags = diags.Append(confDiags)
	reqs, hclDiags := config.ProviderRequirements()
	diags = diags.Append(hclDiags)

	// If we have explicit provider selections on the command line then
	// we'll modify "reqs" to only include those. Modifying this is okay
	// because config.ProviderRequirements generates a fresh map result
	// for each call.
	if len(providerStrs) != 0 {
		providers := map[addrs.Provider]struct{}{}
		for _, raw := range providerStrs {
			addr, moreDiags := addrs.ParseProviderSourceString(raw)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				continue
			}
			providers[addr] = struct{}{}
			if _, exists := reqs[addr]; !exists {
				// Can't request a provider that isn't required by the
				// current configuration.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid provider argument",
					fmt.Sprintf("The provider %s is not required by the current configuration.", addr.String()),
				))
			}
		}

		for addr := range reqs {
			if _, exists := providers[addr]; !exists {
				delete(reqs, addr)
			}
		}
	}

	// We'll also ignore any providers that don't participate in locking.
	for addr := range reqs {
		if !depsfile.ProviderIsLockable(addr) {
			delete(reqs, addr)
		}
	}

	// We'll start our work with whatever locks we already have, so that
	// we'll honor any existing version selections and just add additional
	// hashes for them.
	oldLocks, moreDiags := c.lockedDependencies()
	diags = diags.Append(moreDiags)

	// If we have any error diagnostics already then we won't proceed further.
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Our general strategy here is to install the requested providers into
	// a separate temporary directory -- thus ensuring that the results won't
	// ever be inadvertently executed by other Terraform commands -- and then
	// use the results of that installation to update the lock file for the
	// current working directory. Because we throwaway the packages we
	// downloaded after completing our work, a subsequent "terraform init" will
	// then respect the CLI configuration's provider installation strategies
	// but will verify the packages against the hashes we found upstream.

	// Because our Installer abstraction is a per-platform idea, we'll
	// instantiate one for each of the platforms the user requested, and then
	// merge all of the generated locks together at the end.
	updatedLocks := map[getproviders.Platform]*depsfile.Locks{}
	selectedVersions := map[addrs.Provider]getproviders.Version{}
	ctx, cancel := c.InterruptibleContext()
	defer cancel()
	for _, platform := range platforms {
		tempDir, err := ioutil.TempDir("", "terraform-providers-lock")
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Could not create temporary directory",
				fmt.Sprintf("Failed to create a temporary directory for staging the requested provider packages: %s.", err),
			))
			break
		}
		defer os.RemoveAll(tempDir)

		evts := &providercache.InstallerEvents{
			// Our output from this command is minimal just to show that
			// we're making progress, rather than just silently hanging.
			FetchPackageBegin: func(provider addrs.Provider, version getproviders.Version, loc getproviders.PackageLocation) {
				c.Ui.Output(fmt.Sprintf("- Fetching %s %s for %s...", provider.ForDisplay(), version, platform))
				if prevVersion, exists := selectedVersions[provider]; exists && version != prevVersion {
					// This indicates a weird situation where we ended up
					// selecting a different version for one platform than
					// for another. We won't be able to merge the result
					// in that case, so we'll generate an error.
					//
					// This could potentially happen if there's a provider
					// we've not previously recorded in the lock file and
					// the available versions change while we're running. To
					// avoid that would require pre-locking all of the
					// providers, which is complicated to do with the building
					// blocks we have here, and so we'll wait to do it only
					// if this situation arises often in practice.
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Inconsistent provider versions",
						fmt.Sprintf(
							"The version constraint for %s selected inconsistent versions for different platforms, which is unexpected.\n\nThe upstream registry may have changed its available versions during Terraform's work. If so, re-running this command may produce a successful result.",
							provider,
						),
					))
				}
				selectedVersions[provider] = version
			},
			FetchPackageSuccess: func(provider addrs.Provider, version getproviders.Version, localDir string, auth *getproviders.PackageAuthenticationResult) {
				var keyID string
				if auth != nil && auth.ThirdPartySigned() {
					keyID = auth.KeyID
				}
				if keyID != "" {
					keyID = c.Colorize().Color(fmt.Sprintf(", key ID [reset][bold]%s[reset]", keyID))
				}
				c.Ui.Output(fmt.Sprintf("- Obtained %s checksums for %s (%s%s)", provider.ForDisplay(), platform, auth, keyID))
			},
		}
		ctx := evts.OnContext(ctx)

		dir := providercache.NewDirWithPlatform(tempDir, platform)
		installer := providercache.NewInstaller(dir, source)

		newLocks, err := installer.EnsureProviderVersions(ctx, oldLocks, reqs, providercache.InstallNewProvidersOnly)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Could not retrieve providers for locking",
				fmt.Sprintf("Terraform failed to fetch the requested providers for %s in order to calculate their checksums: %s.", platform, err),
			))
			break
		}
		updatedLocks[platform] = newLocks
	}

	// If we have any error diagnostics from installation then we won't
	// proceed to merging and updating the lock file on disk.
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// We now have a separate updated locks object for each platform. We need
	// to merge those all together so that the final result has the union of
	// all of the checksums we saw for each of the providers we've worked on.
	//
	// We'll copy the old locks first because we want to retain any existing
	// locks for providers that we _didn't_ visit above.
	newLocks := oldLocks.DeepCopy()
	for provider := range reqs {
		oldLock := oldLocks.Provider(provider)

		var version getproviders.Version
		var constraints getproviders.VersionConstraints
		var hashes []getproviders.Hash
		if oldLock != nil {
			version = oldLock.Version()
			constraints = oldLock.VersionConstraints()
			hashes = append(hashes, oldLock.AllHashes()...)
		}
		for _, platformLocks := range updatedLocks {
			platformLock := platformLocks.Provider(provider)
			if platformLock == nil {
				continue // weird, but we'll tolerate it to avoid crashing
			}
			version = platformLock.Version()
			constraints = platformLock.VersionConstraints()

			// We don't make any effort to deduplicate hashes between different
			// platforms here, because the SetProvider method we call below
			// handles that automatically.
			hashes = append(hashes, platformLock.AllHashes()...)
		}
		newLocks.SetProvider(provider, version, constraints, hashes)
	}

	moreDiags = c.replaceLockedDependencies(newLocks)
	diags = diags.Append(moreDiags)

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	c.Ui.Output(c.Colorize().Color("\n[bold][green]Success![reset] [bold]Terraform has updated the lock file.[reset]"))
	c.Ui.Output("\nReview the changes in .terraform.lock.hcl and then commit to your\nversion control system to retain the new checksums.\n")
	return 0
}

func (c *ProvidersLockCommand) Help() string {
	return `
Usage: terraform [global options] providers lock [options] [providers...]

  Normally the dependency lock file (.terraform.lock.hcl) is updated
  automatically by "terraform init", but the information available to the
  normal provider installer can be constrained when you're installing providers
  from filesystem or network mirrors, and so the generated lock file can end
  up incomplete.

  The "providers lock" subcommand addresses that by updating the lock file
  based on the official packages available in the origin registry, ignoring
  the currently-configured installation strategy.

  After this command succeeds, the lock file will contain suitable checksums
  to allow installation of the providers needed by the current configuration
  on all of the selected platforms.

  By default this command updates the lock file for every provider declared
  in the configuration. You can override that behavior by providing one or
  more provider source addresses on the command line.

Options:

  -fs-mirror=dir     Consult the given filesystem mirror directory instead
                     of the origin registry for each of the given providers.

                     This would be necessary to generate lock file entries for
                     a provider that is available only via a mirror, and not
                     published in an upstream registry. In this case, the set
                     of valid checksums will be limited only to what Terraform
                     can learn from the data in the mirror directory.

  -net-mirror=url    Consult the given network mirror (given as a base URL)
                     instead of the origin registry for each of the given
                     providers.

                     This would be necessary to generate lock file entries for
                     a provider that is available only via a mirror, and not
                     published in an upstream registry. In this case, the set
                     of valid checksums will be limited only to what Terraform
                     can learn from the data in the mirror indices.

  -platform=os_arch  Choose a target platform to request package checksums
                     for.

                     By default Terraform will request package checksums
                     suitable only for the platform where you run this
                     command. Use this option multiple times to include
                     checksums for multiple target systems.

                     Target names consist of an operating system and a CPU
                     architecture. For example, "linux_amd64" selects the
                     Linux operating system running on an AMD64 or x86_64
                     CPU. Each provider is available only for a limited
                     set of target platforms.
`
}
