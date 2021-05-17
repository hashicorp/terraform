package providercache

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

type testInstallerEventLogItem struct {
	// The name of the event that occurred, using the same names as the
	// fields of InstallerEvents.
	Event string

	// Most events relate to a specific provider. For the few event types
	// that don't, this will be a zero-value Provider.
	Provider addrs.Provider

	// The type of Args will vary by event, but it should always be something
	// that can be deterministically compared using the go-cmp package.
	Args interface{}
}

// installerLogEventsForTests is a test helper that produces an InstallerEvents
// that writes event notifications (*testInstallerEventLogItem values) to
// the given channel as they occur.
//
// The caller must keep reading from the read side of the given channel
// throughout any installer operation using the returned InstallerEvents.
// It's the caller's responsibility to close the channel if needed and
// clean up any goroutines it started to process the events.
//
// The exact sequence of events emitted for an installer operation might
// change in future, if e.g. we introduce new event callbacks to the
// InstallerEvents struct. Tests using this mechanism may therefore need to
// be updated to reflect such changes.
//
// (The channel-based approach here is so that the control flow for event
// processing will belong to the caller and thus it can safely use its
// testing.T object(s) to emit log lines without non-test-case frames in the
// call stack.)
func installerLogEventsForTests(into chan<- *testInstallerEventLogItem) *InstallerEvents {
	return &InstallerEvents{
		PendingProviders: func(reqs map[addrs.Provider]getproviders.VersionConstraints) {
			into <- &testInstallerEventLogItem{
				Event: "PendingProviders",
				Args:  reqs,
			}
		},
		ProviderAlreadyInstalled: func(provider addrs.Provider, selectedVersion getproviders.Version) {
			into <- &testInstallerEventLogItem{
				Event:    "ProviderAlreadyInstalled",
				Provider: provider,
				Args:     selectedVersion,
			}
		},
		BuiltInProviderAvailable: func(provider addrs.Provider) {
			into <- &testInstallerEventLogItem{
				Event:    "BuiltInProviderAvailable",
				Provider: provider,
			}
		},
		BuiltInProviderFailure: func(provider addrs.Provider, err error) {
			into <- &testInstallerEventLogItem{
				Event:    "BuiltInProviderFailure",
				Provider: provider,
				Args:     err.Error(), // stringified to guarantee cmp-ability
			}
		},
		QueryPackagesBegin: func(provider addrs.Provider, versionConstraints getproviders.VersionConstraints, locked bool) {
			into <- &testInstallerEventLogItem{
				Event:    "QueryPackagesBegin",
				Provider: provider,
				Args: struct {
					Constraints string
					Locked      bool
				}{getproviders.VersionConstraintsString(versionConstraints), locked},
			}
		},
		QueryPackagesSuccess: func(provider addrs.Provider, selectedVersion getproviders.Version) {
			into <- &testInstallerEventLogItem{
				Event:    "QueryPackagesSuccess",
				Provider: provider,
				Args:     selectedVersion.String(),
			}
		},
		QueryPackagesFailure: func(provider addrs.Provider, err error) {
			into <- &testInstallerEventLogItem{
				Event:    "QueryPackagesFailure",
				Provider: provider,
				Args:     err.Error(), // stringified to guarantee cmp-ability
			}
		},
		QueryPackagesWarning: func(provider addrs.Provider, warns []string) {
			into <- &testInstallerEventLogItem{
				Event:    "QueryPackagesWarning",
				Provider: provider,
				Args:     warns,
			}
		},
		LinkFromCacheBegin: func(provider addrs.Provider, version getproviders.Version, cacheRoot string) {
			into <- &testInstallerEventLogItem{
				Event:    "LinkFromCacheBegin",
				Provider: provider,
				Args: struct {
					Version   string
					CacheRoot string
				}{version.String(), cacheRoot},
			}
		},
		LinkFromCacheSuccess: func(provider addrs.Provider, version getproviders.Version, localDir string) {
			into <- &testInstallerEventLogItem{
				Event:    "LinkFromCacheSuccess",
				Provider: provider,
				Args: struct {
					Version  string
					LocalDir string
				}{version.String(), localDir},
			}
		},
		LinkFromCacheFailure: func(provider addrs.Provider, version getproviders.Version, err error) {
			into <- &testInstallerEventLogItem{
				Event:    "LinkFromCacheFailure",
				Provider: provider,
				Args: struct {
					Version string
					Error   string
				}{version.String(), err.Error()},
			}
		},
		FetchPackageMeta: func(provider addrs.Provider, version getproviders.Version) {
			into <- &testInstallerEventLogItem{
				Event:    "FetchPackageMeta",
				Provider: provider,
				Args:     version.String(),
			}
		},
		FetchPackageBegin: func(provider addrs.Provider, version getproviders.Version, location getproviders.PackageLocation) {
			into <- &testInstallerEventLogItem{
				Event:    "FetchPackageBegin",
				Provider: provider,
				Args: struct {
					Version  string
					Location getproviders.PackageLocation
				}{version.String(), location},
			}
		},
		FetchPackageSuccess: func(provider addrs.Provider, version getproviders.Version, localDir string, authResult *getproviders.PackageAuthenticationResult) {
			into <- &testInstallerEventLogItem{
				Event:    "FetchPackageSuccess",
				Provider: provider,
				Args: struct {
					Version    string
					LocalDir   string
					AuthResult string
				}{version.String(), localDir, authResult.String()},
			}
		},
		FetchPackageFailure: func(provider addrs.Provider, version getproviders.Version, err error) {
			into <- &testInstallerEventLogItem{
				Event:    "FetchPackageFailure",
				Provider: provider,
				Args: struct {
					Version string
					Error   string
				}{version.String(), err.Error()},
			}
		},
		ProvidersFetched: func(authResults map[addrs.Provider]*getproviders.PackageAuthenticationResult) {
			into <- &testInstallerEventLogItem{
				Event: "ProvidersFetched",
				Args:  authResults,
			}
		},
		HashPackageFailure: func(provider addrs.Provider, version getproviders.Version, err error) {
			into <- &testInstallerEventLogItem{
				Event:    "HashPackageFailure",
				Provider: provider,
				Args: struct {
					Version string
					Error   string
				}{version.String(), err.Error()},
			}
		},
	}
}
