// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

func TestPackageOverrides(t *testing.T) {
	mustResourceInstance := func(s string) addrs.AbsResourceInstance {
		addr, diags := addrs.ParseAbsResourceInstanceStr(s)
		if len(diags) > 0 {
			t.Fatal(diags)
		}
		return addr
	}

	primary := mustResourceInstance("test_instance.primary")
	secondary := mustResourceInstance("test_instance.secondary")
	tertiary := mustResourceInstance("test_instance.tertiary")

	testrun := mustResourceInstance("test_instance.test_run")
	testfile := mustResourceInstance("test_instance.test_file")
	provider := mustResourceInstance("test_instance.provider")

	// Add a single override to the test run.
	run := &configs.TestRun{
		Overrides: addrs.MakeMap[addrs.Targetable, *configs.Override](),
	}
	run.Overrides.Put(primary, &configs.Override{
		Target: &addrs.Target{
			Subject: testrun,
		},
	})

	// Add a unique item to the test file, and duplicate the test run data.
	file := &configs.TestFile{
		Overrides: addrs.MakeMap[addrs.Targetable, *configs.Override](),
	}
	file.Overrides.Put(primary, &configs.Override{
		Target: &addrs.Target{
			Subject: testfile,
		},
	})
	file.Overrides.Put(secondary, &configs.Override{
		Target: &addrs.Target{
			Subject: testfile,
		},
	})

	// Add all data from the file and run block are duplicating here, and then
	// a unique one.
	config := &configs.Config{
		Module: &configs.Module{
			ProviderConfigs: map[string]*configs.Provider{
				"mock": {
					Mock: true,
					MockData: &configs.MockData{
						Overrides: addrs.MakeMap[addrs.Targetable, *configs.Override](),
					},
				},
				"real": {},
			},
		},
	}
	config.Module.ProviderConfigs["mock"].MockData.Overrides.Put(primary, &configs.Override{
		Target: &addrs.Target{
			Subject: provider,
		},
	})
	config.Module.ProviderConfigs["mock"].MockData.Overrides.Put(secondary, &configs.Override{
		Target: &addrs.Target{
			Subject: provider,
		},
	})
	config.Module.ProviderConfigs["mock"].MockData.Overrides.Put(tertiary, &configs.Override{
		Target: &addrs.Target{
			Subject: provider,
		},
	})

	overrides := PackageOverrides(run, file, config)

	// We now expect that the run and file overrides took precedence.
	first, pOk := overrides.GetResourceOverride(primary, addrs.AbsProviderConfig{
		Provider: addrs.Provider{
			Type: "mock",
		},
	})
	second, sOk := overrides.GetResourceOverride(secondary, addrs.AbsProviderConfig{
		Provider: addrs.Provider{
			Type: "mock",
		},
	})
	third, tOk := overrides.GetResourceOverride(tertiary, addrs.AbsProviderConfig{
		Provider: addrs.Provider{
			Type: "mock",
		},
	})

	if !pOk || !sOk || !tOk {
		t.Fatalf("expected to find all overrides, but got %t %t %t", pOk, sOk, tOk)
	}

	if !first.Target.Subject.(addrs.AbsResourceInstance).Equal(testrun) {
		t.Errorf("expected %s but got %s for primary", testrun, first.Target.Subject)
	}

	if !second.Target.Subject.(addrs.AbsResourceInstance).Equal(testfile) {
		t.Errorf("expected %s but got %s for primary", testfile, second.Target.Subject)
	}

	if !third.Target.Subject.(addrs.AbsResourceInstance).Equal(provider) {
		t.Errorf("expected %s but got %s for primary", provider, third.Target.Subject)
	}

	// Also, final sanity check.
	_, ok := overrides.providerOverrides["real"]
	if ok {
		t.Errorf("shouldn't have stored the real provider but did")
	}

}
