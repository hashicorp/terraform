package providercache

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestInstallPackage(t *testing.T) {
	tmpDirPath, err := ioutil.TempDir("", "terraform-test-providercache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDirPath)
	tmpDirPath, err = filepath.EvalSymlinks(tmpDirPath)
	if err != nil {
		t.Fatal(err)
	}

	linuxPlatform := getproviders.Platform{
		OS:   "linux",
		Arch: "amd64",
	}
	nullProvider := addrs.NewProvider(
		addrs.DefaultRegistryHost, "hashicorp", "null",
	)

	tmpDir := NewDirWithPlatform(tmpDirPath, linuxPlatform)

	meta := getproviders.PackageMeta{
		Provider: nullProvider,
		Version:  versions.MustParseVersion("2.1.0"),

		ProtocolVersions: getproviders.VersionList{versions.MustParseVersion("5.0.0")},
		TargetPlatform:   linuxPlatform,

		Filename: "terraform-provider-null_2.1.0_linux_amd64.zip",
		Location: getproviders.PackageLocalArchive("testdata/terraform-provider-null_2.1.0_linux_amd64.zip"),
	}

	result, err := tmpDir.InstallPackage(context.TODO(), meta, nil)
	if err != nil {
		t.Fatalf("InstallPackage failed: %s", err)
	}
	if result != nil {
		t.Errorf("unexpected result %#v, wanted nil", result)
	}

	// Now we should see the same version reflected in the temporary directory.
	got := tmpDir.AllAvailablePackages()
	want := map[addrs.Provider][]CachedProvider{
		nullProvider: {
			CachedProvider{
				Provider: nullProvider,

				Version: versions.MustParseVersion("2.1.0"),

				PackageDir: tmpDirPath + "/registry.terraform.io/hashicorp/null/2.1.0/linux_amd64",
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong cache contents after install\n%s", diff)
	}
}

func TestLinkFromOtherCache(t *testing.T) {
	srcDirPath := "testdata/cachedir"
	tmpDirPath, err := ioutil.TempDir("", "terraform-test-providercache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDirPath)
	tmpDirPath, err = filepath.EvalSymlinks(tmpDirPath)
	if err != nil {
		t.Fatal(err)
	}

	windowsPlatform := getproviders.Platform{
		OS:   "windows",
		Arch: "amd64",
	}
	nullProvider := addrs.NewProvider(
		addrs.DefaultRegistryHost, "hashicorp", "null",
	)

	srcDir := NewDirWithPlatform(srcDirPath, windowsPlatform)
	tmpDir := NewDirWithPlatform(tmpDirPath, windowsPlatform)

	// First we'll check our preconditions: srcDir should have only the
	// null provider version 2.0.0 in it, because we're faking that we're on
	// Windows, and tmpDir should have no providers in it at all.

	gotSrcInitial := srcDir.AllAvailablePackages()
	wantSrcInitial := map[addrs.Provider][]CachedProvider{
		nullProvider: {
			CachedProvider{
				Provider: nullProvider,

				// We want 2.0.0 rather than 2.1.0 because the 2.1.0 package is
				// still packed and thus not considered to be a cache member.
				Version: versions.MustParseVersion("2.0.0"),

				PackageDir: "testdata/cachedir/registry.terraform.io/hashicorp/null/2.0.0/windows_amd64",
			},
		},
	}
	if diff := cmp.Diff(wantSrcInitial, gotSrcInitial); diff != "" {
		t.Fatalf("incorrect initial source directory contents\n%s", diff)
	}

	gotTmpInitial := tmpDir.AllAvailablePackages()
	wantTmpInitial := map[addrs.Provider][]CachedProvider{}
	if diff := cmp.Diff(wantTmpInitial, gotTmpInitial); diff != "" {
		t.Fatalf("incorrect initial temp directory contents\n%s", diff)
	}

	cacheEntry := srcDir.ProviderLatestVersion(nullProvider)
	if cacheEntry == nil {
		// This is weird because we just checked for the presence of this above
		t.Fatalf("null provider has no latest version in source directory")
	}

	err = tmpDir.LinkFromOtherCache(cacheEntry, nil)
	if err != nil {
		t.Fatalf("LinkFromOtherCache failed: %s", err)
	}

	// Now we should see the same version reflected in the temporary directory.
	got := tmpDir.AllAvailablePackages()
	want := map[addrs.Provider][]CachedProvider{
		nullProvider: {
			CachedProvider{
				Provider: nullProvider,

				// We want 2.0.0 rather than 2.1.0 because the 2.1.0 package is
				// still packed and thus not considered to be a cache member.
				Version: versions.MustParseVersion("2.0.0"),

				PackageDir: tmpDirPath + "/registry.terraform.io/hashicorp/null/2.0.0/windows_amd64",
			},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("wrong cache contents after link\n%s", diff)
	}
}
