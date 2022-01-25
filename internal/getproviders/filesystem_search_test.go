package getproviders

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
)

func TestSearchLocalDirectory(t *testing.T) {
	tests := []struct {
		Fixture string
		Subdir  string
		Want    map[addrs.Provider]PackageMetaList
	}{
		{
			"symlinks",
			"symlink",
			map[addrs.Provider]PackageMetaList{
				addrs.MustParseProviderSourceString("example.com/foo/bar"): {
					{
						Provider:       addrs.MustParseProviderSourceString("example.com/foo/bar"),
						Version:        MustParseVersion("1.0.0"),
						TargetPlatform: Platform{OS: "linux", Arch: "amd64"},
						Filename:       "terraform-provider-bar_1.0.0_linux_amd64.zip",
						Location:       PackageLocalDir("testdata/search-local-directory/symlinks/real/example.com/foo/bar/1.0.0/linux_amd64"),
					},
				},
				// This search doesn't find example.net/foo/bar because only
				// the top-level search directory is supported as being a
				// symlink, and so we ignore the example.net symlink to
				// example.com that is one level deeper.
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Fixture, func(t *testing.T) {
			fullDir := filepath.Join("testdata/search-local-directory", test.Fixture, test.Subdir)
			got, err := SearchLocalDirectory(fullDir)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
			}
			want := test.Want

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}
