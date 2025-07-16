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

func Test_appendLockedDependencies(t *testing.T) {

	providerA := tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "my-org", "providerA")
	providerB := tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "my-org", "providerB")
	v1_0_0 := providerreqs.MustParseVersion("1.0.0")
	v2_0_0 := providerreqs.MustParseVersion("2.0.0")
	versionConstraint, _ := providerreqs.ParseVersionConstraints(">=1.0.0")
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
				configLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)
				return configLocks
			}(),
			stateLocks: depsfile.NewLocks(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)
				return combinedLocks
			}(),
		},
		"1 provider described in state, none in config": {
			configLocks: depsfile.NewLocks(),
			stateLocks: func() *depsfile.Locks {
				stateLocks := depsfile.NewLocks()
				stateLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)
				return stateLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)
				return combinedLocks
			}(),
		},
		"1 provider described in config, 1 completely different provider described in state": {
			configLocks: func() *depsfile.Locks {
				configLocks := depsfile.NewLocks()
				configLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)
				return configLocks
			}(),
			stateLocks: func() *depsfile.Locks {
				stateLocks := depsfile.NewLocks()

				// Imagine that the state contains:
				// 1) state for resources in the config
				stateLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)

				// 2) also, state using a provider that's deleted from the config and only present in state
				stateLocks.SetProvider(providerB, v2_0_0, versionConstraint, hashesProviderB)

				return stateLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)
				combinedLocks.SetProvider(providerB, v2_0_0, versionConstraint, hashesProviderB)
				return combinedLocks
			}(),
		},
		"1 provider described in config, same provider described in state at same version": {
			configLocks: func() *depsfile.Locks {
				configLocks := depsfile.NewLocks()
				configLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)
				return configLocks
			}(),
			stateLocks: func() *depsfile.Locks {
				stateLocks := depsfile.NewLocks()
				// Matches config providers
				stateLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)
				return stateLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)
				return combinedLocks
			}(),
		},
		"1 provider described in config, same provider described in state at DIFFERENT version": {
			configLocks: func() *depsfile.Locks {
				configLocks := depsfile.NewLocks()
				// v1.0.0
				configLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)
				return configLocks
			}(),
			stateLocks: func() *depsfile.Locks {
				stateLocks := depsfile.NewLocks()
				// v2.0.0
				stateLocks.SetProvider(providerA, v2_0_0, versionConstraint, hashesProviderA)
				return stateLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				// TODO(SarahFrench/radeksimko): Should we expect v1 or v2 here?
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraint, hashesProviderA)
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

			// Set up the 'prior' locks file containing providers obtained from the config
			diags := m.replaceLockedDependencies(tc.configLocks) // saves file
			if diags.HasErrors() {
				t.Fatalf("unexpected error in test setup: %s", diags.Err().Error())
			}

			// Code under test - combine deps from state with prior deps from config
			stateProviderDiags := m.appendLockedDependencies(tc.stateLocks)
			if stateProviderDiags.HasErrors() {
				t.Fatalf("unexpected error from code under test: %s", stateProviderDiags.Err().Error())
			}

			// Assert the dependency lock file contains all the expected entries.
			deps, depsDiags := m.lockedDependencies()
			if depsDiags.HasErrors() {
				t.Fatalf("unexpected error when reading dep lock file: %s", depsDiags.Err().Error())
			}

			if !deps.Equal(tc.expectedLocks) {
				// Something is wrong - try to get useful feedback to show in the UI when running tests
				if len(deps.AllProviders()) != len(tc.expectedLocks.AllProviders()) {
					t.Fatalf("expected deps lock file to describe %d providers, but got %d:\n %#v",
						len(tc.expectedLocks.AllProviders()),
						len(deps.AllProviders()),
						deps.AllProviders(),
					)
				}
				for _, lock := range tc.expectedLocks.AllProviders() {
					if match := deps.Provider(lock.Provider()); match == nil {
						t.Fatalf("expected deps lock file to include provider %s, but it's missing", lock.Provider())
					}
				}
				if diff := cmp.Diff(tc.expectedLocks, deps); diff != "" {
					t.Errorf("difference in file contents detected\n%s", diff)
				}
			}
		})
	}
}
