package providercache

import (
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestDirReading(t *testing.T) {
	testDir := "testdata/cachedir"

	// We'll force using particular platforms for unit testing purposes,
	// so that we'll get consistent results on all platforms.
	windowsPlatform := getproviders.Platform{ // only null 2.0.0 is cached
		OS:   "windows",
		Arch: "amd64",
	}
	linuxPlatform := getproviders.Platform{ // various provider versions are cached
		OS:   "linux",
		Arch: "amd64",
	}

	nullProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost, "hashicorp", "null",
	)
	randomProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost, "hashicorp", "random",
	)
	randomBetaProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost, "hashicorp", "random-beta",
	)
	nonExistProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost, "bloop", "nonexist",
	)
	legacyProvider := addrs.NewLegacyProvider("legacy")
	missingExecutableProvider := addrs.NewProvider(
		addrs.DefaultProviderRegistryHost, "missing", "executable",
	)

	t.Run("ProviderLatestVersion", func(t *testing.T) {
		t.Run("exists", func(t *testing.T) {
			dir := NewDirWithPlatform(testDir, windowsPlatform)

			got := dir.ProviderLatestVersion(nullProvider)
			want := &CachedProvider{
				Provider: nullProvider,

				// We want 2.0.0 rather than 2.1.0 because the 2.1.0 package is
				// still packed and thus not considered to be a cache member.
				Version: versions.MustParseVersion("2.0.0"),

				PackageDir: "testdata/cachedir/registry.terraform.io/hashicorp/null/2.0.0/windows_amd64",
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
		t.Run("no package for current platform", func(t *testing.T) {
			dir := NewDirWithPlatform(testDir, windowsPlatform)

			// random provider is only cached for linux_amd64 in our fixtures dir
			got := dir.ProviderLatestVersion(randomProvider)
			var want *CachedProvider

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
		t.Run("no versions available at all", func(t *testing.T) {
			dir := NewDirWithPlatform(testDir, windowsPlatform)

			// nonexist provider is not present in our fixtures dir at all
			got := dir.ProviderLatestVersion(nonExistProvider)
			var want *CachedProvider

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	})

	t.Run("ProviderVersion", func(t *testing.T) {
		t.Run("exists", func(t *testing.T) {
			dir := NewDirWithPlatform(testDir, windowsPlatform)

			got := dir.ProviderVersion(nullProvider, versions.MustParseVersion("2.0.0"))
			want := &CachedProvider{
				Provider: nullProvider,
				Version:  versions.MustParseVersion("2.0.0"),

				PackageDir: "testdata/cachedir/registry.terraform.io/hashicorp/null/2.0.0/windows_amd64",
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
		t.Run("specified version is not cached", func(t *testing.T) {
			dir := NewDirWithPlatform(testDir, windowsPlatform)

			// there is no v5.0.0 package in our fixtures dir
			got := dir.ProviderVersion(nullProvider, versions.MustParseVersion("5.0.0"))
			var want *CachedProvider

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
		t.Run("no package for current platform", func(t *testing.T) {
			dir := NewDirWithPlatform(testDir, windowsPlatform)

			// random provider 1.2.0 is only cached for linux_amd64 in our fixtures dir
			got := dir.ProviderVersion(randomProvider, versions.MustParseVersion("1.2.0"))
			var want *CachedProvider

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
		t.Run("no versions available at all", func(t *testing.T) {
			dir := NewDirWithPlatform(testDir, windowsPlatform)

			// nonexist provider is not present in our fixtures dir at all
			got := dir.ProviderVersion(nonExistProvider, versions.MustParseVersion("1.0.0"))
			var want *CachedProvider

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	})

	t.Run("AllAvailablePackages", func(t *testing.T) {
		dir := NewDirWithPlatform(testDir, linuxPlatform)

		got := dir.AllAvailablePackages()
		want := map[addrs.Provider][]CachedProvider{
			legacyProvider: {
				{
					Provider:   legacyProvider,
					Version:    versions.MustParseVersion("1.0.0"),
					PackageDir: "testdata/cachedir/registry.terraform.io/-/legacy/1.0.0/linux_amd64",
				},
			},
			nullProvider: {
				{
					Provider:   nullProvider,
					Version:    versions.MustParseVersion("2.0.0"),
					PackageDir: "testdata/cachedir/registry.terraform.io/hashicorp/null/2.0.0/linux_amd64",
				},
			},
			randomProvider: {
				{
					Provider:   randomProvider,
					Version:    versions.MustParseVersion("1.2.0"),
					PackageDir: "testdata/cachedir/registry.terraform.io/hashicorp/random/1.2.0/linux_amd64",
				},
			},
			randomBetaProvider: {
				{
					Provider:   randomBetaProvider,
					Version:    versions.MustParseVersion("1.2.0"),
					PackageDir: "testdata/cachedir/registry.terraform.io/hashicorp/random-beta/1.2.0/linux_amd64",
				},
			},
			missingExecutableProvider: {
				{
					Provider:   missingExecutableProvider,
					Version:    versions.MustParseVersion("2.0.0"),
					PackageDir: "testdata/cachedir/registry.terraform.io/missing/executable/2.0.0/linux_amd64",
				},
			},
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
}
