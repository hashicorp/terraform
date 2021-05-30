package providercache

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/apparentlymart/go-versions/versions/constraints"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestEnsureProviderVersions(t *testing.T) {
	// This is a sort of hybrid between table-driven and imperative-style
	// testing, because the overall sequence of steps is the same for all
	// of the test cases but the setup and verification have enough different
	// permutations that it ends up being more concise to express them as
	// normal code.
	type Test struct {
		Source     getproviders.Source
		Prepare    func(*testing.T, *Installer, *Dir)
		LockFile   string
		Reqs       getproviders.Requirements
		Mode       InstallMode
		Check      func(*testing.T, *Dir, *depsfile.Locks)
		WantErr    string
		WantEvents func(*Installer, *Dir) map[addrs.Provider][]*testInstallerEventLogItem
	}

	// noProvider is just the zero value of addrs.Provider, which we're
	// using in this test as the key for installer events that are not
	// specific to a particular provider.
	var noProvider addrs.Provider
	beepProvider := addrs.MustParseProviderSourceString("example.com/foo/beep")
	beepProviderDir := getproviders.PackageLocalDir("testdata/beep-provider")
	fakePlatform := getproviders.Platform{OS: "bleep", Arch: "bloop"}
	wrongPlatform := getproviders.Platform{OS: "wrong", Arch: "wrong"}
	beepProviderHash := getproviders.HashScheme1.New("2y06Ykj0FRneZfGCTxI9wRTori8iB7ZL5kQ6YyEnh84=")
	terraformProvider := addrs.MustParseProviderSourceString("terraform.io/builtin/terraform")

	tests := map[string]Test{
		"no dependencies": {
			Mode: InstallNewProvidersOnly,
			Check: func(t *testing.T, dir *Dir, locks *depsfile.Locks) {
				if allCached := dir.AllAvailablePackages(); len(allCached) != 0 {
					t.Errorf("unexpected cache directory entries\n%s", spew.Sdump(allCached))
				}
				if allLocked := locks.AllProviders(); len(allLocked) != 0 {
					t.Errorf("unexpected provider lock entries\n%s", spew.Sdump(allLocked))
				}
			},
			WantEvents: func(*Installer, *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args:  map[addrs.Provider]getproviders.VersionConstraints(nil),
						},
					},
				}
			},
		},
		"successful initial install of one provider": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("1.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.1.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
				},
				nil,
			),
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
			},
			Check: func(t *testing.T, dir *Dir, locks *depsfile.Locks) {
				if allCached := dir.AllAvailablePackages(); len(allCached) != 1 {
					t.Errorf("wrong number of cache directory entries; want only one\n%s", spew.Sdump(allCached))
				}
				if allLocked := locks.AllProviders(); len(allLocked) != 1 {
					t.Errorf("wrong number of provider lock entries; want only one\n%s", spew.Sdump(allLocked))
				}

				gotLock := locks.Provider(beepProvider)
				wantLock := depsfile.NewProviderLock(
					beepProvider,
					getproviders.MustParseVersion("2.1.0"),
					getproviders.MustParseVersionConstraints(">= 2.0.0"),
					[]getproviders.Hash{beepProviderHash},
				)
				if diff := cmp.Diff(wantLock, gotLock, depsfile.ProviderLockComparer); diff != "" {
					t.Errorf("wrong lock entry\n%s", diff)
				}

				gotEntry := dir.ProviderLatestVersion(beepProvider)
				wantEntry := &CachedProvider{
					Provider:   beepProvider,
					Version:    getproviders.MustParseVersion("2.1.0"),
					PackageDir: filepath.Join(dir.BasePath(), "example.com/foo/beep/2.1.0/bleep_bloop"),
				}
				if diff := cmp.Diff(wantEntry, gotEntry); diff != "" {
					t.Errorf("wrong cache entry\n%s", diff)
				}
			},
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
							},
						},
						{
							Event: "ProvidersFetched",
							Args: map[addrs.Provider]*getproviders.PackageAuthenticationResult{
								beepProvider: nil,
							},
						},
					},
					beepProvider: {
						{
							Event:    "QueryPackagesBegin",
							Provider: beepProvider,
							Args: struct {
								Constraints string
								Locked      bool
							}{">= 2.0.0", false},
						},
						{
							Event:    "QueryPackagesSuccess",
							Provider: beepProvider,
							Args:     "2.1.0",
						},
						{
							Event:    "FetchPackageMeta",
							Provider: beepProvider,
							Args:     "2.1.0",
						},
						{
							Event:    "FetchPackageBegin",
							Provider: beepProvider,
							Args: struct {
								Version  string
								Location getproviders.PackageLocation
							}{"2.1.0", beepProviderDir},
						},
						{
							Event:    "FetchPackageSuccess",
							Provider: beepProvider,
							Args: struct {
								Version    string
								LocalDir   string
								AuthResult string
							}{
								"2.1.0",
								filepath.Join(dir.BasePath(), "example.com/foo/beep/2.1.0/bleep_bloop"),
								"unauthenticated",
							},
						},
					},
				}
			},
		},
		"successful initial install of one provider through a cold global cache": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.1.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
				},
				nil,
			),
			Prepare: func(t *testing.T, inst *Installer, dir *Dir) {
				globalCacheDirPath := tmpDir(t)
				globalCacheDir := NewDirWithPlatform(globalCacheDirPath, fakePlatform)
				inst.SetGlobalCacheDir(globalCacheDir)
			},
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
			},
			Check: func(t *testing.T, dir *Dir, locks *depsfile.Locks) {
				if allCached := dir.AllAvailablePackages(); len(allCached) != 1 {
					t.Errorf("wrong number of cache directory entries; want only one\n%s", spew.Sdump(allCached))
				}
				if allLocked := locks.AllProviders(); len(allLocked) != 1 {
					t.Errorf("wrong number of provider lock entries; want only one\n%s", spew.Sdump(allLocked))
				}

				gotLock := locks.Provider(beepProvider)
				wantLock := depsfile.NewProviderLock(
					beepProvider,
					getproviders.MustParseVersion("2.1.0"),
					getproviders.MustParseVersionConstraints(">= 2.0.0"),
					[]getproviders.Hash{beepProviderHash},
				)
				if diff := cmp.Diff(wantLock, gotLock, depsfile.ProviderLockComparer); diff != "" {
					t.Errorf("wrong lock entry\n%s", diff)
				}

				gotEntry := dir.ProviderLatestVersion(beepProvider)
				wantEntry := &CachedProvider{
					Provider:   beepProvider,
					Version:    getproviders.MustParseVersion("2.1.0"),
					PackageDir: filepath.Join(dir.BasePath(), "example.com/foo/beep/2.1.0/bleep_bloop"),
				}
				if diff := cmp.Diff(wantEntry, gotEntry); diff != "" {
					t.Errorf("wrong cache entry\n%s", diff)
				}
			},
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
							},
						},
						{
							Event: "ProvidersFetched",
							Args: map[addrs.Provider]*getproviders.PackageAuthenticationResult{
								beepProvider: nil,
							},
						},
					},
					beepProvider: {
						{
							Event:    "QueryPackagesBegin",
							Provider: beepProvider,
							Args: struct {
								Constraints string
								Locked      bool
							}{">= 2.0.0", false},
						},
						{
							Event:    "QueryPackagesSuccess",
							Provider: beepProvider,
							Args:     "2.1.0",
						},
						{
							Event:    "FetchPackageMeta",
							Provider: beepProvider,
							Args:     "2.1.0",
						},
						{
							Event:    "FetchPackageBegin",
							Provider: beepProvider,
							Args: struct {
								Version  string
								Location getproviders.PackageLocation
							}{"2.1.0", beepProviderDir},
						},
						{
							Event:    "FetchPackageSuccess",
							Provider: beepProvider,
							Args: struct {
								Version    string
								LocalDir   string
								AuthResult string
							}{
								"2.1.0",
								// NOTE: With global cache enabled, the fetch
								// goes into the global cache dir and
								// we then to it from the local cache dir.
								filepath.Join(inst.globalCacheDir.BasePath(), "example.com/foo/beep/2.1.0/bleep_bloop"),
								"unauthenticated",
							},
						},
					},
				}
			},
		},
		"successful initial install of one provider through a warm global cache": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.1.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
				},
				nil,
			),
			Prepare: func(t *testing.T, inst *Installer, dir *Dir) {
				globalCacheDirPath := tmpDir(t)
				globalCacheDir := NewDirWithPlatform(globalCacheDirPath, fakePlatform)
				_, err := globalCacheDir.InstallPackage(
					context.Background(),
					getproviders.PackageMeta{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.1.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					nil,
				)
				if err != nil {
					t.Fatalf("failed to populate global cache: %s", err)
				}
				inst.SetGlobalCacheDir(globalCacheDir)
			},
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
			},
			Check: func(t *testing.T, dir *Dir, locks *depsfile.Locks) {
				if allCached := dir.AllAvailablePackages(); len(allCached) != 1 {
					t.Errorf("wrong number of cache directory entries; want only one\n%s", spew.Sdump(allCached))
				}
				if allLocked := locks.AllProviders(); len(allLocked) != 1 {
					t.Errorf("wrong number of provider lock entries; want only one\n%s", spew.Sdump(allLocked))
				}

				gotLock := locks.Provider(beepProvider)
				wantLock := depsfile.NewProviderLock(
					beepProvider,
					getproviders.MustParseVersion("2.1.0"),
					getproviders.MustParseVersionConstraints(">= 2.0.0"),
					[]getproviders.Hash{beepProviderHash},
				)
				if diff := cmp.Diff(wantLock, gotLock, depsfile.ProviderLockComparer); diff != "" {
					t.Errorf("wrong lock entry\n%s", diff)
				}

				gotEntry := dir.ProviderLatestVersion(beepProvider)
				wantEntry := &CachedProvider{
					Provider:   beepProvider,
					Version:    getproviders.MustParseVersion("2.1.0"),
					PackageDir: filepath.Join(dir.BasePath(), "example.com/foo/beep/2.1.0/bleep_bloop"),
				}
				if diff := cmp.Diff(wantEntry, gotEntry); diff != "" {
					t.Errorf("wrong cache entry\n%s", diff)
				}
			},
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
							},
						},
					},
					beepProvider: {
						{
							Event:    "QueryPackagesBegin",
							Provider: beepProvider,
							Args: struct {
								Constraints string
								Locked      bool
							}{">= 2.0.0", false},
						},
						{
							Event:    "QueryPackagesSuccess",
							Provider: beepProvider,
							Args:     "2.1.0",
						},
						{
							Event:    "LinkFromCacheBegin",
							Provider: beepProvider,
							Args: struct {
								Version   string
								CacheRoot string
							}{
								"2.1.0",
								inst.globalCacheDir.BasePath(),
							},
						},
						{
							Event:    "LinkFromCacheSuccess",
							Provider: beepProvider,
							Args: struct {
								Version  string
								LocalDir string
							}{
								"2.1.0",
								filepath.Join(dir.BasePath(), "/example.com/foo/beep/2.1.0/bleep_bloop"),
							},
						},
					},
				}
			},
		},
		"successful reinstall of one previously-locked provider": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("1.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.1.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
				},
				nil,
			),
			LockFile: `
				provider "example.com/foo/beep" {
					version     = "2.0.0"
					constraints = ">= 2.0.0"
					hashes = [
						"h1:2y06Ykj0FRneZfGCTxI9wRTori8iB7ZL5kQ6YyEnh84=",
					]
				}
			`,
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
			},
			Check: func(t *testing.T, dir *Dir, locks *depsfile.Locks) {
				if allCached := dir.AllAvailablePackages(); len(allCached) != 1 {
					t.Errorf("wrong number of cache directory entries; want only one\n%s", spew.Sdump(allCached))
				}
				if allLocked := locks.AllProviders(); len(allLocked) != 1 {
					t.Errorf("wrong number of provider lock entries; want only one\n%s", spew.Sdump(allLocked))
				}

				gotLock := locks.Provider(beepProvider)
				wantLock := depsfile.NewProviderLock(
					beepProvider,
					getproviders.MustParseVersion("2.0.0"),
					getproviders.MustParseVersionConstraints(">= 2.0.0"),
					[]getproviders.Hash{"h1:2y06Ykj0FRneZfGCTxI9wRTori8iB7ZL5kQ6YyEnh84="},
				)
				if diff := cmp.Diff(wantLock, gotLock, depsfile.ProviderLockComparer); diff != "" {
					t.Errorf("wrong lock entry\n%s", diff)
				}

				gotEntry := dir.ProviderLatestVersion(beepProvider)
				wantEntry := &CachedProvider{
					Provider:   beepProvider,
					Version:    getproviders.MustParseVersion("2.0.0"),
					PackageDir: filepath.Join(dir.BasePath(), "example.com/foo/beep/2.0.0/bleep_bloop"),
				}
				if diff := cmp.Diff(wantEntry, gotEntry); diff != "" {
					t.Errorf("wrong cache entry\n%s", diff)
				}
			},
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
							},
						},
						{
							Event: "ProvidersFetched",
							Args: map[addrs.Provider]*getproviders.PackageAuthenticationResult{
								beepProvider: nil,
							},
						},
					},
					beepProvider: {
						{
							Event:    "QueryPackagesBegin",
							Provider: beepProvider,
							Args: struct {
								Constraints string
								Locked      bool
							}{">= 2.0.0", true},
						},
						{
							Event:    "QueryPackagesSuccess",
							Provider: beepProvider,
							Args:     "2.0.0",
						},
						{
							Event:    "FetchPackageMeta",
							Provider: beepProvider,
							Args:     "2.0.0",
						},
						{
							Event:    "FetchPackageBegin",
							Provider: beepProvider,
							Args: struct {
								Version  string
								Location getproviders.PackageLocation
							}{"2.0.0", beepProviderDir},
						},
						{
							Event:    "FetchPackageSuccess",
							Provider: beepProvider,
							Args: struct {
								Version    string
								LocalDir   string
								AuthResult string
							}{
								"2.0.0",
								filepath.Join(dir.BasePath(), "example.com/foo/beep/2.0.0/bleep_bloop"),
								"unauthenticated",
							},
						},
					},
				}
			},
		},
		"skipped install of one previously-locked and installed provider": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
				},
				nil,
			),
			LockFile: `
				provider "example.com/foo/beep" {
					version     = "2.0.0"
					constraints = ">= 2.0.0"
					hashes = [
						"h1:2y06Ykj0FRneZfGCTxI9wRTori8iB7ZL5kQ6YyEnh84=",
					]
				}
			`,
			Prepare: func(t *testing.T, inst *Installer, dir *Dir) {
				_, err := dir.InstallPackage(
					context.Background(),
					getproviders.PackageMeta{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					nil,
				)
				if err != nil {
					t.Fatalf("installation to the test dir failed: %s", err)
				}
			},
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
			},
			Check: func(t *testing.T, dir *Dir, locks *depsfile.Locks) {
				if allCached := dir.AllAvailablePackages(); len(allCached) != 1 {
					t.Errorf("wrong number of cache directory entries; want only one\n%s", spew.Sdump(allCached))
				}
				if allLocked := locks.AllProviders(); len(allLocked) != 1 {
					t.Errorf("wrong number of provider lock entries; want only one\n%s", spew.Sdump(allLocked))
				}

				gotLock := locks.Provider(beepProvider)
				wantLock := depsfile.NewProviderLock(
					beepProvider,
					getproviders.MustParseVersion("2.0.0"),
					getproviders.MustParseVersionConstraints(">= 2.0.0"),
					[]getproviders.Hash{"h1:2y06Ykj0FRneZfGCTxI9wRTori8iB7ZL5kQ6YyEnh84="},
				)
				if diff := cmp.Diff(wantLock, gotLock, depsfile.ProviderLockComparer); diff != "" {
					t.Errorf("wrong lock entry\n%s", diff)
				}

				gotEntry := dir.ProviderLatestVersion(beepProvider)
				wantEntry := &CachedProvider{
					Provider:   beepProvider,
					Version:    getproviders.MustParseVersion("2.0.0"),
					PackageDir: filepath.Join(dir.BasePath(), "example.com/foo/beep/2.0.0/bleep_bloop"),
				}
				if diff := cmp.Diff(wantEntry, gotEntry); diff != "" {
					t.Errorf("wrong cache entry\n%s", diff)
				}
			},
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
							},
						},
					},
					beepProvider: {
						{
							Event:    "QueryPackagesBegin",
							Provider: beepProvider,
							Args: struct {
								Constraints string
								Locked      bool
							}{">= 2.0.0", true},
						},
						{
							Event:    "QueryPackagesSuccess",
							Provider: beepProvider,
							Args:     "2.0.0",
						},
						{
							Event:    "ProviderAlreadyInstalled",
							Provider: beepProvider,
							Args:     versions.Version{Major: 2, Minor: 0, Patch: 0},
						},
					},
				}
			},
		},
		"successful upgrade of one previously-locked provider": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("1.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.1.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
				},
				nil,
			),
			LockFile: `
				provider "example.com/foo/beep" {
					version     = "2.0.0"
					constraints = ">= 2.0.0"
					hashes = [
						"h1:2y06Ykj0FRneZfGCTxI9wRTori8iB7ZL5kQ6YyEnh84=",
					]
				}
			`,
			Mode: InstallUpgrades,
			Reqs: getproviders.Requirements{
				beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
			},
			Check: func(t *testing.T, dir *Dir, locks *depsfile.Locks) {
				if allCached := dir.AllAvailablePackages(); len(allCached) != 1 {
					t.Errorf("wrong number of cache directory entries; want only one\n%s", spew.Sdump(allCached))
				}
				if allLocked := locks.AllProviders(); len(allLocked) != 1 {
					t.Errorf("wrong number of provider lock entries; want only one\n%s", spew.Sdump(allLocked))
				}

				gotLock := locks.Provider(beepProvider)
				wantLock := depsfile.NewProviderLock(
					beepProvider,
					getproviders.MustParseVersion("2.1.0"),
					getproviders.MustParseVersionConstraints(">= 2.0.0"),
					[]getproviders.Hash{"h1:2y06Ykj0FRneZfGCTxI9wRTori8iB7ZL5kQ6YyEnh84="},
				)
				if diff := cmp.Diff(wantLock, gotLock, depsfile.ProviderLockComparer); diff != "" {
					t.Errorf("wrong lock entry\n%s", diff)
				}

				gotEntry := dir.ProviderLatestVersion(beepProvider)
				wantEntry := &CachedProvider{
					Provider:   beepProvider,
					Version:    getproviders.MustParseVersion("2.1.0"),
					PackageDir: filepath.Join(dir.BasePath(), "example.com/foo/beep/2.1.0/bleep_bloop"),
				}
				if diff := cmp.Diff(wantEntry, gotEntry); diff != "" {
					t.Errorf("wrong cache entry\n%s", diff)
				}
			},
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
							},
						},
						{
							Event: "ProvidersFetched",
							Args: map[addrs.Provider]*getproviders.PackageAuthenticationResult{
								beepProvider: nil,
							},
						},
					},
					beepProvider: {
						{
							Event:    "QueryPackagesBegin",
							Provider: beepProvider,
							Args: struct {
								Constraints string
								Locked      bool
							}{">= 2.0.0", false},
						},
						{
							Event:    "QueryPackagesSuccess",
							Provider: beepProvider,
							Args:     "2.1.0",
						},
						{
							Event:    "FetchPackageMeta",
							Provider: beepProvider,
							Args:     "2.1.0",
						},
						{
							Event:    "FetchPackageBegin",
							Provider: beepProvider,
							Args: struct {
								Version  string
								Location getproviders.PackageLocation
							}{"2.1.0", beepProviderDir},
						},
						{
							Event:    "FetchPackageSuccess",
							Provider: beepProvider,
							Args: struct {
								Version    string
								LocalDir   string
								AuthResult string
							}{
								"2.1.0",
								filepath.Join(dir.BasePath(), "example.com/foo/beep/2.1.0/bleep_bloop"),
								"unauthenticated",
							},
						},
					},
				}
			},
		},
		"successful install of a built-in provider": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{},
				nil,
			),
			Prepare: func(t *testing.T, inst *Installer, dir *Dir) {
				inst.SetBuiltInProviderTypes([]string{"terraform"})
			},
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				terraformProvider: nil,
			},
			Check: func(t *testing.T, dir *Dir, locks *depsfile.Locks) {
				// Built-in providers are neither included in the cache
				// directory nor mentioned in the lock file, because they
				// are compiled directly into the Terraform executable.
				if allCached := dir.AllAvailablePackages(); len(allCached) != 0 {
					t.Errorf("wrong number of cache directory entries; want none\n%s", spew.Sdump(allCached))
				}
				if allLocked := locks.AllProviders(); len(allLocked) != 0 {
					t.Errorf("wrong number of provider lock entries; want none\n%s", spew.Sdump(allLocked))
				}
			},
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								terraformProvider: constraints.IntersectionSpec(nil),
							},
						},
					},
					terraformProvider: {
						{
							Event:    "BuiltInProviderAvailable",
							Provider: terraformProvider,
						},
					},
				}
			},
		},
		"failed install of a non-existing built-in provider": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{},
				nil,
			),
			Prepare: func(t *testing.T, inst *Installer, dir *Dir) {
				// NOTE: We're intentionally not calling
				// inst.SetBuiltInProviderTypes to make the "terraform"
				// built-in provider available here, so requests for it
				// should fail.
			},
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				terraformProvider: nil,
			},
			WantErr: `some providers could not be installed:
- terraform.io/builtin/terraform: this Terraform release has no built-in provider named "terraform"`,
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								terraformProvider: constraints.IntersectionSpec(nil),
							},
						},
					},
					terraformProvider: {
						{
							Event:    "BuiltInProviderFailure",
							Provider: terraformProvider,
							Args:     `this Terraform release has no built-in provider named "terraform"`,
						},
					},
				}
			},
		},
		"failed install when a built-in provider has a version constraint": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{},
				nil,
			),
			Prepare: func(t *testing.T, inst *Installer, dir *Dir) {
				inst.SetBuiltInProviderTypes([]string{"terraform"})
			},
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				terraformProvider: getproviders.MustParseVersionConstraints(">= 1.0.0"),
			},
			WantErr: `some providers could not be installed:
- terraform.io/builtin/terraform: built-in providers do not support explicit version constraints`,
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								terraformProvider: getproviders.MustParseVersionConstraints(">= 1.0.0"),
							},
						},
					},
					terraformProvider: {
						{
							Event:    "BuiltInProviderFailure",
							Provider: terraformProvider,
							Args:     `built-in providers do not support explicit version constraints`,
						},
					},
				}
			},
		},
		"locked version is excluded by new version constraint": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("1.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
				},
				nil,
			),
			LockFile: `
				provider "example.com/foo/beep" {
					version     = "1.0.0"
					constraints = ">= 1.0.0"
					hashes = [
						"h1:2y06Ykj0FRneZfGCTxI9wRTori8iB7ZL5kQ6YyEnh84=",
					]
				}
			`,
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
			},
			Check: func(t *testing.T, dir *Dir, locks *depsfile.Locks) {
				if allCached := dir.AllAvailablePackages(); len(allCached) != 0 {
					t.Errorf("wrong number of cache directory entries; want none\n%s", spew.Sdump(allCached))
				}
				if allLocked := locks.AllProviders(); len(allLocked) != 1 {
					t.Errorf("wrong number of provider lock entries; want only one\n%s", spew.Sdump(allLocked))
				}

				gotLock := locks.Provider(beepProvider)
				wantLock := depsfile.NewProviderLock(
					beepProvider,
					getproviders.MustParseVersion("1.0.0"),
					getproviders.MustParseVersionConstraints(">= 1.0.0"),
					[]getproviders.Hash{"h1:2y06Ykj0FRneZfGCTxI9wRTori8iB7ZL5kQ6YyEnh84="},
				)
				if diff := cmp.Diff(wantLock, gotLock, depsfile.ProviderLockComparer); diff != "" {
					t.Errorf("wrong lock entry\n%s", diff)
				}
			},
			WantErr: `some providers could not be installed:
- example.com/foo/beep: locked provider example.com/foo/beep 1.0.0 does not match configured version constraint >= 2.0.0; must use terraform init -upgrade to allow selection of new versions`,
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
							},
						},
					},
					beepProvider: {
						{
							Event:    "QueryPackagesBegin",
							Provider: beepProvider,
							Args: struct {
								Constraints string
								Locked      bool
							}{">= 2.0.0", true},
						},
						{
							Event:    "QueryPackagesFailure",
							Provider: beepProvider,
							Args:     `locked provider example.com/foo/beep 1.0.0 does not match configured version constraint >= 2.0.0; must use terraform init -upgrade to allow selection of new versions`,
						},
					},
				}
			},
		},
		"locked version is no longer available": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("1.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("2.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
				},
				nil,
			),
			LockFile: `
				provider "example.com/foo/beep" {
					version     = "1.2.0"
					constraints = ">= 1.0.0"
					hashes = [
						"h1:2y06Ykj0FRneZfGCTxI9wRTori8iB7ZL5kQ6YyEnh84=",
					]
				}
			`,
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				beepProvider: getproviders.MustParseVersionConstraints(">= 1.0.0"),
			},
			Check: func(t *testing.T, dir *Dir, locks *depsfile.Locks) {
				if allCached := dir.AllAvailablePackages(); len(allCached) != 0 {
					t.Errorf("wrong number of cache directory entries; want none\n%s", spew.Sdump(allCached))
				}
				if allLocked := locks.AllProviders(); len(allLocked) != 1 {
					t.Errorf("wrong number of provider lock entries; want only one\n%s", spew.Sdump(allLocked))
				}

				gotLock := locks.Provider(beepProvider)
				wantLock := depsfile.NewProviderLock(
					beepProvider,
					getproviders.MustParseVersion("1.2.0"),
					getproviders.MustParseVersionConstraints(">= 1.0.0"),
					[]getproviders.Hash{"h1:2y06Ykj0FRneZfGCTxI9wRTori8iB7ZL5kQ6YyEnh84="},
				)
				if diff := cmp.Diff(wantLock, gotLock, depsfile.ProviderLockComparer); diff != "" {
					t.Errorf("wrong lock entry\n%s", diff)
				}
			},
			WantErr: `some providers could not be installed:
- example.com/foo/beep: the previously-selected version 1.2.0 is no longer available`,
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								beepProvider: getproviders.MustParseVersionConstraints(">= 1.0.0"),
							},
						},
					},
					beepProvider: {
						{
							Event:    "QueryPackagesBegin",
							Provider: beepProvider,
							Args: struct {
								Constraints string
								Locked      bool
							}{">= 1.0.0", true},
						},
						{
							Event:    "QueryPackagesFailure",
							Provider: beepProvider,
							Args:     `the previously-selected version 1.2.0 is no longer available`,
						},
					},
				}
			},
		},
		"no versions match the version constraint": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("1.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
				},
				nil,
			),
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
			},
			WantErr: `some providers could not be installed:
- example.com/foo/beep: no available releases match the given constraints >= 2.0.0`,
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								beepProvider: getproviders.MustParseVersionConstraints(">= 2.0.0"),
							},
						},
					},
					beepProvider: {
						{
							Event:    "QueryPackagesBegin",
							Provider: beepProvider,
							Args: struct {
								Constraints string
								Locked      bool
							}{">= 2.0.0", false},
						},
						{
							Event:    "QueryPackagesFailure",
							Provider: beepProvider,
							Args:     `no available releases match the given constraints >= 2.0.0`,
						},
					},
				}
			},
		},
		"version exists but doesn't support the current platform": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("1.0.0"),
						TargetPlatform: wrongPlatform,
						Location:       beepProviderDir,
					},
				},
				nil,
			),
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				beepProvider: getproviders.MustParseVersionConstraints(">= 1.0.0"),
			},
			WantErr: `some providers could not be installed:
- example.com/foo/beep: provider example.com/foo/beep 1.0.0 is not available for bleep_bloop`,
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								beepProvider: getproviders.MustParseVersionConstraints(">= 1.0.0"),
							},
						},
					},
					beepProvider: {
						{
							Event:    "QueryPackagesBegin",
							Provider: beepProvider,
							Args: struct {
								Constraints string
								Locked      bool
							}{">= 1.0.0", false},
						},
						{
							Event:    "QueryPackagesSuccess",
							Provider: beepProvider,
							Args:     "1.0.0",
						},
						{
							Event:    "FetchPackageMeta",
							Provider: beepProvider,
							Args:     "1.0.0",
						},
						{
							Event:    "FetchPackageFailure",
							Provider: beepProvider,
							Args: struct {
								Version string
								Error   string
							}{
								"1.0.0",
								"provider example.com/foo/beep 1.0.0 is not available for bleep_bloop",
							},
						},
					},
				}
			},
		},
		"available package doesn't match locked hash": {
			Source: getproviders.NewMockSource(
				[]getproviders.PackageMeta{
					{
						Provider:       beepProvider,
						Version:        getproviders.MustParseVersion("1.0.0"),
						TargetPlatform: fakePlatform,
						Location:       beepProviderDir,
					},
				},
				nil,
			),
			LockFile: `
				provider "example.com/foo/beep" {
					version     = "1.0.0"
					constraints = ">= 1.0.0"
					hashes = [
						"h1:does-not-match",
					]
				}
			`,
			Mode: InstallNewProvidersOnly,
			Reqs: getproviders.Requirements{
				beepProvider: getproviders.MustParseVersionConstraints(">= 1.0.0"),
			},
			WantErr: `some providers could not be installed:
- example.com/foo/beep: the local package for example.com/foo/beep 1.0.0 doesn't match any of the checksums previously recorded in the dependency lock file (this might be because the available checksums are for packages targeting different platforms)`,
			WantEvents: func(inst *Installer, dir *Dir) map[addrs.Provider][]*testInstallerEventLogItem {
				return map[addrs.Provider][]*testInstallerEventLogItem{
					noProvider: {
						{
							Event: "PendingProviders",
							Args: map[addrs.Provider]getproviders.VersionConstraints{
								beepProvider: getproviders.MustParseVersionConstraints(">= 1.0.0"),
							},
						},
					},
					beepProvider: {
						{
							Event:    "QueryPackagesBegin",
							Provider: beepProvider,
							Args: struct {
								Constraints string
								Locked      bool
							}{">= 1.0.0", true},
						},
						{
							Event:    "QueryPackagesSuccess",
							Provider: beepProvider,
							Args:     "1.0.0",
						},
						{
							Event:    "FetchPackageMeta",
							Provider: beepProvider,
							Args:     "1.0.0",
						},
						{
							Event:    "FetchPackageBegin",
							Provider: beepProvider,
							Args: struct {
								Version  string
								Location getproviders.PackageLocation
							}{"1.0.0", beepProviderDir},
						},
						{
							Event:    "FetchPackageFailure",
							Provider: beepProvider,
							Args: struct {
								Version string
								Error   string
							}{
								"1.0.0",
								`the local package for example.com/foo/beep 1.0.0 doesn't match any of the checksums previously recorded in the dependency lock file (this might be because the available checksums are for packages targeting different platforms)`,
							},
						},
					},
				}
			},
		},
	}

	ctx := context.Background()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.Check == nil && test.WantEvents == nil && test.WantErr == "" {
				t.Fatalf("invalid test: must set at least one of Check, WantEvents, or WantErr")
			}

			outputDir := NewDirWithPlatform(tmpDir(t), fakePlatform)
			source := test.Source
			if source == nil {
				source = getproviders.NewMockSource(nil, nil)
			}
			inst := NewInstaller(outputDir, source)
			if test.Prepare != nil {
				test.Prepare(t, inst, outputDir)
			}

			locks, lockDiags := depsfile.LoadLocksFromBytes([]byte(test.LockFile), "test.lock.hcl")
			if lockDiags.HasErrors() {
				t.Fatalf("invalid lock file: %s", lockDiags.Err().Error())
			}

			providerEvents := make(map[addrs.Provider][]*testInstallerEventLogItem)
			eventsCh := make(chan *testInstallerEventLogItem)
			var newLocks *depsfile.Locks
			var instErr error
			go func(ch chan *testInstallerEventLogItem) {
				events := installerLogEventsForTests(ch)
				ctx := events.OnContext(ctx)
				newLocks, instErr = inst.EnsureProviderVersions(ctx, locks, test.Reqs, test.Mode)
				close(eventsCh) // exits the event loop below
			}(eventsCh)
			for evt := range eventsCh {
				// We do the event collection in the main goroutine, rather than
				// running the installer itself in the main goroutine, so that
				// we can safely t.Log in here without violating the testing.T
				// usage rules.
				if evt.Provider == (addrs.Provider{}) {
					t.Logf("%s(%s)", evt.Event, spew.Sdump(evt.Args))
				} else {
					t.Logf("%s: %s(%s)", evt.Provider, evt.Event, spew.Sdump(evt.Args))
				}
				providerEvents[evt.Provider] = append(providerEvents[evt.Provider], evt)
			}

			if test.WantErr != "" {
				if instErr == nil {
					t.Errorf("succeeded; want error\nwant: %s", test.WantErr)
				} else if got, want := instErr.Error(), test.WantErr; got != want {
					t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
			} else if instErr != nil {
				t.Errorf("unexpected error\ngot: %s", instErr.Error())
			}

			if test.Check != nil {
				test.Check(t, outputDir, newLocks)
			}

			if test.WantEvents != nil {
				wantEvents := test.WantEvents(inst, outputDir)
				if diff := cmp.Diff(wantEvents, providerEvents); diff != "" {
					t.Errorf("wrong installer events\n%s", diff)
				}
			}
		})
	}
}

func TestEnsureProviderVersions_local_source(t *testing.T) {
	// create filesystem source using the test provider cache dir
	source := getproviders.NewFilesystemMirrorSource("testdata/cachedir")

	// create a temporary workdir
	tmpDirPath, err := ioutil.TempDir("", "terraform-test-providercache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDirPath)

	// set up the installer using the temporary directory and filesystem source
	platform := getproviders.Platform{OS: "linux", Arch: "amd64"}
	dir := NewDirWithPlatform(tmpDirPath, platform)
	installer := NewInstaller(dir, source)

	tests := map[string]struct {
		provider string
		version  string
		wantHash getproviders.Hash // getproviders.NilHash if not expected to be installed
		err      string
	}{
		"install-unpacked": {
			provider: "null",
			version:  "2.0.0",
			wantHash: getproviders.HashScheme1.New("qjsREM4DqEWECD43FcPqddZ9oxCG+IaMTxvWPciS05g="),
		},
		"invalid-zip-file": {
			provider: "null",
			version:  "2.1.0",
			wantHash: getproviders.NilHash,
			err:      "zip: not a valid zip file",
		},
		"version-constraint-unmet": {
			provider: "null",
			version:  "2.2.0",
			wantHash: getproviders.NilHash,
			err:      "no available releases match the given constraints 2.2.0",
		},
		"missing-executable": {
			provider: "missing/executable",
			version:  "2.0.0",
			wantHash: getproviders.NilHash, // installation fails for a provider with no executable
			err:      "provider binary not found: could not find executable file starting with terraform-provider-executable",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.TODO()

			provider := addrs.MustParseProviderSourceString(test.provider)
			versionConstraint := getproviders.MustParseVersionConstraints(test.version)
			version := getproviders.MustParseVersion(test.version)
			reqs := getproviders.Requirements{
				provider: versionConstraint,
			}

			newLocks, err := installer.EnsureProviderVersions(ctx, depsfile.NewLocks(), reqs, InstallNewProvidersOnly)
			gotProviderlocks := newLocks.AllProviders()
			wantProviderLocks := map[addrs.Provider]*depsfile.ProviderLock{
				provider: depsfile.NewProviderLock(
					provider,
					version,
					getproviders.MustParseVersionConstraints("= 2.0.0"),
					[]getproviders.Hash{
						test.wantHash,
					},
				),
			}
			if test.wantHash == getproviders.NilHash {
				wantProviderLocks = map[addrs.Provider]*depsfile.ProviderLock{}
			}

			if diff := cmp.Diff(wantProviderLocks, gotProviderlocks, depsfile.ProviderLockComparer); diff != "" {
				t.Errorf("wrong selected\n%s", diff)
			}

			if test.err == "" && err == nil {
				return
			}

			switch err := err.(type) {
			case InstallerError:
				providerError, ok := err.ProviderErrors[provider]
				if !ok {
					t.Fatalf("did not get error for provider %s", provider)
				}

				if got := providerError.Error(); got != test.err {
					t.Fatalf("wrong result\ngot:  %s\nwant: %s\n", got, test.err)
				}
			default:
				t.Fatalf("wrong error type. Expected InstallerError, got %T", err)
			}
		})
	}
}

// This test only verifies protocol errors and does not try for successfull
// installation (at the time of writing, the test files aren't signed so the
// signature verification fails); that's left to the e2e tests.
func TestEnsureProviderVersions_protocol_errors(t *testing.T) {
	source, _, close := testRegistrySource(t)
	defer close()

	// create a temporary workdir
	tmpDirPath, err := ioutil.TempDir("", "terraform-test-providercache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDirPath)

	version0 := getproviders.MustParseVersionConstraints("0.1.0") // supports protocol version 1.0
	version1 := getproviders.MustParseVersion("1.2.0")            // this is the expected result in tests with a match
	version2 := getproviders.MustParseVersionConstraints("2.0")   // supports protocol version 99

	// set up the installer using the temporary directory and mock source
	platform := getproviders.Platform{OS: "gameboy", Arch: "lr35902"}
	dir := NewDirWithPlatform(tmpDirPath, platform)
	installer := NewInstaller(dir, source)

	tests := map[string]struct {
		provider     addrs.Provider
		inputVersion getproviders.VersionConstraints
		wantVersion  getproviders.Version
	}{
		"too old": {
			addrs.MustParseProviderSourceString("example.com/awesomesauce/happycloud"),
			version0,
			version1,
		},
		"too new": {
			addrs.MustParseProviderSourceString("example.com/awesomesauce/happycloud"),
			version2,
			version1,
		},
		"unsupported": {
			addrs.MustParseProviderSourceString("example.com/weaksauce/unsupported-protocol"),
			version0,
			getproviders.UnspecifiedVersion,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			reqs := getproviders.Requirements{
				test.provider: test.inputVersion,
			}
			ctx := context.TODO()
			_, err := installer.EnsureProviderVersions(ctx, depsfile.NewLocks(), reqs, InstallNewProvidersOnly)

			switch err := err.(type) {
			case nil:
				t.Fatalf("expected error, got success")
			case InstallerError:
				providerError, ok := err.ProviderErrors[test.provider]
				if !ok {
					t.Fatalf("did not get error for provider %s", test.provider)
				}

				switch providerError := providerError.(type) {
				case getproviders.ErrProtocolNotSupported:
					if !providerError.Suggestion.Same(test.wantVersion) {
						t.Fatalf("wrong result\ngot:  %s\nwant: %s\n", providerError.Suggestion, test.wantVersion)
					}
				default:
					t.Fatalf("wrong error type. Expected ErrProtocolNotSupported, got %T", err)
				}
			default:
				t.Fatalf("wrong error type. Expected InstallerError, got %T", err)
			}
		})
	}
}

// testServices starts up a local HTTP server running a fake provider registry
// service and returns a service discovery object pre-configured to consider
// the host "example.com" to be served by the fake registry service.
//
// The returned discovery object also knows the hostname "not.example.com"
// which does not have a provider registry at all and "too-new.example.com"
// which has a "providers.v99" service that is inoperable but could be useful
// to test the error reporting for detecting an unsupported protocol version.
// It also knows fails.example.com but it refers to an endpoint that doesn't
// correctly speak HTTP, to simulate a protocol error.
//
// The second return value is a function to call at the end of a test function
// to shut down the test server. After you call that function, the discovery
// object becomes useless.
func testServices(t *testing.T) (services *disco.Disco, baseURL string, cleanup func()) {
	server := httptest.NewServer(http.HandlerFunc(fakeRegistryHandler))

	services = disco.New()
	services.ForceHostServices(svchost.Hostname("example.com"), map[string]interface{}{
		"providers.v1": server.URL + "/providers/v1/",
	})
	services.ForceHostServices(svchost.Hostname("not.example.com"), map[string]interface{}{})
	services.ForceHostServices(svchost.Hostname("too-new.example.com"), map[string]interface{}{
		// This service doesn't actually work; it's here only to be
		// detected as "too new" by the discovery logic.
		"providers.v99": server.URL + "/providers/v99/",
	})
	services.ForceHostServices(svchost.Hostname("fails.example.com"), map[string]interface{}{
		"providers.v1": server.URL + "/fails-immediately/",
	})

	// We'll also permit registry.terraform.io here just because it's our
	// default and has some unique features that are not allowed on any other
	// hostname. It behaves the same as example.com, which should be preferred
	// if you're not testing something specific to the default registry in order
	// to ensure that most things are hostname-agnostic.
	services.ForceHostServices(svchost.Hostname("registry.terraform.io"), map[string]interface{}{
		"providers.v1": server.URL + "/providers/v1/",
	})

	return services, server.URL, func() {
		server.Close()
	}
}

// testRegistrySource is a wrapper around testServices that uses the created
// discovery object to produce a Source instance that is ready to use with the
// fake registry services.
//
// As with testServices, the second return value is a function to call at the end
// of your test in order to shut down the test server.
func testRegistrySource(t *testing.T) (source *getproviders.RegistrySource, baseURL string, cleanup func()) {
	services, baseURL, close := testServices(t)
	source = getproviders.NewRegistrySource(services)
	return source, baseURL, close
}

func fakeRegistryHandler(resp http.ResponseWriter, req *http.Request) {
	path := req.URL.EscapedPath()
	if strings.HasPrefix(path, "/fails-immediately/") {
		// Here we take over the socket and just close it immediately, to
		// simulate one possible way a server might not be an HTTP server.
		hijacker, ok := resp.(http.Hijacker)
		if !ok {
			// Not hijackable, so we'll just fail normally.
			// If this happens, tests relying on this will fail.
			resp.WriteHeader(500)
			resp.Write([]byte(`cannot hijack`))
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			resp.WriteHeader(500)
			resp.Write([]byte(`hijack failed`))
			return
		}
		conn.Close()
		return
	}

	if strings.HasPrefix(path, "/pkg/") {
		switch path {
		case "/pkg/awesomesauce/happycloud_1.2.0.zip":
			resp.Write([]byte("some zip file"))
		case "/pkg/awesomesauce/happycloud_1.2.0_SHA256SUMS":
			resp.Write([]byte("000000000000000000000000000000000000000000000000000000000000f00d happycloud_1.2.0.zip\n"))
		case "/pkg/awesomesauce/happycloud_1.2.0_SHA256SUMS.sig":
			resp.Write([]byte("GPG signature"))
		default:
			resp.WriteHeader(404)
			resp.Write([]byte("unknown package file download"))
		}
		return
	}

	if !strings.HasPrefix(path, "/providers/v1/") {
		resp.WriteHeader(404)
		resp.Write([]byte(`not a provider registry endpoint`))
		return
	}

	pathParts := strings.Split(path, "/")[3:]
	if len(pathParts) < 2 {
		resp.WriteHeader(404)
		resp.Write([]byte(`unexpected number of path parts`))
		return
	}
	log.Printf("[TRACE] fake provider registry request for %#v", pathParts)
	if len(pathParts) == 2 {
		switch pathParts[0] + "/" + pathParts[1] {

		case "-/legacy":
			// NOTE: This legacy lookup endpoint is specific to
			// registry.terraform.io and not expected to work on any other
			// registry host.
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			resp.Write([]byte(`{"namespace":"legacycorp"}`))

		default:
			resp.WriteHeader(404)
			resp.Write([]byte(`unknown namespace or provider type for direct lookup`))
		}
	}

	if len(pathParts) < 3 {
		resp.WriteHeader(404)
		resp.Write([]byte(`unexpected number of path parts`))
		return
	}

	if pathParts[2] == "versions" {
		if len(pathParts) != 3 {
			resp.WriteHeader(404)
			resp.Write([]byte(`extraneous path parts`))
			return
		}

		switch pathParts[0] + "/" + pathParts[1] {
		case "awesomesauce/happycloud":
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			// Note that these version numbers are intentionally misordered
			// so we can test that the client-side code places them in the
			// correct order (lowest precedence first).
			resp.Write([]byte(`{"versions":[{"version":"0.1.0","protocols":["1.0"]},{"version":"2.0.0","protocols":["99.0"]},{"version":"1.2.0","protocols":["5.0"]}, {"version":"1.0.0","protocols":["5.0"]}]}`))
		case "weaksauce/unsupported-protocol":
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			resp.Write([]byte(`{"versions":[{"version":"0.1.0","protocols":["0.1"]}]}`))
		case "weaksauce/no-versions":
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			resp.Write([]byte(`{"versions":[]}`))
		default:
			resp.WriteHeader(404)
			resp.Write([]byte(`unknown namespace or provider type`))
		}
		return
	}

	if len(pathParts) == 6 && pathParts[3] == "download" {
		switch pathParts[0] + "/" + pathParts[1] {
		case "awesomesauce/happycloud":
			if pathParts[4] == "nonexist" {
				resp.WriteHeader(404)
				resp.Write([]byte(`unsupported OS`))
				return
			}
			version := pathParts[2]
			body := map[string]interface{}{
				"protocols":             []string{"99.0"},
				"os":                    pathParts[4],
				"arch":                  pathParts[5],
				"filename":              "happycloud_" + version + ".zip",
				"shasum":                "000000000000000000000000000000000000000000000000000000000000f00d",
				"download_url":          "/pkg/awesomesauce/happycloud_" + version + ".zip",
				"shasums_url":           "/pkg/awesomesauce/happycloud_" + version + "_SHA256SUMS",
				"shasums_signature_url": "/pkg/awesomesauce/happycloud_" + version + "_SHA256SUMS.sig",
				"signing_keys": map[string]interface{}{
					"gpg_public_keys": []map[string]interface{}{
						{
							"ascii_armor": getproviders.HashicorpPublicKey,
						},
					},
				},
			}
			enc, err := json.Marshal(body)
			if err != nil {
				resp.WriteHeader(500)
				resp.Write([]byte("failed to encode body"))
			}
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			resp.Write(enc)
		case "weaksauce/unsupported-protocol":
			var protocols []string
			version := pathParts[2]
			switch version {
			case "0.1.0":
				protocols = []string{"1.0"}
			case "2.0.0":
				protocols = []string{"99.0"}
			default:
				protocols = []string{"5.0"}
			}

			body := map[string]interface{}{
				"protocols":             protocols,
				"os":                    pathParts[4],
				"arch":                  pathParts[5],
				"filename":              "happycloud_" + version + ".zip",
				"shasum":                "000000000000000000000000000000000000000000000000000000000000f00d",
				"download_url":          "/pkg/awesomesauce/happycloud_" + version + ".zip",
				"shasums_url":           "/pkg/awesomesauce/happycloud_" + version + "_SHA256SUMS",
				"shasums_signature_url": "/pkg/awesomesauce/happycloud_" + version + "_SHA256SUMS.sig",
				"signing_keys": map[string]interface{}{
					"gpg_public_keys": []map[string]interface{}{
						{
							"ascii_armor": getproviders.HashicorpPublicKey,
						},
					},
				},
			}
			enc, err := json.Marshal(body)
			if err != nil {
				resp.WriteHeader(500)
				resp.Write([]byte("failed to encode body"))
			}
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			resp.Write(enc)
		default:
			resp.WriteHeader(404)
			resp.Write([]byte(`unknown namespace/provider/version/architecture`))
		}
		return
	}

	resp.WriteHeader(404)
	resp.Write([]byte(`unrecognized path scheme`))
}

// In order to be able to compare the recorded temp dir paths, we need to
// normalize the path to match what the installer would report.
func tmpDir(t *testing.T) string {
	d := t.TempDir()
	unlinked, err := filepath.EvalSymlinks(d)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(unlinked)
}
