// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"slices"
	"testing"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/providercache"
)

// Test the impacts of dev_overrides and reattached/unmanaged providers on the provider installation process.
// The locks returned from EnsureProviderVersions are what's saved to the dependency lock file, so we are interested
// in how the pre-existing locks and how providers are overidden impacts the locks returned from that installation process.
func TestEnsureProviderVersions_devOverrideAndReattachedProviders(t *testing.T) {
	providerSource := newMockProviderSource(t, map[string][]string{
		"provider-a": {"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
		"provider-b": {"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
		"provider-c": {"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
		"provider-d": {"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
	})

	providerA := addrs.NewDefaultProvider("provider-a")
	providerB := addrs.NewDefaultProvider("provider-b")
	providerC := addrs.NewDefaultProvider("provider-c")
	providerD := addrs.NewDefaultProvider("provider-d")

	// In all test cases the imagined config required providers A through D at specific versions.
	reqs := providerreqs.Requirements{
		providerA: providerreqs.MustParseVersionConstraints("1.0.0"),
		providerB: providerreqs.MustParseVersionConstraints("2.0.0"),
		providerC: providerreqs.MustParseVersionConstraints("3.0.0"),
		providerD: providerreqs.MustParseVersionConstraints("4.0.0"),
	}

	// Some tests are installing providers for the first time, and prior locks include only A.
	priorLocksJustA := depsfile.NewLocks()
	priorLocksJustA.SetProvider(
		providerA,
		versions.MustParseVersion("1.0.0"),
		providerreqs.MustParseVersionConstraints("1.0.0"),
		nil, // no hashes needed for this test
	)

	// Other tests are performing an install after all providers (A-D) have already been added to the dependency lock file.
	priorLocksABCD := depsfile.NewLocks()
	priorLocksABCD.SetProvider(
		providerA,
		versions.MustParseVersion("1.0.0"),
		providerreqs.MustParseVersionConstraints("1.0.0"),
		nil, // no hashes needed for this test
	)
	priorLocksABCD.SetProvider(
		providerB,
		versions.MustParseVersion("2.0.0"),
		providerreqs.MustParseVersionConstraints("2.0.0"),
		nil, // no hashes needed for this test
	)
	priorLocksABCD.SetProvider(
		providerC,
		versions.MustParseVersion("3.0.0"),
		providerreqs.MustParseVersionConstraints("3.0.0"),
		nil, // no hashes needed for this test
	)
	priorLocksABCD.SetProvider(
		providerD,
		versions.MustParseVersion("4.0.0"),
		providerreqs.MustParseVersionConstraints("4.0.0"),
		nil, // no hashes needed for this test
	)

	cases := map[string]struct {
		providerDevOverrides map[addrs.Provider]getproviders.PackageLocalDir
		unmanagedProviders   map[addrs.Provider]*plugin.ReattachConfig

		priorLocks                   *depsfile.Locks
		expectedProviderTypesInLocks []string
	}{
		"no overrides or unmanaged providers": {
			providerDevOverrides: map[addrs.Provider]getproviders.PackageLocalDir{},
			unmanagedProviders:   map[addrs.Provider]*plugin.ReattachConfig{},
			priorLocks:           priorLocksJustA, // Prior locks contain only provider A.
			expectedProviderTypesInLocks: []string{
				// All required providers are installed as expected.
				providerA.ForDisplay(),
				providerB.ForDisplay(),
				providerC.ForDisplay(),
				providerD.ForDisplay(),
			},
		},
		"reattachment present at first-time installation of provider": {
			providerDevOverrides: map[addrs.Provider]getproviders.PackageLocalDir{},
			unmanagedProviders: map[addrs.Provider]*plugin.ReattachConfig{
				providerD: {
					Protocol:        "grpc",
					ProtocolVersion: 6,
					Pid:             1234567890,
					Test:            true,
					Addr:            nil,
				},
			},
			priorLocks: priorLocksJustA, // Prior locks contain only provider A.
			expectedProviderTypesInLocks: []string{
				providerA.ForDisplay(),
				providerB.ForDisplay(), // New
				providerC.ForDisplay(), // New
				// D is not installed due to being reattached/unmanaged
			},
		},
		"reattachment present at subsequent installation of provider": {
			providerDevOverrides: map[addrs.Provider]getproviders.PackageLocalDir{},
			unmanagedProviders: map[addrs.Provider]*plugin.ReattachConfig{
				providerD: {
					Protocol:        "grpc",
					ProtocolVersion: 6,
					Pid:             1234567890,
					Test:            true,
					Addr:            nil,
				},
			},
			priorLocks: priorLocksABCD, // Prior locks include the provider that's being reattached/unmanaged.
			expectedProviderTypesInLocks: []string{
				providerA.ForDisplay(),
				providerB.ForDisplay(),
				providerC.ForDisplay(),
				providerD.ForDisplay(), // Pre-existing lock for D is expected to be unaffected by use of reattachment/unmanaged.
			},
		},
		"dev override present at first-time installation of provider": {
			providerDevOverrides: map[addrs.Provider]getproviders.PackageLocalDir{
				providerD: "/path/to/local/provider-d",
			},
			unmanagedProviders: map[addrs.Provider]*plugin.ReattachConfig{},
			priorLocks:         priorLocksJustA, // Prior locks contain only provider A.
			expectedProviderTypesInLocks: []string{
				providerA.ForDisplay(),
				providerB.ForDisplay(),
				providerC.ForDisplay(),
				providerD.ForDisplay(), // D is installed despite being dev overridden
			},
		},

		"dev override present at subsequent installation of provider": {
			providerDevOverrides: map[addrs.Provider]getproviders.PackageLocalDir{
				providerD: "/path/to/local/provider-d",
			},
			unmanagedProviders: map[addrs.Provider]*plugin.ReattachConfig{},
			priorLocks:         priorLocksABCD, // Prior locks include the provider that's being dev overridden.
			expectedProviderTypesInLocks: []string{
				providerA.ForDisplay(),
				providerB.ForDisplay(),
				providerC.ForDisplay(),
				providerD.ForDisplay(), // Pre-existing lock for D is expected to be unaffected by use of dev_override.
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// Temp dir needed because provider installation process writes to filesystem.
			td := t.TempDir()
			t.Chdir(td)

			meta := Meta{
				ProviderSource: providerSource,

				ProviderDevOverrides: tc.providerDevOverrides,
				UnmanagedProviders:   tc.unmanagedProviders,
			}

			inst := meta.providerInstaller()
			if inst == nil {
				t.Fatal("expected installer, got nil")
			}

			// We cannot make assertions about internals of the installer resulting from (Meta).providerInstaller(),
			// but we can make assertions on outputs from using the installer. Arguably this is more informative.

			ctx := t.Context()
			locks, err := inst.EnsureProviderVersions(ctx, tc.priorLocks, reqs, providercache.InstallNewProvidersOnly)
			if err != nil {
				t.Fatal(err)
			}
			if locks == nil {
				t.Fatal("expected locks, got nil")
			}

			var gotProviderTypes []string
			for addr := range locks.AllProviders() {
				gotProviderTypes = append(gotProviderTypes, addr.ForDisplay())
			}

			slices.Sort(tc.expectedProviderTypesInLocks)
			slices.Sort(gotProviderTypes)
			if diff := cmp.Diff(tc.expectedProviderTypesInLocks, gotProviderTypes); diff != "" {
				t.Errorf("unexpected difference in expected provider types in locks: %s", diff)
			}
		})
	}
}
