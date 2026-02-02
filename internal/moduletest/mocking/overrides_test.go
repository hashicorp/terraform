// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocking

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/definitions"
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
		Overrides: addrs.MakeMap[addrs.Targetable, *definitions.Override](),
	}
	run.Overrides.Put(primary, &definitions.Override{
		Target: &addrs.Target{
			Subject: testrun,
		},
	})

	// Add a unique item to the test file, and duplicate the test run data.
	file := &configs.TestFile{
		Overrides: addrs.MakeMap[addrs.Targetable, *definitions.Override](),
	}
	file.Overrides.Put(primary, &definitions.Override{
		Target: &addrs.Target{
			Subject: testfile,
		},
	})
	file.Overrides.Put(secondary, &definitions.Override{
		Target: &addrs.Target{
			Subject: testfile,
		},
	})

	mocks := map[addrs.RootProviderConfig]*definitions.MockData{
		addrs.RootProviderConfig{
			Provider: addrs.NewDefaultProvider("mock"),
		}: {
			Overrides: addrs.MakeMap[addrs.Targetable, *definitions.Override](
				addrs.MakeMapElem[addrs.Targetable, *definitions.Override](primary, &definitions.Override{
					Target: &addrs.Target{
						Subject: provider,
					},
				}),
				addrs.MakeMapElem[addrs.Targetable, *definitions.Override](secondary, &definitions.Override{
					Target: &addrs.Target{
						Subject: provider,
					},
				}),
				addrs.MakeMapElem[addrs.Targetable, *definitions.Override](tertiary, &definitions.Override{
					Target: &addrs.Target{
						Subject: provider,
					},
				})),
		},
	}

	overrides, _ := PackageOverrides(nil, run, file, mocks)

	// We now expect that the run and file overrides took precedence.
	first, fOk := overrides.GetResourceOverride(primary, addrs.AbsProviderConfig{
		Provider: addrs.NewDefaultProvider("mock"),
	})
	second, sOk := overrides.GetResourceOverride(secondary, addrs.AbsProviderConfig{
		Provider: addrs.NewDefaultProvider("mock"),
	})
	third, tOk := overrides.GetResourceOverride(tertiary, addrs.AbsProviderConfig{
		Provider: addrs.NewDefaultProvider("mock"),
	})

	if !fOk || !sOk || !tOk {
		t.Errorf("expected to find all overrides, but got %t %t %t", fOk, sOk, tOk)
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

}
