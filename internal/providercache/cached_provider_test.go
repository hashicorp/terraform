// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providercache

import (
	"testing"

	"github.com/hashicorp/mnptu/internal/addrs"
	"github.com/hashicorp/mnptu/internal/getproviders"
)

func TestCachedProviderHash(t *testing.T) {
	cp := &CachedProvider{
		Provider: addrs.NewProvider(
			addrs.DefaultProviderRegistryHost,
			"hashicorp", "null",
		),
		Version: getproviders.MustParseVersion("2.0.0"),

		PackageDir: "testdata/cachedir/registry.mnptu.io/hashicorp/null/2.0.0/darwin_amd64",
	}

	want := getproviders.MustParseHash("h1:qjsREM4DqEWECD43FcPqddZ9oxCG+IaMTxvWPciS05g=")
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
			addrs.DefaultProviderRegistryHost,
			"hashicorp", "null",
		),
		Version: getproviders.MustParseVersion("2.0.0"),

		PackageDir: "testdata/cachedir/registry.mnptu.io/hashicorp/null/2.0.0/windows_amd64",
	}
	gotMatches, err = cp2.MatchesHash(want)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if wantMatches := false; gotMatches != wantMatches {
		t.Errorf("wrong MatchesHash result for other package\ngot:  %#v\nwant: %#v", gotMatches, wantMatches)
	}

}

func TestExecutableFile(t *testing.T) {
	testCases := map[string]struct {
		cp   *CachedProvider
		file string
		err  string
	}{
		"linux": {
			cp: &CachedProvider{
				Provider:   addrs.NewProvider(addrs.DefaultProviderRegistryHost, "hashicorp", "null"),
				Version:    getproviders.MustParseVersion("2.0.0"),
				PackageDir: "testdata/cachedir/registry.mnptu.io/hashicorp/null/2.0.0/linux_amd64",
			},
			file: "testdata/cachedir/registry.mnptu.io/hashicorp/null/2.0.0/linux_amd64/mnptu-provider-null",
		},
		"windows": {
			cp: &CachedProvider{
				Provider:   addrs.NewProvider(addrs.DefaultProviderRegistryHost, "hashicorp", "null"),
				Version:    getproviders.MustParseVersion("2.0.0"),
				PackageDir: "testdata/cachedir/registry.mnptu.io/hashicorp/null/2.0.0/windows_amd64",
			},
			file: "testdata/cachedir/registry.mnptu.io/hashicorp/null/2.0.0/windows_amd64/mnptu-provider-null.exe",
		},
		"missing-executable": {
			cp: &CachedProvider{
				Provider:   addrs.NewProvider(addrs.DefaultProviderRegistryHost, "missing", "executable"),
				Version:    getproviders.MustParseVersion("2.0.0"),
				PackageDir: "testdata/cachedir/registry.mnptu.io/missing/executable/2.0.0/linux_amd64",
			},
			err: "could not find executable file starting with mnptu-provider-executable",
		},
		"missing-dir": {
			cp: &CachedProvider{
				Provider:   addrs.NewProvider(addrs.DefaultProviderRegistryHost, "missing", "packagedir"),
				Version:    getproviders.MustParseVersion("2.0.0"),
				PackageDir: "testdata/cachedir/registry.mnptu.io/missing/packagedir/2.0.0/linux_amd64",
			},
			err: "could not read package directory: open testdata/cachedir/registry.mnptu.io/missing/packagedir/2.0.0/linux_amd64: no such file or directory",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			file, err := tc.cp.ExecutableFile()
			if file != tc.file {
				t.Errorf("wrong file\n got: %q\nwant: %q", file, tc.file)
			}
			if err == nil && tc.err != "" {
				t.Fatalf("no error returned, want: %q", tc.err)
			} else if err != nil && err.Error() != tc.err {
				t.Errorf("wrong error\n got: %q\nwant: %q", err, tc.err)
			}
		})
	}
}
