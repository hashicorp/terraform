package getproviders

import (
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/google/go-cmp/cmp"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/addrs"
)

func TestFilesystemMirrorSourceAllAvailablePackages(t *testing.T) {
	source := NewFilesystemMirrorSource("testdata/filesystem-mirror")
	got, err := source.AllAvailablePackages()
	if err != nil {
		t.Fatal(err)
	}

	want := map[addrs.Provider]PackageMetaList{
		nullProvider: {
			{
				Provider:       nullProvider,
				Version:        versions.MustParseVersion("2.0.0"),
				TargetPlatform: Platform{"darwin", "amd64"},
				Filename:       "terraform-provider-null_2.0.0_darwin_amd64.zip",
				Location:       PackageLocalDir("testdata/filesystem-mirror/registry.terraform.io/hashicorp/null/2.0.0/darwin_amd64"),
			},
			{
				Provider:       nullProvider,
				Version:        versions.MustParseVersion("2.0.0"),
				TargetPlatform: Platform{"linux", "amd64"},
				Filename:       "terraform-provider-null_2.0.0_linux_amd64.zip",
				Location:       PackageLocalDir("testdata/filesystem-mirror/registry.terraform.io/hashicorp/null/2.0.0/linux_amd64"),
			},
			{
				Provider:       nullProvider,
				Version:        versions.MustParseVersion("2.1.0"),
				TargetPlatform: Platform{"linux", "amd64"},
				Filename:       "terraform-provider-null_2.1.0_linux_amd64.zip",
				Location:       PackageLocalArchive("testdata/filesystem-mirror/registry.terraform.io/hashicorp/null/terraform-provider-null_2.1.0_linux_amd64.zip"),
			},
			{
				Provider:       nullProvider,
				Version:        versions.MustParseVersion("2.0.0"),
				TargetPlatform: Platform{"windows", "amd64"},
				Filename:       "terraform-provider-null_2.0.0_windows_amd64.zip",
				Location:       PackageLocalDir("testdata/filesystem-mirror/registry.terraform.io/hashicorp/null/2.0.0/windows_amd64"),
			},
		},
		randomProvider: {
			{
				Provider:       randomProvider,
				Version:        versions.MustParseVersion("1.2.0"),
				TargetPlatform: Platform{"linux", "amd64"},
				Filename:       "terraform-provider-random_1.2.0_linux_amd64.zip",
				Location:       PackageLocalDir("testdata/filesystem-mirror/registry.terraform.io/hashicorp/random/1.2.0/linux_amd64"),
			},
		},
		happycloudProvider: {
			{
				Provider:       happycloudProvider,
				Version:        versions.MustParseVersion("0.1.0-alpha.2"),
				TargetPlatform: Platform{"darwin", "amd64"},
				Filename:       "terraform-provider-happycloud_0.1.0-alpha.2_darwin_amd64.zip",
				Location:       PackageLocalDir("testdata/filesystem-mirror/tfe.example.com/AwesomeCorp/happycloud/0.1.0-alpha.2/darwin_amd64"),
			},
		},
		legacyProvider: {
			{
				Provider:       legacyProvider,
				Version:        versions.MustParseVersion("1.0.0"),
				TargetPlatform: Platform{"linux", "amd64"},
				Filename:       "terraform-provider-legacy_1.0.0_linux_amd64.zip",
				Location:       PackageLocalDir("testdata/filesystem-mirror/registry.terraform.io/-/legacy/1.0.0/linux_amd64"),
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("incorrect result\n%s", diff)
	}
}

func TestFilesystemMirrorSourceAvailableVersions(t *testing.T) {
	source := NewFilesystemMirrorSource("testdata/filesystem-mirror")
	got, err := source.AvailableVersions(nullProvider)
	if err != nil {
		t.Fatal(err)
	}

	want := VersionList{
		versions.MustParseVersion("2.0.0"),
		versions.MustParseVersion("2.1.0"),
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("incorrect result\n%s", diff)
	}
}

func TestFilesystemMirrorSourcePackageMeta(t *testing.T) {
	t.Run("available platform", func(t *testing.T) {
		source := NewFilesystemMirrorSource("testdata/filesystem-mirror")
		got, err := source.PackageMeta(
			nullProvider, versions.MustParseVersion("2.0.0"), Platform{"linux", "amd64"},
		)
		if err != nil {
			t.Fatal(err)
		}

		want := PackageMeta{
			Provider:       nullProvider,
			Version:        versions.MustParseVersion("2.0.0"),
			TargetPlatform: Platform{"linux", "amd64"},
			Filename:       "terraform-provider-null_2.0.0_linux_amd64.zip",
			Location:       PackageLocalDir("testdata/filesystem-mirror/registry.terraform.io/hashicorp/null/2.0.0/linux_amd64"),
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("incorrect result\n%s", diff)
		}
	})
	t.Run("unavailable platform", func(t *testing.T) {
		source := NewFilesystemMirrorSource("testdata/filesystem-mirror")
		// We'll request a version that does exist in the fixture directory,
		// but for a platform that isn't supported.
		_, err := source.PackageMeta(
			nullProvider, versions.MustParseVersion("2.0.0"), Platform{"nonexist", "nonexist"},
		)

		if err == nil {
			t.Fatalf("succeeded; want error")
		}

		// This specific error type is important so callers can use it to
		// generate an actionable error message e.g. by checking to see if
		// _any_ versions of this provider support the given platform, or
		// similar helpful hints.
		wantErr := ErrPlatformNotSupported{
			Provider: nullProvider,
			Version:  versions.MustParseVersion("2.0.0"),
			Platform: Platform{"nonexist", "nonexist"},
		}
		if diff := cmp.Diff(wantErr, err); diff != "" {
			t.Errorf("incorrect error\n%s", diff)
		}
	})
}

var nullProvider = addrs.Provider{
	Hostname:  svchost.Hostname("registry.terraform.io"),
	Namespace: "hashicorp",
	Type:      "null",
}
var randomProvider = addrs.Provider{
	Hostname:  svchost.Hostname("registry.terraform.io"),
	Namespace: "hashicorp",
	Type:      "random",
}
var happycloudProvider = addrs.Provider{
	Hostname:  svchost.Hostname("tfe.example.com"),
	Namespace: "awesomecorp",
	Type:      "happycloud",
}
var legacyProvider = addrs.NewLegacyProvider("legacy")
