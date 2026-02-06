// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
)

// This tests combining locks from config and state. Locks derived from state are always unconstrained, i.e. no version constraint data,
// so this test
func Test_mergeLockedDependencies_config_and_state(t *testing.T) {
	providerA := tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "my-org", "providerA")
	providerB := tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "my-org", "providerB")
	v1_0_0 := providerreqs.MustParseVersion("1.0.0")
	versionConstraintv1, _ := providerreqs.ParseVersionConstraints("1.0.0")
	hashesProviderA := []providerreqs.Hash{providerreqs.MustParseHash("providerA:this-is-providerA")}
	hashesProviderB := []providerreqs.Hash{providerreqs.MustParseHash("providerB:this-is-providerB")}

	var versionUnconstrained providerreqs.VersionConstraints = nil
	noVersion := versions.Version{}

	cases := map[string]struct {
		configLocks   *depsfile.Locks
		stateLocks    *depsfile.Locks
		expectedLocks *depsfile.Locks
	}{
		"no locks when all inputs empty": {
			configLocks:   depsfile.NewLocks(),
			stateLocks:    depsfile.NewLocks(),
			expectedLocks: depsfile.NewLocks(),
		},
		"nil config locks": {
			configLocks: nil,
			stateLocks: func() *depsfile.Locks {
				configLocks := depsfile.NewLocks()
				configLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return configLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return combinedLocks
			}(),
		},
		"nil state locks": {
			configLocks: func() *depsfile.Locks {
				configLocks := depsfile.NewLocks()
				configLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return configLocks
			}(),
			stateLocks: nil,
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return combinedLocks
			}(),
		},
		"all nil locks": {
			configLocks:   nil,
			stateLocks:    nil,
			expectedLocks: depsfile.NewLocks(),
		},
		"when provider only described in config, output locks have matching constraints": {
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
		"when provider only described in state, output locks are unconstrained": {
			configLocks: depsfile.NewLocks(),
			stateLocks: func() *depsfile.Locks {
				stateLocks := depsfile.NewLocks()
				stateLocks.SetProvider(providerA, noVersion, versionUnconstrained, hashesProviderA)
				return stateLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, noVersion, versionUnconstrained, hashesProviderA)
				return combinedLocks
			}(),
		},
		"different providers present in state and config are combined, with version constraints kept on config providers": {
			configLocks: func() *depsfile.Locks {
				configLocks := depsfile.NewLocks()
				configLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return configLocks
			}(),
			stateLocks: func() *depsfile.Locks {
				stateLocks := depsfile.NewLocks()

				// Imagine that the state locks contain:
				// 1) provider for resources in the config
				stateLocks.SetProvider(providerA, noVersion, versionUnconstrained, hashesProviderA)

				// 2) also, a provider that's deleted from the config and only present in state
				stateLocks.SetProvider(providerB, noVersion, versionUnconstrained, hashesProviderB)

				return stateLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)     // version constraint preserved
				combinedLocks.SetProvider(providerB, noVersion, versionUnconstrained, hashesProviderB) // sourced from state only
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
			// mergedLocks := m.mergeLockedDependencies(tc.configLocks, tc.stateLocks)
			mergedLocks := m.mergeLockedDependencies(tc.configLocks, tc.stateLocks)

			// We cannot use (l *depsfile.Locks) Equal here as it doesn't compare version constraints
			// Instead, inspect entries directly
			if len(mergedLocks.AllProviders()) != len(tc.expectedLocks.AllProviders()) {
				t.Fatalf("expected merged dependencies to include %d providers, but got %d:\n %#v",
					len(tc.expectedLocks.AllProviders()),
					len(mergedLocks.AllProviders()),
					mergedLocks.AllProviders(),
				)
			}
			for _, lock := range tc.expectedLocks.AllProviders() {
				match := mergedLocks.Provider(lock.Provider())
				if match == nil {
					t.Fatalf("expected merged dependencies to include provider %s, but it's missing", lock.Provider())
				}
				if len(match.VersionConstraints()) != len(lock.VersionConstraints()) {
					t.Fatalf("detected a problem with version constraints for provider %s, got: %d, want %d",
						lock.Provider(),
						len(match.VersionConstraints()),
						len(lock.VersionConstraints()),
					)
				}
				if len(match.VersionConstraints()) > 0 && len(lock.VersionConstraints()) > 0 {
					gotConstraints := match.VersionConstraints()[0]
					wantConstraints := lock.VersionConstraints()[0]

					if gotConstraints.Boundary.String() != wantConstraints.Boundary.String() {
						t.Fatalf("expected merged dependencies to include provider %s with version constraint %v, but instead got %v",
							lock.Provider(),
							gotConstraints.Boundary.String(),
							wantConstraints.Boundary.String(),
						)
					}
				}
			}
			if diff := cmp.Diff(tc.expectedLocks, mergedLocks); diff != "" {
				t.Errorf("difference in file contents detected\n%s", diff)
			}
		})
	}
}
