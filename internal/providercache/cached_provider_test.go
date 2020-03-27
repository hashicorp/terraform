package providercache

import (
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestCachedProviderHash(t *testing.T) {
	cp := &CachedProvider{
		Provider: addrs.NewProvider(
			addrs.DefaultRegistryHost,
			"hashicorp", "null",
		),
		Version: getproviders.MustParseVersion("2.0.0"),

		PackageDir:     "testdata/cachedir/registry.terraform.io/hashicorp/null/2.0.0/darwin_amd64",
		ExecutableFile: "testdata/cachedir/registry.terraform.io/hashicorp/null/2.0.0/darwin_amd64/terraform-provider-null",
	}

	want := "h1:qjsREM4DqEWECD43FcPqddZ9oxCG+IaMTxvWPciS05g="
	got, err := cp.Hash()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if got != want {
		t.Errorf("wrong Hash result\ngot:  %s\nwant: %s", got, want)
	}

	gotMatches, err := cp.MatchesHash(want)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if wantMatches := true; gotMatches != wantMatches {
		t.Errorf("wrong MatchesHash result\ngot:  %#v\nwant: %#v", gotMatches, wantMatches)
	}

	// The windows build has a different hash because its executable filename
	// has a .exe suffix, but the darwin build (hashed above) does not.
	cp2 := &CachedProvider{
		Provider: addrs.NewProvider(
			addrs.DefaultRegistryHost,
			"hashicorp", "null",
		),
		Version: getproviders.MustParseVersion("2.0.0"),

		PackageDir:     "testdata/cachedir/registry.terraform.io/hashicorp/null/2.0.0/windows_amd64",
		ExecutableFile: "testdata/cachedir/registry.terraform.io/hashicorp/null/2.0.0/windows_amd64/terraform-provider-null",
	}
	gotMatches, err = cp2.MatchesHash(want)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if wantMatches := false; gotMatches != wantMatches {
		t.Errorf("wrong MatchesHash result for other package\ngot:  %#v\nwant: %#v", gotMatches, wantMatches)
	}

}
