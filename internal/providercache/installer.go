package providercache

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/apparentlymart/go-versions/versions"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/copydir"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// Installer is the main type in this package, representing a provider installer
// with a particular configuration-specific cache directory and an optional
// global cache directory.
type Installer struct {
	// targetDir is the cache directory we're ultimately aiming to get the
	// requested providers installed into.
	targetDir *Dir

	// source is the provider source that the installer will use to discover
	// what provider versions are available for installation and to
	// find the source locations for any versions that are not already
	// available via one of the cache directories.
	source getproviders.Source

	// globalCacheDir is an optional additional directory that will, if
	// provided, be treated as a read-through cache when retrieving new
	// provider versions. That is, new packages are fetched into this
	// directory first and then linked into targetDir, which allows sharing
	// both the disk space and the download time for a particular provider
	// version between different configurations on the same system.
	globalCacheDir *Dir

	// builtInProviderTypes is an optional set of types that should be
	// considered valid to appear in the special terraform.io/builtin/...
	// namespace, which we use for providers that are built in to Terraform
	// and thus do not need any separate installation step.
	builtInProviderTypes []string

	// unmanagedProviderTypes is a set of provider addresses that should be
	// considered implemented, but that Terraform does not manage the
	// lifecycle for, and therefore does not need to worry about the
	// installation of.
	unmanagedProviderTypes map[addrs.Provider]struct{}
}

// NewInstaller constructs and returns a new installer with the given target
// directory and provider source.
//
// A newly-created installer does not have a global cache directory configured,
// but a caller can make a follow-up call to SetGlobalCacheDir to provide
// one prior to taking any installation actions.
//
// The target directory MUST NOT also be an input consulted by the given source,
// or the result is undefined.
func NewInstaller(targetDir *Dir, source getproviders.Source) *Installer {
	return &Installer{
		targetDir: targetDir,
		source:    source,
	}
}

// SetGlobalCacheDir activates a second tier of caching for the receiving
// installer, with the given directory used as a read-through cache for
// installation operations that need to retrieve new packages.
//
// The global cache directory for an installer must never be the same as its
// target directory, and must not be used as one of its provider sources.
// If these overlap then undefined behavior will result.
func (i *Installer) SetGlobalCacheDir(cacheDir *Dir) {
	// A little safety check to catch straightforward mistakes where the
	// directories overlap. Better to panic early than to do
	// possibly-distructive actions on the cache directory downstream.
	if same, err := copydir.SameFile(i.targetDir.baseDir, cacheDir.baseDir); err == nil && same {
		panic(fmt.Sprintf("global cache directory %s must not match the installation target directory %s", cacheDir.baseDir, i.targetDir.baseDir))
	}
	i.globalCacheDir = cacheDir
}

// SetBuiltInProviderTypes tells the receiver to consider the type names in the
// given slice to be valid as providers in the special special
// terraform.io/builtin/... namespace that we use for providers that are
// built in to Terraform and thus do not need a separate installation step.
//
// If a caller requests installation of a provider in that namespace, the
// installer will treat it as a no-op if its name exists in this list, but
// will produce an error if it does not.
//
// The default, if this method isn't called, is for there to be no valid
// builtin providers.
//
// Do not modify the buffer under the given slice after passing it to this
// method.
func (i *Installer) SetBuiltInProviderTypes(types []string) {
	i.builtInProviderTypes = types
}

// SetUnmanagedProviderTypes tells the receiver to consider the providers
// indicated by the passed addrs.Providers as unmanaged. Terraform does not
// need to control the lifecycle of these providers, and they are assumed to be
// running already when Terraform is started. Because these are essentially
// processes, not binaries, Terraform will not do any work to ensure presence
// or versioning of these binaries.
func (i *Installer) SetUnmanagedProviderTypes(types map[addrs.Provider]struct{}) {
	i.unmanagedProviderTypes = types
}

// EnsureProviderVersions compares the given provider requirements with what
// is already available in the installer's target directory and then takes
// appropriate installation actions to ensure that suitable packages
// are available in the target cache directory.
//
// The given mode modifies how the operation will treat providers that already
// have acceptable versions available in the target cache directory. See the
// documentation for InstallMode and the InstallMode values for more
// information.
//
// The given context can be used to cancel the overall installation operation
// (causing any operations in progress to fail with an error), and can also
// include an InstallerEvents value for optional intermediate progress
// notifications.
//
// If a given InstallerEvents subscribes to notifications about installation
// failures then those notifications will be redundant with the ones included
// in the final returned error value so callers should show either one or the
// other, and not both.
func (i *Installer) EnsureProviderVersions(ctx context.Context, reqs getproviders.Requirements, mode InstallMode) (getproviders.Selections, error) {
	errs := map[addrs.Provider]error{}
	evts := installerEventsForContext(ctx)

	if cb := evts.PendingProviders; cb != nil {
		cb(reqs)
	}

	// Here we'll keep track of which exact version we've selected for each
	// provider in the requirements.
	selected := map[addrs.Provider]getproviders.Version{}

	// Step 1: Which providers might we need to fetch a new version of?
	// This produces the subset of requirements we need to ask the provider
	// source about.
	have := i.targetDir.AllAvailablePackages()
	mightNeed := map[addrs.Provider]getproviders.VersionSet{}
MightNeedProvider:
	for provider, versionConstraints := range reqs {
		if provider.IsBuiltIn() {
			// Built in providers do not require installation but we'll still
			// verify that the requested provider name is valid.
			valid := false
			for _, name := range i.builtInProviderTypes {
				if name == provider.Type {
					valid = true
					break
				}
			}
			var err error
			if valid {
				if len(versionConstraints) == 0 {
					// Other than reporting an event for the outcome of this
					// provider, we'll do nothing else with it: it's just
					// automatically available for use.
					if cb := evts.BuiltInProviderAvailable; cb != nil {
						cb(provider)
					}
				} else {
					// A built-in provider is not permitted to have an explicit
					// version constraint, because we can only use the version
					// that is built in to the current Terraform release.
					err = fmt.Errorf("built-in providers do not support explicit version constraints")
				}
			} else {
				err = fmt.Errorf("this Terraform release has no built-in provider named %q", provider.Type)
			}
			if err != nil {
				errs[provider] = err
				if cb := evts.BuiltInProviderFailure; cb != nil {
					cb(provider, err)
				}
			}
			continue
		}
		if _, ok := i.unmanagedProviderTypes[provider]; ok {
			// unmanaged providers do not require installation
			continue
		}
		acceptableVersions := versions.MeetingConstraints(versionConstraints)
		if mode.forceQueryAllProviders() {
			// If our mode calls for us to look for newer versions regardless
			// of whether an existing version is acceptable, we "might need"
			// _all_ of the requested providers.
			mightNeed[provider] = acceptableVersions
			continue
		}
		havePackages, ok := have[provider]
		if !ok { // If we don't have any versions at all then we'll definitely need it
			mightNeed[provider] = acceptableVersions
			continue
		}
		// If we already have some versions installed and our mode didn't
		// force us to check for new ones anyway then we'll check only if
		// there isn't already at least one version in our cache that is
		// in the set of acceptable versions.
		for _, pkg := range havePackages {
			if acceptableVersions.Has(pkg.Version) {
				// We will take no further actions for this provider, because
				// a version we have is already acceptable.
				selected[provider] = pkg.Version
				if cb := evts.ProviderAlreadyInstalled; cb != nil {
					cb(provider, pkg.Version)
				}
				continue MightNeedProvider
			}
		}
		// If we get here then we didn't find any cached version that is
		// in our set of acceptable versions.
		mightNeed[provider] = acceptableVersions
	}

	// Step 2: Query the provider source for each of the providers we selected
	// in the first step and select the latest available version that is
	// in the set of acceptable versions.
	//
	// This produces a set of packages to install to our cache in the next step.
	need := map[addrs.Provider]getproviders.Version{}
NeedProvider:
	for provider, acceptableVersions := range mightNeed {
		if err := ctx.Err(); err != nil {
			// If our context has been cancelled or reached a timeout then
			// we'll abort early, because subsequent operations against
			// that context will fail immediately anyway.
			return nil, err
		}

		if cb := evts.QueryPackagesBegin; cb != nil {
			cb(provider, reqs[provider])
		}
		available, warnings, err := i.source.AvailableVersions(ctx, provider)
		if err != nil {
			// TODO: Consider retrying a few times for certain types of
			// source errors that seem likely to be transient.
			errs[provider] = err
			if cb := evts.QueryPackagesFailure; cb != nil {
				cb(provider, err)
			}
			// We will take no further actions for this provider.
			continue
		}
		if len(warnings) > 0 {
			if cb := evts.QueryPackagesWarning; cb != nil {
				cb(provider, warnings)
			}
		}
		available.Sort()                           // put the versions in increasing order of precedence
		for i := len(available) - 1; i >= 0; i-- { // walk backwards to consider newer versions first
			if acceptableVersions.Has(available[i]) {
				need[provider] = available[i]
				if cb := evts.QueryPackagesSuccess; cb != nil {
					cb(provider, available[i])
				}
				continue NeedProvider
			}
		}
		// If we get here then the source has no packages that meet the given
		// version constraint, which we model as a query error.
		err = fmt.Errorf("no available releases match the given constraints %s", getproviders.VersionConstraintsString(reqs[provider]))
		errs[provider] = err
		if cb := evts.QueryPackagesFailure; cb != nil {
			cb(provider, err)
		}
	}

	// Step 3: For each provider version we've decided we need to install,
	// install its package into our target cache (possibly via the global cache).
	authResults := map[addrs.Provider]*getproviders.PackageAuthenticationResult{} // record auth results for all successfully fetched providers
	targetPlatform := i.targetDir.targetPlatform                                  // we inherit this to behave correctly in unit tests
	for provider, version := range need {
		if err := ctx.Err(); err != nil {
			// If our context has been cancelled or reached a timeout then
			// we'll abort early, because subsequent operations against
			// that context will fail immediately anyway.
			return nil, err
		}

		if i.globalCacheDir != nil {
			// Step 3a: If our global cache already has this version available then
			// we'll just link it in.
			if cached := i.globalCacheDir.ProviderVersion(provider, version); cached != nil {
				if cb := evts.LinkFromCacheBegin; cb != nil {
					cb(provider, version, i.globalCacheDir.baseDir)
				}
				err := i.targetDir.LinkFromOtherCache(cached)
				if err != nil {
					errs[provider] = err
					if cb := evts.LinkFromCacheFailure; cb != nil {
						cb(provider, version, err)
					}
					continue
				}
				// We'll fetch what we just linked to make sure it actually
				// did show up there.
				new := i.targetDir.ProviderVersion(provider, version)
				if new == nil {
					err := fmt.Errorf("after linking %s from provider cache at %s it is still not detected in the target directory; this is a bug in Terraform", provider, i.globalCacheDir.baseDir)
					if cb := evts.LinkFromCacheFailure; cb != nil {
						cb(provider, version, err)
					}
					continue
				}
				selected[provider] = version
				if cb := evts.LinkFromCacheSuccess; cb != nil {
					cb(provider, version, new.PackageDir)
				}
				continue // Don't need to do full install, then.
			}
		}

		// Step 3b: Get the package metadata for the selected version from our
		// provider source.
		//
		// This is the step where we might detect and report that the provider
		// isn't available for the current platform.
		if cb := evts.FetchPackageMeta; cb != nil {
			cb(provider, version)
		}
		meta, err := i.source.PackageMeta(ctx, provider, version, targetPlatform)
		if err != nil {
			errs[provider] = err
			if cb := evts.FetchPackageFailure; cb != nil {
				cb(provider, version, err)
			}
			continue
		}

		// Step 3c: Retrieve the package indicated by the metadata we received,
		// either directly into our target directory or via the global cache
		// directory.
		if cb := evts.FetchPackageBegin; cb != nil {
			cb(provider, version, meta.Location)
		}
		var installTo, linkTo *Dir
		if i.globalCacheDir != nil {
			installTo = i.globalCacheDir
			linkTo = i.targetDir
		} else {
			installTo = i.targetDir
			linkTo = nil // no linking needed
		}
		authResult, err := installTo.InstallPackage(ctx, meta)
		if err != nil {
			// TODO: Consider retrying for certain kinds of error that seem
			// likely to be transient. For now, we just treat all errors equally.
			errs[provider] = err
			if cb := evts.FetchPackageFailure; cb != nil {
				cb(provider, version, err)
			}
			continue
		}
		new := installTo.ProviderVersion(provider, version)
		if new == nil {
			err := fmt.Errorf("after installing %s it is still not detected in the target directory; this is a bug in Terraform", provider)
			errs[provider] = err
			if cb := evts.FetchPackageFailure; cb != nil {
				cb(provider, version, err)
			}
			continue
		}
		if linkTo != nil {
			// We skip emitting the "LinkFromCache..." events here because
			// it's simpler for the caller to treat them as mutually exclusive.
			// We can just subsume the linking step under the "FetchPackage..."
			// series here (and that's why we use FetchPackageFailure below).
			err := linkTo.LinkFromOtherCache(new)
			if err != nil {
				errs[provider] = err
				if cb := evts.FetchPackageFailure; cb != nil {
					cb(provider, version, err)
				}
				continue
			}
		}
		authResults[provider] = authResult
		selected[provider] = version
		if cb := evts.FetchPackageSuccess; cb != nil {
			cb(provider, version, new.PackageDir, authResult)
		}
	}

	// Emit final event for fetching if any were successfully fetched
	if cb := evts.ProvidersFetched; cb != nil && len(authResults) > 0 {
		cb(authResults)
	}

	// We'll remember our selections in a lock file inside the target directory,
	// so callers can recover those exact selections later by calling
	// SelectedPackages on the same installer.
	lockEntries := map[addrs.Provider]lockFileEntry{}
	for provider, version := range selected {
		cached := i.targetDir.ProviderVersion(provider, version)
		if cached == nil {
			err := fmt.Errorf("selected package for %s is no longer present in the target directory; this is a bug in Terraform", provider)
			errs[provider] = err
			if cb := evts.HashPackageFailure; cb != nil {
				cb(provider, version, err)
			}
			continue
		}
		if _, err := cached.ExecutableFile(); err != nil {
			err := fmt.Errorf("provider binary not found: %s", err)
			errs[provider] = err
			if cb := evts.HashPackageFailure; cb != nil {
				cb(provider, version, err)
			}
			continue
		}
		hash, err := cached.Hash()
		if err != nil {
			errs[provider] = fmt.Errorf("failed to calculate checksum for installed provider %s package: %s", provider, err)
			if cb := evts.HashPackageFailure; cb != nil {
				cb(provider, version, err)
			}
			continue
		}
		lockEntries[provider] = lockFileEntry{
			SelectedVersion: version,
			PackageHash:     hash.String(),
		}
	}
	err := i.lockFile().Write(lockEntries)
	if err != nil {
		// This is one of few cases where this function does _not_ return an
		// InstallerError, because failure to write the lock file is a more
		// general problem, not specific to a certain provider.
		return selected, fmt.Errorf("failed to record a manifest of selected providers: %s", err)
	}

	if len(errs) > 0 {
		return selected, InstallerError{
			ProviderErrors: errs,
		}
	}
	return selected, nil
}

func (i *Installer) lockFile() *lockFile {
	return &lockFile{
		filename: filepath.Join(i.targetDir.baseDir, "selections.json"),
	}
}

// SelectedPackages returns the metadata about the packages chosen by the
// most recent call to EnsureProviderVersions, which are recorded in a lock
// file in the installer's target directory.
//
// If EnsureProviderVersions has never been run against the current target
// directory, the result is a successful empty response indicating that nothing
// is selected.
//
// SelectedPackages also verifies that the package contents are consistent
// with the checksums that were recorded at installation time, reporting an
// error if not.
func (i *Installer) SelectedPackages() (map[addrs.Provider]*CachedProvider, error) {
	entries, err := i.lockFile().Read()
	if err != nil {
		// Read does not return an error for "file not found", so this should
		// always be some other error.
		return nil, fmt.Errorf("failed to read selections file: %s", err)
	}

	ret := make(map[addrs.Provider]*CachedProvider, len(entries))
	errs := make(map[addrs.Provider]error)
	for provider, entry := range entries {
		cached := i.targetDir.ProviderVersion(provider, entry.SelectedVersion)
		if cached == nil {
			errs[provider] = fmt.Errorf("package for selected version %s is no longer available in the local cache directory", entry.SelectedVersion)
			continue
		}

		hash, err := getproviders.ParseHash(entry.PackageHash)
		if err != nil {
			errs[provider] = fmt.Errorf("local cache for %s has invalid hash %q: %s", provider, entry.PackageHash, err)
			continue
		}

		ok, err := cached.MatchesHash(hash)
		if err != nil {
			errs[provider] = fmt.Errorf("failed to verify checksum for v%s package: %s", entry.SelectedVersion, err)
			continue
		}
		if !ok {
			errs[provider] = fmt.Errorf("checksum mismatch for v%s package", entry.SelectedVersion)
			continue
		}
		ret[provider] = cached
	}

	if len(errs) > 0 {
		return ret, InstallerError{
			ProviderErrors: errs,
		}
	}
	return ret, nil
}

// InstallMode customizes the details of how an install operation treats
// providers that have versions already cached in the target directory.
type InstallMode rune

const (
	// InstallNewProvidersOnly is an InstallMode that causes the installer
	// to accept any existing version of a requested provider that is already
	// cached as long as it's in the given version sets, without checking
	// whether new versions are available that are also in the given version
	// sets.
	InstallNewProvidersOnly InstallMode = 'N'

	// InstallUpgrades is an InstallMode that causes the installer to check
	// all requested providers to see if new versions are available that
	// are also in the given version sets, even if a suitable version of
	// a given provider is already available.
	InstallUpgrades InstallMode = 'U'
)

func (m InstallMode) forceQueryAllProviders() bool {
	return m == InstallUpgrades
}

// InstallerError is an error type that may be returned (but is not guaranteed)
// from Installer.EnsureProviderVersions to indicate potentially several
// separate failed installation outcomes for different providers included in
// the overall request.
type InstallerError struct {
	ProviderErrors map[addrs.Provider]error
}

func (err InstallerError) Error() string {
	addrs := make([]addrs.Provider, 0, len(err.ProviderErrors))
	for addr := range err.ProviderErrors {
		addrs = append(addrs, addr)
	}
	sort.Slice(addrs, func(i, j int) bool {
		return addrs[i].LessThan(addrs[j])
	})
	var b strings.Builder
	b.WriteString("some providers could not be installed:\n")
	for _, addr := range addrs {
		providerErr := err.ProviderErrors[addr]
		fmt.Fprintf(&b, "- %s: %s\n", addr, providerErr)
	}
	return b.String()
}
