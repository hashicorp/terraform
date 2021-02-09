package providercache

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/apparentlymart/go-versions/versions"

	"github.com/hashicorp/terraform/addrs"
	copydir "github.com/hashicorp/terraform/internal/copy"
	"github.com/hashicorp/terraform/internal/depsfile"
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

// Clone returns a new Installer which has the a new target directory but
// the same optional global cache directory, the same installation sources,
// and the same built-in/unmanaged providers. The result can be mutated further
// using the various setter methods without affecting the original.
func (i *Installer) Clone(targetDir *Dir) *Installer {
	// For now all of our setter methods just overwrite field values in
	// their entirety, rather than mutating things on the other side of
	// the shared pointers, and so we can safely just shallow-copy the
	// root. We might need to be more careful here if in future we add
	// methods that allow deeper mutations through the stored pointers.
	ret := *i
	ret.targetDir = targetDir
	return &ret
}

// ProviderSource returns the getproviders.Source that the installer would
// use for installing any new providers.
func (i *Installer) ProviderSource() getproviders.Source {
	return i.source
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

// HasGlobalCacheDir returns true if someone has previously called
// SetGlobalCacheDir to configure a global cache directory for this installer.
func (i *Installer) HasGlobalCacheDir() bool {
	return i.globalCacheDir != nil
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
func (i *Installer) EnsureProviderVersions(ctx context.Context, locks *depsfile.Locks, reqs getproviders.Requirements, mode InstallMode) (*depsfile.Locks, error) {
	errs := map[addrs.Provider]error{}
	evts := installerEventsForContext(ctx)

	// We'll work with a copy of the given locks, so we can modify it and
	// return the updated locks without affecting the caller's object.
	// We'll add or replace locks in here during our work so that the final
	// locks file reflects what the installer has selected.
	locks = locks.DeepCopy()

	if cb := evts.PendingProviders; cb != nil {
		cb(reqs)
	}

	// Step 1: Which providers might we need to fetch a new version of?
	// This produces the subset of requirements we need to ask the provider
	// source about. If we're in the normal (non-upgrade) mode then we'll
	// just ask the source to confirm the continued existence of what
	// was locked, or otherwise we'll find the newest version matching the
	// configured version constraint.
	mightNeed := map[addrs.Provider]getproviders.VersionSet{}
	locked := map[addrs.Provider]bool{}
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
		if !mode.forceQueryAllProviders() {
			// If we're not forcing potential changes of version then an
			// existing selection from the lock file takes priority over
			// the currently-configured version constraints.
			if lock := locks.Provider(provider); lock != nil {
				if !acceptableVersions.Has(lock.Version()) {
					err := fmt.Errorf(
						"locked provider %s %s does not match configured version constraint %s; must use terraform init -upgrade to allow selection of new versions",
						provider, lock.Version(), getproviders.VersionConstraintsString(versionConstraints),
					)
					errs[provider] = err
					// This is a funny case where we're returning an error
					// before we do any querying at all. To keep the event
					// stream consistent without introducing an extra event
					// type, we'll emit an artificial QueryPackagesBegin for
					// this provider before we indicate that it failed using
					// QueryPackagesFailure.
					if cb := evts.QueryPackagesBegin; cb != nil {
						cb(provider, versionConstraints, true)
					}
					if cb := evts.QueryPackagesFailure; cb != nil {
						cb(provider, err)
					}
					continue
				}
				acceptableVersions = versions.Only(lock.Version())
				locked[provider] = true
			}
		}
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
			cb(provider, reqs[provider], locked[provider])
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
		if locked[provider] {
			// This situation should be a rare one: it suggests that a
			// version was previously available but was yanked for some
			// reason.
			lock := locks.Provider(provider)
			err = fmt.Errorf("the previously-selected version %s is no longer available", lock.Version())
		} else {
			err = fmt.Errorf("no available releases match the given constraints %s", getproviders.VersionConstraintsString(reqs[provider]))
		}
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

		lock := locks.Provider(provider)
		var preferredHashes []getproviders.Hash
		if lock != nil && lock.Version() == version { // hash changes are expected if the version is also changing
			preferredHashes = lock.PreferredHashes()
		}

		// If our target directory already has the provider version that fulfills the lock file, carry on
		if installed := i.targetDir.ProviderVersion(provider, version); installed != nil {
			if len(preferredHashes) > 0 {
				if matches, _ := installed.MatchesAnyHash(preferredHashes); matches {
					if cb := evts.ProviderAlreadyInstalled; cb != nil {
						cb(provider, version)
					}
					continue
				}
			}
		}

		if i.globalCacheDir != nil {
			// Step 3a: If our global cache already has this version available then
			// we'll just link it in.
			if cached := i.globalCacheDir.ProviderVersion(provider, version); cached != nil {
				if cb := evts.LinkFromCacheBegin; cb != nil {
					cb(provider, version, i.globalCacheDir.baseDir)
				}
				if _, err := cached.ExecutableFile(); err != nil {
					err := fmt.Errorf("provider binary not found: %s", err)
					errs[provider] = err
					if cb := evts.LinkFromCacheFailure; cb != nil {
						cb(provider, version, err)
					}
					continue
				}

				err := i.targetDir.LinkFromOtherCache(cached, preferredHashes)
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
					errs[provider] = err
					if cb := evts.LinkFromCacheFailure; cb != nil {
						cb(provider, version, err)
					}
					continue
				}

				// The LinkFromOtherCache call above should've verified that
				// the package matches one of the hashes previously recorded,
				// if any. We'll now augment those hashes with one freshly
				// calculated from the package we just linked, which allows
				// the lock file to gradually transition to recording newer hash
				// schemes when they become available.
				var newHashes []getproviders.Hash
				if lock != nil && lock.Version() == version {
					// If the version we're installing is identical to the
					// one we previously locked then we'll keep all of the
					// hashes we saved previously and add to it. Otherwise
					// we'll be starting fresh, because each version has its
					// own set of packages and thus its own hashes.
					newHashes = append(newHashes, preferredHashes...)

					// NOTE: The behavior here is unfortunate when a particular
					// provider version was already cached on the first time
					// the current configuration requested it, because that
					// means we don't currently get the opportunity to fetch
					// and verify the checksums for the new package from
					// upstream. That's currently unavoidable because upstream
					// checksums are in the "ziphash" format and so we can't
					// verify them against our cache directory's unpacked
					// packages: we'd need to go fetch the package from the
					// origin and compare against it, which would defeat the
					// purpose of the global cache.
					//
					// If we fetch from upstream on the first encounter with
					// a particular provider then we'll end up in the other
					// codepath below where we're able to also include the
					// checksums from the origin registry.
				}
				newHash, err := cached.Hash()
				if err != nil {
					err := fmt.Errorf("after linking %s from provider cache at %s, failed to compute a checksum for it: %s", provider, i.globalCacheDir.baseDir, err)
					errs[provider] = err
					if cb := evts.LinkFromCacheFailure; cb != nil {
						cb(provider, version, err)
					}
					continue
				}
				// The hashes slice gets deduplicated in the lock file
				// implementation, so we don't worry about potentially
				// creating a duplicate here.
				newHashes = append(newHashes, newHash)
				locks.SetProvider(provider, version, reqs[provider], newHashes)

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
		authResult, err := installTo.InstallPackage(ctx, meta, preferredHashes)
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
		if _, err := new.ExecutableFile(); err != nil {
			err := fmt.Errorf("provider binary not found: %s", err)
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
			// We also don't do a hash check here because we already did that
			// as part of the installTo.InstallPackage call above.
			err := linkTo.LinkFromOtherCache(new, nil)
			if err != nil {
				errs[provider] = err
				if cb := evts.FetchPackageFailure; cb != nil {
					cb(provider, version, err)
				}
				continue
			}
		}
		authResults[provider] = authResult

		// The InstallPackage call above should've verified that
		// the package matches one of the hashes previously recorded,
		// if any. We'll now augment those hashes with a new set populated
		// with the hashes returned by the upstream source and from the
		// package we've just installed, which allows the lock file to
		// gradually transition to newer hash schemes when they become
		// available.
		//
		// This is assuming that if a package matches both a hash we saw before
		// _and_ a new hash then the new hash is a valid substitute for
		// the previous hash.
		//
		// The hashes slice gets deduplicated in the lock file
		// implementation, so we don't worry about potentially
		// creating duplicates here.
		var newHashes []getproviders.Hash
		if lock != nil && lock.Version() == version {
			// If the version we're installing is identical to the
			// one we previously locked then we'll keep all of the
			// hashes we saved previously and add to it. Otherwise
			// we'll be starting fresh, because each version has its
			// own set of packages and thus its own hashes.
			newHashes = append(newHashes, preferredHashes...)
		}
		newHash, err := new.Hash()
		if err != nil {
			err := fmt.Errorf("after installing %s, failed to compute a checksum for it: %s", provider, err)
			errs[provider] = err
			if cb := evts.FetchPackageFailure; cb != nil {
				cb(provider, version, err)
			}
			continue
		}
		newHashes = append(newHashes, newHash)
		if authResult.SignedByAnyParty() {
			// We'll trust new hashes from upstream only if they were verified
			// as signed by a suitable key. Otherwise, we'd record only
			// a new hash we just calculated ourselves from the bytes on disk,
			// and so the hashes would cover only the current platform.
			newHashes = append(newHashes, meta.AcceptableHashes()...)
		}
		locks.SetProvider(provider, version, reqs[provider], newHashes)

		if cb := evts.FetchPackageSuccess; cb != nil {
			cb(provider, version, new.PackageDir, authResult)
		}
	}

	// Emit final event for fetching if any were successfully fetched
	if cb := evts.ProvidersFetched; cb != nil && len(authResults) > 0 {
		cb(authResults)
	}

	if len(errs) > 0 {
		return locks, InstallerError{
			ProviderErrors: errs,
		}
	}
	return locks, nil
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
	return strings.TrimSpace(b.String())
}
