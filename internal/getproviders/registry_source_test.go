package getproviders

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/google/go-cmp/cmp"
	svchost "github.com/hashicorp/terraform-svchost"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestSourceAvailableVersions(t *testing.T) {
	source, baseURL, close := testRegistrySource(t)
	defer close()

	tests := []struct {
		provider     string
		wantVersions []string
		wantErr      string
	}{
		// These test cases are relying on behaviors of the fake provider
		// registry server implemented in registry_client_test.go.
		{
			"example.com/awesomesauce/happycloud",
			[]string{"0.1.0", "1.0.0", "1.2.0", "2.0.0"},
			``,
		},
		{
			"example.com/weaksauce/no-versions",
			nil,
			``, // having no versions is not an error, it's just odd
		},
		{
			"example.com/nonexist/nonexist",
			nil,
			`provider registry example.com does not have a provider named example.com/nonexist/nonexist`,
		},
		{
			"not.example.com/foo/bar",
			nil,
			`host not.example.com does not offer a Terraform provider registry`,
		},
		{
			"too-new.example.com/foo/bar",
			nil,
			`host too-new.example.com does not support the provider registry protocol required by this Terraform version, but may be compatible with a different Terraform version`,
		},
		{
			"fails.example.com/foo/bar",
			nil,
			`could not query provider registry for fails.example.com/foo/bar: the request failed after 2 attempts, please try again later: Get "` + baseURL + `/fails-immediately/foo/bar/versions": EOF`,
		},
	}

	for _, test := range tests {
		t.Run(test.provider, func(t *testing.T) {
			provider := addrs.MustParseProviderSourceString(test.provider)
			gotVersions, _, err := source.AvailableVersions(context.Background(), provider)

			if err != nil {
				if test.wantErr == "" {
					t.Fatalf("wrong error\ngot:  %s\nwant: <nil>", err.Error())
				}
				if got, want := err.Error(), test.wantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			}

			if test.wantErr != "" {
				t.Fatalf("wrong error\ngot:  <nil>\nwant: %s", test.wantErr)
			}

			var gotVersionsStr []string
			if gotVersions != nil {
				gotVersionsStr = make([]string, len(gotVersions))
				for i, v := range gotVersions {
					gotVersionsStr[i] = v.String()
				}
			}

			if diff := cmp.Diff(test.wantVersions, gotVersionsStr); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func TestSourceAvailableVersions_warnings(t *testing.T) {
	source, _, close := testRegistrySource(t)
	defer close()

	provider := addrs.MustParseProviderSourceString("example.com/weaksauce/no-versions")
	_, warnings, err := source.AvailableVersions(context.Background(), provider)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if len(warnings) != 1 {
		t.Fatalf("wrong number of warnings. Expected 1, got %d", len(warnings))
	}

}

func TestSourcePackageMeta(t *testing.T) {
	source, baseURL, close := testRegistrySource(t)
	defer close()

	tests := []struct {
		provider   string
		version    string
		os, arch   string
		want       PackageMeta
		wantHashes []Hash
		wantErr    string
	}{
		// These test cases are relying on behaviors of the fake provider
		// registry server implemented in registry_client_test.go.
		{
			"example.com/awesomesauce/happycloud",
			"1.2.0",
			"linux", "amd64",
			PackageMeta{
				Provider: addrs.NewProvider(
					svchost.Hostname("example.com"), "awesomesauce", "happycloud",
				),
				Version:          versions.MustParseVersion("1.2.0"),
				ProtocolVersions: VersionList{versions.MustParseVersion("5.0.0")},
				TargetPlatform:   Platform{"linux", "amd64"},
				Filename:         "happycloud_1.2.0.zip",
				Location:         PackageHTTPURL(baseURL + "/pkg/awesomesauce/happycloud_1.2.0.zip"),
				Authentication: PackageAuthenticationAll(
					NewMatchingChecksumAuthentication(
						[]byte("000000000000000000000000000000000000000000000000000000000000f00d happycloud_1.2.0.zip\n000000000000000000000000000000000000000000000000000000000000face happycloud_1.2.0_face.zip\n"),
						"happycloud_1.2.0.zip",
						[32]byte{30: 0xf0, 31: 0x0d},
					),
					NewArchiveChecksumAuthentication(Platform{"linux", "amd64"}, [32]byte{30: 0xf0, 31: 0x0d}),
					NewSignatureAuthentication(
						[]byte("000000000000000000000000000000000000000000000000000000000000f00d happycloud_1.2.0.zip\n000000000000000000000000000000000000000000000000000000000000face happycloud_1.2.0_face.zip\n"),
						[]byte("GPG signature"),
						[]SigningKey{
							{ASCIIArmor: HashicorpPublicKey},
						},
					),
				),
			},
			[]Hash{
				"zh:000000000000000000000000000000000000000000000000000000000000f00d",
				"zh:000000000000000000000000000000000000000000000000000000000000face",
			},
			``,
		},
		{
			"example.com/awesomesauce/happycloud",
			"1.2.0",
			"nonexist", "amd64",
			PackageMeta{},
			nil,
			`provider example.com/awesomesauce/happycloud 1.2.0 is not available for nonexist_amd64`,
		},
		{
			"not.example.com/awesomesauce/happycloud",
			"1.2.0",
			"linux", "amd64",
			PackageMeta{},
			nil,
			`host not.example.com does not offer a Terraform provider registry`,
		},
		{
			"too-new.example.com/awesomesauce/happycloud",
			"1.2.0",
			"linux", "amd64",
			PackageMeta{},
			nil,
			`host too-new.example.com does not support the provider registry protocol required by this Terraform version, but may be compatible with a different Terraform version`,
		},
		{
			"fails.example.com/awesomesauce/happycloud",
			"1.2.0",
			"linux", "amd64",
			PackageMeta{},
			nil,
			`could not query provider registry for fails.example.com/awesomesauce/happycloud: the request failed after 2 attempts, please try again later: Get "http://placeholder-origin/fails-immediately/awesomesauce/happycloud/1.2.0/download/linux/amd64": EOF`,
		},
	}

	// Sometimes error messages contain specific HTTP endpoint URLs, but
	// since our test server is on a random port we'd not be able to
	// consistently match those. Instead, we'll normalize the URLs.
	urlPattern := regexp.MustCompile(`http://[^/]+/`)

	cmpOpts := cmp.Comparer(Version.Same)

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s for %s_%s", test.provider, test.os, test.arch), func(t *testing.T) {
			// TEMP: We don't yet have a function for parsing provider
			// source addresses so we'll just fake it in here for now.
			parts := strings.Split(test.provider, "/")
			providerAddr := addrs.Provider{
				Hostname:  svchost.Hostname(parts[0]),
				Namespace: parts[1],
				Type:      parts[2],
			}

			version := versions.MustParseVersion(test.version)

			got, err := source.PackageMeta(context.Background(), providerAddr, version, Platform{test.os, test.arch})

			if err != nil {
				if test.wantErr == "" {
					t.Fatalf("wrong error\ngot:  %s\nwant: <nil>", err.Error())
				}
				gotErr := urlPattern.ReplaceAllLiteralString(err.Error(), "http://placeholder-origin/")
				if got, want := gotErr, test.wantErr; got != want {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			}

			if test.wantErr != "" {
				t.Fatalf("wrong error\ngot:  <nil>\nwant: %s", test.wantErr)
			}

			if diff := cmp.Diff(test.want, got, cmpOpts); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
			if diff := cmp.Diff(test.wantHashes, got.AcceptableHashes()); diff != "" {
				t.Errorf("wrong AcceptableHashes result\n%s", diff)
			}
		})
	}

}
