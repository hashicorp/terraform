// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-svchost/disco"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/packages"
)

func TestPackagesServer_ProviderPackageVersions(t *testing.T) {

	tcs := map[string]struct {
		source           string
		expectedVersions []string
		expectedWarnings []string
		sourceFn         providerSourceFn
	}{
		"single_version": {
			source:           "hashicorp/foo",
			expectedVersions: []string{"0.1.0"},
			sourceFn: func(_ *disco.Disco) getproviders.Source {
				packages := []getproviders.PackageMeta{
					{
						Provider: addrs.MustParseProviderSourceString("hashicorp/foo"),
						Version:  versions.MustParseVersion("0.1.0"),
					},
				}
				return getproviders.NewMockSource(packages, nil)
			},
		},
		"multiple_versions": {
			source:           "hashicorp/foo",
			expectedVersions: []string{"0.1.0", "0.2.0"},
			sourceFn: func(_ *disco.Disco) getproviders.Source {
				packages := []getproviders.PackageMeta{
					{
						Provider: addrs.MustParseProviderSourceString("hashicorp/foo"),
						Version:  versions.MustParseVersion("0.1.0"),
					},
					{
						Provider: addrs.MustParseProviderSourceString("hashicorp/foo"),
						Version:  versions.MustParseVersion("0.2.0"),
					},
				}
				return getproviders.NewMockSource(packages, nil)
			},
		},
		"with_warnings": {
			source:           "hashicorp/foo",
			expectedVersions: []string{"0.1.0"},
			expectedWarnings: []string{"- warning one", "- warning two"},
			sourceFn: func(_ *disco.Disco) getproviders.Source {
				packages := []getproviders.PackageMeta{
					{
						Provider: addrs.MustParseProviderSourceString("hashicorp/foo"),
						Version:  versions.MustParseVersion("0.1.0"),
					},
				}
				warnings := map[addrs.Provider]getproviders.Warnings{
					addrs.MustParseProviderSourceString("hashicorp/foo"): {
						"warning one",
						"warning two",
					},
				}
				return getproviders.NewMockSource(packages, warnings)
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			service := &packagesServer{
				providerSourceFn: tc.sourceFn,
			}

			response, err := service.ProviderPackageVersions(context.Background(), &packages.ProviderPackageVersions_Request{
				SourceAddr: tc.source,
			})
			if err != nil {
				t.Fatal(err)
			}

			if len(tc.expectedWarnings) > 0 {
				for _, diag := range response.Diagnostics {
					if diag.Severity == terraform1.Diagnostic_WARNING && diag.Summary == "Additional provider information from registry" {
						expected := fmt.Sprintf("The remote registry returned warnings for %s:\n%s", tc.source, strings.Join(tc.expectedWarnings, "\n"))
						if diff := cmp.Diff(expected, diag.Detail); len(diff) > 0 {
							t.Errorf(diff)
						}
					}
				}

				// We're expecting only one diagnostic with the warnings.
				if len(response.Diagnostics) > 1 {
					for _, diag := range response.Diagnostics {
						t.Errorf("unexpected diagnostics: %s", diag.Detail)
					}
					return
				}
			} else {
				// Otherwise we're expecting no diagnostics.
				if len(response.Diagnostics) > 0 {
					for _, diag := range response.Diagnostics {
						t.Errorf("unexpected diagnostics: %s", diag.Detail)
					}
					return
				}
			}

			if diff := cmp.Diff(tc.expectedVersions, response.Versions); len(diff) > 0 {
				t.Errorf(diff)
			}
		})
	}

}

func TestPackagesServer_FetchProviderPackage(t *testing.T) {
	providerHashes := providerHashes(t)

	tcs := map[string]struct {
		// source, version, platforms, and hashes are what we're going to pass
		// in as the request.
		source    string
		version   string
		platforms []string
		hashes    []string

		// platformLocations, and platformHashes are what we're going to use to
		// create our virtual provider metadata.
		platformLocations map[string]string
		platformHashes    map[string][]string

		// diagnostics are the expected diagnostics for each platform.
		diagnostics map[string][]string
	}{
		"single_version_and_platform": {
			source:    "hashicorp/foo",
			version:   "0.1.0",
			platforms: []string{"linux_amd64"},
			platformLocations: map[string]string{
				"linux_amd64": "terraform_provider_foo",
			},
		},
		"single_version_multiple_platforms": {
			source:    "hashicorp/foo",
			version:   "0.1.0",
			platforms: []string{"linux_amd64", "darwin_arm64"},
			platformLocations: map[string]string{
				"linux_amd64":  "terraform_provider_foo",
				"darwin_arm64": "terraform_provider_bar",
			},
		},
		"single_version_and_platform_with_hashes": {
			source:    "hashicorp/foo",
			version:   "0.1.0",
			platforms: []string{"linux_amd64"},
			platformLocations: map[string]string{
				"linux_amd64": "terraform_provider_foo",
			},
			platformHashes: map[string][]string{
				"linux_amd64": {
					"h1:dJTExJ11p+lRE8FAm4HWzTw+uMEyfE6AXXxiOgl/nB0=",
				},
			},
		},
		"single_version_and_platform_with_hashes_clash": {
			source:    "hashicorp/foo",
			version:   "0.1.0",
			hashes:    []string{"h1:Hod4iOH+qbXMtH4orEmCem6F3T+YRPhDSNlXmOIRNuY="},
			platforms: []string{"linux_amd64"},
			platformLocations: map[string]string{
				"linux_amd64": "terraform_provider_foo",
			},
			platformHashes: map[string][]string{
				"linux_amd64": {
					"h1:dJTExJ11p+lRE8FAm4HWzTw+uMEyfE6AXXxiOgl/nB0=",
				},
			},
			diagnostics: map[string][]string{
				"linux_amd64": {
					"the local package for registry.terraform.io/hashicorp/foo 0.1.0 doesn't match any of the checksums previously recorded in the dependency lock file",
				},
			},
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			service := &packagesServer{
				providerSourceFn: func(_ *disco.Disco) getproviders.Source {
					var providers []getproviders.PackageMeta
					for _, p := range tc.platforms {
						platform := parsePlatform(t, p)

						var authentication getproviders.PackageAuthentication
						if len(tc.platformHashes) > 0 {
							authentication = getproviders.NewPackageHashAuthentication(platform, func() []getproviders.Hash {
								var hashes []getproviders.Hash
								for _, hash := range tc.platformHashes[p] {
									hashes = append(hashes, getproviders.Hash(hash))
								}
								return hashes
							}())
						}

						providers = append(providers, getproviders.PackageMeta{
							Provider:       addrs.MustParseProviderSourceString(tc.source),
							Version:        versions.MustParseVersion(tc.version),
							TargetPlatform: platform,
							Location:       getproviders.PackageLocalDir(path.Join("testdata", "providers", tc.platformLocations[p])),
							Authentication: authentication,
						})
					}

					return getproviders.NewMockSource(providers, nil)
				},
			}

			cacheDir := t.TempDir()
			response, err := service.FetchProviderPackage(context.Background(), &packages.FetchProviderPackage_Request{
				CacheDir:   cacheDir,
				SourceAddr: tc.source,
				Version:    tc.version,
				Platforms:  tc.platforms,
				Hashes:     tc.hashes,
			})
			if err != nil {
				t.Fatal(err)
			}

			if len(response.Diagnostics) > 0 {
				for _, diag := range response.Diagnostics {
					t.Errorf("unexpected diagnostics: %s", diag.Detail)
				}
				return
			}

			if len(response.Results) != len(tc.platforms) {
				t.Fatalf("wrong number of results")
			}

			for ix, platform := range tc.platforms {
				result := response.Results[ix]

				if tc.diagnostics != nil && len(tc.diagnostics[platform]) > 0 {
					if len(result.Diagnostics) != len(tc.diagnostics[platform]) {
						t.Fatalf("expected %d diagnostics for %s but found %d", len(tc.diagnostics[platform]), platform, len(result.Diagnostics))
					}
					for ix, expected := range tc.diagnostics[platform] {
						if !strings.Contains(result.Diagnostics[ix].Detail, expected) {
							t.Errorf("expected: %s\nactual:    %s", expected, result.Diagnostics[ix])
						}
					}

					return
				} else {
					if len(result.Diagnostics) > 0 {
						for _, diag := range result.Diagnostics {
							t.Errorf("unexpected diagnostics for %s: %s", platform, diag.Detail)
						}
						return
					}
				}

				if diff := cmp.Diff(providerHashes[tc.platformLocations[platform]], result.Provider.Hashes); len(diff) > 0 {
					t.Errorf(diff)
				}
			}
		})
	}
}

func providerHashes(t *testing.T) map[string][]string {
	var hashes map[string][]string

	data, err := os.ReadFile("testdata/providers/hashes.json")
	if err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(data, &hashes); err != nil {
		t.Fatal(err)
	}

	return hashes
}

func parsePlatform(t *testing.T, raw string) getproviders.Platform {
	platform, err := getproviders.ParsePlatform(raw)
	if err != nil {
		t.Fatal(err)
	}
	return platform
}
