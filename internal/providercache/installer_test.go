package providercache

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestEnsureProviderVersions(t *testing.T) {
	// Set up a test provider "foo" with two versions which support different protocols
	// used by both package metas
	provider := addrs.NewDefaultProvider("foo")
	platform := getproviders.Platform{OS: "gameboy", Arch: "lr35902"}

	// foo version 1.0 supports protocol 4
	version1 := getproviders.MustParseVersion("1.0.0")
	protocols1 := getproviders.VersionList{getproviders.MustParseVersion("4.0")}
	meta1, close1, _ := getproviders.FakeInstallablePackageMeta(provider, version1, protocols1, platform)
	defer close1()

	// foo version 2.0 supports protocols 4 and 5.2
	version2 := getproviders.MustParseVersion("2.0.0")
	protocols2 := getproviders.VersionList{getproviders.MustParseVersion("4.0"), getproviders.MustParseVersion("5.2")}
	meta2, close2, _ := getproviders.FakeInstallablePackageMeta(provider, version2, protocols2, platform)
	defer close2()

	// foo version 3.0 supports protocol 6
	version3 := getproviders.MustParseVersion("3.0.0")
	protocols3 := getproviders.VersionList{getproviders.MustParseVersion("6.0")}
	meta3, close3, _ := getproviders.FakeInstallablePackageMeta(provider, version3, protocols3, platform)
	defer close3()

	// set up the mock source
	source := getproviders.NewMockSource(
		[]getproviders.PackageMeta{meta1, meta2, meta3},
	)

	// create a temporary workdir
	tmpDirPath, err := ioutil.TempDir("", "terraform-test-providercache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDirPath)

	// set up the installer using the temporary directory and mock source
	dir := NewDirWithPlatform(tmpDirPath, platform)
	installer := NewInstaller(dir, source)

	// First test: easy case. The requested version supports the current plugin protocol version
	reqs := getproviders.Requirements{
		provider: getproviders.MustParseVersionConstraints("2.0"),
	}
	ctx := context.TODO()
	selections, err := installer.EnsureProviderVersions(ctx, reqs, InstallNewProvidersOnly)
	if err != nil {
		t.Fatalf("expected sucess, got error: %s", err)
	}
	if len(selections) != 1 {
		t.Fatalf("wrong number of results. Got %d, expected 1", len(selections))
	}
	got := selections[provider]
	if !got.Same(version2) {
		t.Fatalf("wrong result. Expected provider version %s, got %s", version2, got)
	}

	// For the second test, set the requirement to something later than the
	// version that supports the current plugin protocol version 5.0
	reqs[provider] = getproviders.MustParseVersionConstraints("3.0")

	selections, err = installer.EnsureProviderVersions(ctx, reqs, InstallNewProvidersOnly)
	if err == nil {
		t.Fatalf("expected error, got success")
	}
	if len(selections) != 0 {
		t.Errorf("wrong number of results. Got %d, expected 0", len(selections))
	}
	if !strings.Contains(err.Error(), "Provider version 2.0.0 is the latest compatible version.") {
		t.Fatalf("wrong error: %s", err)
	}

	// For the third test, set the requirement to something earlier than the
	// version that supports the current plugin protocol version 5.0
	reqs[provider] = getproviders.MustParseVersionConstraints("1.0")

	selections, err = installer.EnsureProviderVersions(ctx, reqs, InstallNewProvidersOnly)
	if err == nil {
		t.Fatalf("expected error, got success")
	}
	if len(selections) != 0 {
		t.Errorf("wrong number of results. Got %d, expected 0", len(selections))
	}
	if !strings.Contains(err.Error(), "Provider version 2.0.0 is the earliest compatible version.") {
		t.Fatalf("wrong error: %s", err)
	}
}
