// Copyright IBM Corp. 2014, 2026
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

// This tests combining locks from multiple sources using the mergeLockedDependencies method.
func Test_mergeLockedDependencies(t *testing.T) {
	providerA := tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "my-org", "providerA")
	providerB := tfaddr.NewProvider(tfaddr.DefaultProviderRegistryHost, "my-org", "providerB")
	v1_0_0 := providerreqs.MustParseVersion("1.0.0")
	v2_0_0 := providerreqs.MustParseVersion("2.0.0")
	versionConstraintv1, _ := providerreqs.ParseVersionConstraints("1.0.0")
	versionConstraintv2, _ := providerreqs.ParseVersionConstraints("2.0.0")
	hashesProviderA := []providerreqs.Hash{providerreqs.MustParseHash("providerA:this-is-providerA")}
	hashesProviderB := []providerreqs.Hash{providerreqs.MustParseHash("providerB:this-is-providerB")}

	cases := map[string]struct {
		baseLocks       *depsfile.Locks
		additionalLocks *depsfile.Locks
		expectedLocks   *depsfile.Locks
	}{
		"no locks when all inputs empty": {
			baseLocks:       depsfile.NewLocks(),
			additionalLocks: depsfile.NewLocks(),
			expectedLocks:   depsfile.NewLocks(),
		},
		"when provider only described in base locks": {
			baseLocks: func() *depsfile.Locks {
				configLocks := depsfile.NewLocks()
				configLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return configLocks
			}(),
			additionalLocks: depsfile.NewLocks(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return combinedLocks
			}(),
		},
		"when provider only described in additional locks": {
			baseLocks: depsfile.NewLocks(),
			additionalLocks: func() *depsfile.Locks {
				stateLocks := depsfile.NewLocks()
				stateLocks.SetProvider(providerA, v2_0_0, versionConstraintv2, hashesProviderA)
				return stateLocks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v2_0_0, versionConstraintv2, hashesProviderA)
				return combinedLocks
			}(),
		},
		"different non-overlapping providers present in base and additional locks are combined, output locks have matching constraints": {
			baseLocks: func() *depsfile.Locks {
				locks := depsfile.NewLocks()
				locks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return locks
			}(),
			additionalLocks: func() *depsfile.Locks {
				locks := depsfile.NewLocks()
				locks.SetProvider(providerB, v2_0_0, versionConstraintv2, hashesProviderB)
				return locks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				combinedLocks.SetProvider(providerB, v2_0_0, versionConstraintv2, hashesProviderB)
				return combinedLocks
			}(),
		},
		"when the same provider is present in both base and additional locks, output locks have the version constraints from the base locks": {
			baseLocks: func() *depsfile.Locks {
				locks := depsfile.NewLocks()
				locks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA)
				return locks
			}(),
			additionalLocks: func() *depsfile.Locks {
				locks := depsfile.NewLocks()

				// Overlap with base locks, but different version
				locks.SetProvider(providerA, v2_0_0, versionConstraintv2, hashesProviderA)

				// Unique to additional locks, should be preserved in output
				locks.SetProvider(providerB, v2_0_0, versionConstraintv2, hashesProviderB)

				return locks
			}(),
			expectedLocks: func() *depsfile.Locks {
				combinedLocks := depsfile.NewLocks()
				combinedLocks.SetProvider(providerA, v1_0_0, versionConstraintv1, hashesProviderA) // base locks not overridden by additional locks
				combinedLocks.SetProvider(providerB, v2_0_0, versionConstraintv2, hashesProviderB)
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

			// Code under test
			mergedLocks := m.mergeLockedDependencies(tc.baseLocks, tc.additionalLocks)

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
