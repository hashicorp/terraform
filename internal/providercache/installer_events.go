package providercache

import (
	"context"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// InstallerEvents is a collection of function references that can be
// associated with an Installer object in order to be notified about various
// installation lifecycle events during an install operation.
//
// The set of supported events is primarily motivated by allowing ongoing
// progress reports in the UI of the command running provider installation,
// and so this only exposes information interesting to display and does not
// allow the recipient of the events to influence the ongoing process.
//
// Any of the fields may be left as nil to signal that the caller is not
// interested in the associated event. It's better to leave a field set to
// nil than to assign a do-nothing function into it because the installer
// may choose to skip preparing certain temporary data structures if it can see
// that a particular event is not used.
type InstallerEvents struct {
	// The PendingProviders event is called prior to other events to give
	// the recipient prior notice of the full set of distinct provider
	// addresses it can expect to see mentioned in the other events.
	//
	// A recipient driving a UI might, for example, use this to pre-allocate
	// UI space for status reports for all of the providers and then update
	// those positions in-place as other events arrive.
	PendingProviders func(reqs map[addrs.Provider]getproviders.VersionConstraints)

	// ProviderAlreadyInstalled is called for any provider that was included
	// in PendingProviders but requires no further action because a suitable
	// version is already present in the local provider cache directory.
	//
	// This event can also appear after the QueryPackages... series if
	// querying determines that a version already available is the newest
	// available version.
	ProviderAlreadyInstalled func(provider addrs.Provider, selectedVersion getproviders.Version)

	// The BuiltInProvider... family of events describe the outcome for any
	// requested providers that are built in to Terraform. Only one of these
	// methods will be called for each such provider, and no other method
	// will be called for them except that they are included in the
	// aggregate PendingProviders map.
	//
	// The "Available" event reports that the requested builtin provider is
	// available in this release of Terraform. The "Failure" event reports
	// either that the provider is unavailable or that the request for it
	// is invalid somehow.
	BuiltInProviderAvailable func(provider addrs.Provider)
	BuiltInProviderFailure   func(provider addrs.Provider, err error)

	// The QueryPackages... family of events delimit the operation of querying
	// a provider source for information about available packages matching
	// a particular version constraint, prior to selecting a single version
	// to install.
	//
	// A particular install operation includes only one query per distinct
	// provider, so a caller can use the provider argument as a unique
	// identifier to correlate between successive events.
	//
	// The Begin, Success, and Failure events will each occur only once per
	// distinct provider.
	//
	// The Warning event is unique to the registry source
	QueryPackagesBegin   func(provider addrs.Provider, versionConstraints getproviders.VersionConstraints, locked bool)
	QueryPackagesSuccess func(provider addrs.Provider, selectedVersion getproviders.Version)
	QueryPackagesFailure func(provider addrs.Provider, err error)
	QueryPackagesWarning func(provider addrs.Provider, warn []string)

	// The LinkFromCache... family of events delimit the operation of linking
	// a selected provider package from the system-wide shared cache into the
	// current configuration's local cache.
	//
	// This sequence occurs instead of the FetchPackage... sequence if the
	// QueryPackages... sequence selects a version that is already in the
	// system-wide cache, and thus we will skip fetching it from the
	// originating provider source and take it from the shared cache instead.
	//
	// Linking should, in most cases, be a much faster operation than
	// fetching. However, it could still potentially be slow in some unusual
	// cases like a particularly large source package on a system where symlinks
	// are impossible, or when either of the cache directories are on a network
	// filesystem accessed over a slow link.
	LinkFromCacheBegin   func(provider addrs.Provider, version getproviders.Version, cacheRoot string)
	LinkFromCacheSuccess func(provider addrs.Provider, version getproviders.Version, localDir string)
	LinkFromCacheFailure func(provider addrs.Provider, version getproviders.Version, err error)

	// The FetchPackage... family of events delimit the operation of retrieving
	// a package from a particular source location.
	//
	// A particular install operation includes only one fetch per distinct
	// provider, so a caller can use the provider argument as a unique
	// identifier to correlate between successive events.
	//
	// A particular provider will either notify the LinkFromCache... events
	// or the FetchPackage... events, never both in the same install operation.
	//
	// The Query, Begin, Success, and Failure events will each occur only once
	// per distinct provider.
	FetchPackageMeta    func(provider addrs.Provider, version getproviders.Version) // fetching metadata prior to real download
	FetchPackageBegin   func(provider addrs.Provider, version getproviders.Version, location getproviders.PackageLocation)
	FetchPackageSuccess func(provider addrs.Provider, version getproviders.Version, localDir string, authResult *getproviders.PackageAuthenticationResult)
	FetchPackageFailure func(provider addrs.Provider, version getproviders.Version, err error)

	// The ProvidersFetched event is called after all fetch operations if at
	// least one provider was fetched successfully.
	ProvidersFetched func(authResults map[addrs.Provider]*getproviders.PackageAuthenticationResult)

	// HashPackageFailure is called if the installer is unable to determine
	// the hash of the contents of an installed package after installation.
	// In that case, the selection will not be recorded in the target cache
	// directory's lock file.
	HashPackageFailure func(provider addrs.Provider, version getproviders.Version, err error)
}

// OnContext produces a context with all of the same behaviors as the given
// context except that it will additionally carry the receiving
// InstallerEvents.
//
// Passing the resulting context to an installer request will cause the
// installer to send event notifications via the callbacks inside.
func (e *InstallerEvents) OnContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxInstallerEvents, e)
}

// installerEventsForContext looks on the given context for a registered
// InstallerEvents and returns a pointer to it if so.
//
// For caller convenience, if there is no events object attached to the
// given context this function will construct one that has all of its
// fields set to nil and return that, freeing the caller from having to
// do a nil check on the result before dereferencing it.
func installerEventsForContext(ctx context.Context) *InstallerEvents {
	v := ctx.Value(ctxInstallerEvents)
	if v != nil {
		return v.(*InstallerEvents)
	}
	return &InstallerEvents{}
}

type ctxInstallerEventsType int

const ctxInstallerEvents = ctxInstallerEventsType(0)
