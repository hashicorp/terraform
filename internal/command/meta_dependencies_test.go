// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
)

func Test_mergeLockedDependencies(t *testing.T) {

	providerA := tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "my-org", "providerA")
	providerB := tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "my-org", "providerB")
	v1_0_0 := providerreqs.MustParseVersion("1.0.0")
	v2_0_0 := providerreqs.MustParseVersion("2.0.0")
	versionConstraintv1, _ := providerreqs.ParseVersionConstraints("1.0.0")
	versionConstraintv2, _ := providerreqs.ParseVersionConstraints(">=2.0.0")
	hashesProviderA := []providerreqs.Hash{providerreqs.MustParseHash("providerA:this-is-providerA")}
	hashesProviderB := []providerreqs.Hash{providerreqs.MustParseHash("providerB:this-is-providerB")}

	cases := map[string]struct {
		configLocks   *depsfile.Locks
		stateLocks    *depsfile.Locks
		expectedLocks *depsfile.Locks
	}{
		"no providers described in either config or state": {
			configLocks:   depsfile.NewLocks(),
			stateLocks:    depsfile.NewLocks(),
			expectedLocks: depsfile.NewLocks(),
		},
		"1 provider described in config, none in state": {
			configLocks: func() *depsfile.Locks {
				configLocks := depsfile.NewLocks()
				configLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return configLocks
			}(),
			stateLocks: depsfile.NewLocks(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return combinedLocks
			}(),
		},
		"1 provider described in state, none in config": {
			configLocks: depsfile.NewLocks(),
			stateLocks: func() *depsfile.Locks {
				stateLocks := depsfile.NewLocks()
				stateLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return stateLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return combinedLocks
			}(),
		},
		"1 provider described in config, 1 completely different provider described in state": {
			configLocks: func() *depsfile.Locks {
				configLocks := depsfile.NewLocks()
				configLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return configLocks
			}(),
			stateLocks: func() *depsfile.Locks {
				stateLocks := depsfile.NewLocks()

				// Imagine that the state contains:
				// 1) state for resources in the config
				stateLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)

				// 2) also, state using a provider that's deleted from the config and only present in state
				stateLocks.SetProvider(providerB, v2_0_0, versionConstraintv2, hashesProviderB)

				return stateLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				combinedLocks.SetProvider(providerB, v2_0_0, versionConstraintv2, hashesProviderB)
				return combinedLocks
			}(),
		},
		"1 provider described in config, same provider described in state at same version": {
			configLocks: func() *depsfile.Locks {
				configLocks := depsfile.NewLocks()
				configLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return configLocks
			}(),
			stateLocks: func() *depsfile.Locks {
				stateLocks := depsfile.NewLocks()
				// Matches config providers
				stateLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return stateLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return combinedLocks
			}(),
		},
		"1 provider described in config, same provider described in state at DIFFERENT version": {
			configLocks: func() *depsfile.Locks {
				configLocks := depsfile.NewLocks()
				// v1.0.0
				configLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return configLocks
			}(),
			stateLocks: func() *depsfile.Locks {
				stateLocks := depsfile.NewLocks()
				// v2.0.0
				stateLocks.SetProvider(providerA, v2_0_0, versionConstraintv2, hashesProviderA)
				return stateLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				// TODO(SarahFrench/radeksimko): Should we expect v1 or v2 here?
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return combinedLocks
			}(),
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// Use tmp dir as we're creating lock files in the test
			td := t.TempDir()
			t.Chdir(td)

			ui := new(cli.MockUi)
			view, _ := testView(t)

			m := Meta{
				Ui:   ui,
				View: view,
			}

			// Code under test - combine deps from state with prior deps from config
			mergedLocks := m.mergeLockedDependencies(tc.configLocks, tc.stateLocks)

			if !mergedLocks.Equal(tc.expectedLocks) {
				// Something is wrong - try to get useful feedback to show in the UI when running tests
				if len(mergedLocks.AllProviders()) != len(tc.expectedLocks.AllProviders()) {
					t.Fatalf("expected merged dependencies to include %d providers, but got %d:\n %#v",
						len(tc.expectedLocks.AllProviders()),
						len(mergedLocks.AllProviders()),
						mergedLocks.AllProviders(),
					)
				}
				for _, lock := range tc.expectedLocks.AllProviders() {
					if match := mergedLocks.Provider(lock.Provider()); match == nil {
						t.Fatalf("expected merged dependencies to include provider %s, but it's missing", lock.Provider())
					}
				}
				if diff := cmp.Diff(tc.expectedLocks, mergedLocks); diff != "" {
					t.Errorf("difference in file contents detected\n%s", diff)
				}
			}
		})
	}
}
