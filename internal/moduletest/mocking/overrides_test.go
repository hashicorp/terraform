// Copyright IBM Corp. 2014, 2026
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

	mustTarget := func(s string) *addrs.TargetPattern {
		target, diags := addrs.ParseTargetStr(s)
		if diags.HasErrors() {
			t.Fatal(diags.Err())
		}
		return &target
	}

	testrun := mustTarget("test_instance.test_run")
	testfile := mustTarget("test_instance.test_file")
	provider := mustTarget("test_instance.provider")

	primaryTarget := *mustTarget(primary.String())
	secondaryTarget := *mustTarget(secondary.String())
	tertiaryTarget := *mustTarget(tertiary.String())

	// Add a single override to the test run.
	run := &configs.TestRun{
		Overrides: addrs.MakeMap[addrs.TargetPattern, *configs.Override](),
	}
	run.Overrides.Put(primaryTarget, &configs.Override{
		Target: testrun,
	})

	// Add a unique item to the test file, and duplicate the test run data.
	file := &configs.TestFile{
		Overrides: addrs.MakeMap[addrs.TargetPattern, *configs.Override](),
	}
	file.Overrides.Put(primaryTarget, &configs.Override{
		Target: testfile,
	})
	file.Overrides.Put(secondaryTarget, &configs.Override{
		Target: testfile,
	})

	mocks := map[addrs.RootProviderConfig]*configs.MockData{
		addrs.RootProviderConfig{
			Provider: addrs.NewDefaultProvider("mock"),
		}: {
			Overrides: addrs.MakeMap[addrs.TargetPattern, *configs.Override](
				addrs.MakeMapElem[addrs.TargetPattern, *configs.Override](primaryTarget, &configs.Override{
					Target: provider,
				}),
				addrs.MakeMapElem[addrs.TargetPattern, *configs.Override](secondaryTarget, &configs.Override{
					Target: provider,
				}),
				addrs.MakeMapElem[addrs.TargetPattern, *configs.Override](tertiaryTarget, &configs.Override{
					Target: provider,
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

	if !first.Target.Equal(*testrun) {
		t.Errorf("expected %s but got %s for primary", testrun, first.Target)
	}

	if !second.Target.Equal(*testfile) {
		t.Errorf("expected %s but got %s for secondary", testfile, second.Target)
	}

	if !third.Target.Equal(*provider) {
		t.Errorf("expected %s but got %s for tertiary", provider, third.Target)
	}

}

func TestOverrides_targetPatterns(t *testing.T) {
	override := func(target string) addrs.MapElem[addrs.TargetPattern, *configs.Override] {
		parsed, diags := addrs.ParseTargetStr(target)
		if diags.HasErrors() {
			t.Fatal(diags.Err())
		}
		return addrs.MakeMapElem(parsed, &configs.Override{Target: &parsed})
	}

	mockProvider := addrs.AbsProviderConfig{Provider: addrs.NewDefaultProvider("mock")}
	otherProvider := addrs.AbsProviderConfig{Provider: addrs.NewDefaultProvider("other")}

	overrides := OverridesForTesting(func(providers map[addrs.RootProviderConfig]addrs.Map[addrs.TargetPattern, *configs.Override]) {
		providers[addrs.RootProviderConfig{Provider: mockProvider.Provider}] = addrs.MakeMap(
			override("test_instance.c[0]"),
			override("test_instance.d"),
		)
	}, func(locals addrs.Map[addrs.TargetPattern, *configs.Override]) {
		for _, elem := range []addrs.MapElem[addrs.TargetPattern, *configs.Override]{
			override("test_instance.a"),
			override("test_instance.a[1]"),
			override("module.child.test_instance.b"),
			override("test_instance.c"),
			override("test_instance.d"),
			override("module.child"),
			override("module.child[1]"),
			override("module.other"),
		} {
			locals.Put(elem.Key, elem.Value)
		}
	})

	resourceTests := []struct {
		inst     string
		provider addrs.AbsProviderConfig
		want     string
	}{
		// the most specific override is used
		{"test_instance.a[1]", otherProvider, "test_instance.a[1]"},
		{"test_instance.a[0]", otherProvider, "test_instance.a"},
		// keyless module steps select every module instance
		{"module.child[2].test_instance.b[0]", otherProvider, "module.child.test_instance.b"},
		// a more specific mock provider override is used over a local one
		{"test_instance.c[0]", mockProvider, "test_instance.c[0]"},
		{"test_instance.c[1]", mockProvider, "test_instance.c"},
		{"test_instance.c[0]", otherProvider, "test_instance.c"},
		// but local overrides take precedence for the same target
		{"test_instance.d", mockProvider, "test_instance.d"},
		// module overrides don't apply to the resources within them
		{"module.other.test_instance.z", otherProvider, ""},
	}
	for _, test := range resourceTests {
		inst, diags := addrs.ParseAbsResourceInstanceStr(test.inst)
		if diags.HasErrors() {
			t.Fatal(diags.Err())
		}
		got, ok := overrides.GetResourceOverride(inst, test.provider)
		switch {
		case test.want == "" && ok:
			t.Errorf("%s: unexpected override %s", test.inst, got.Target)
		case test.want != "" && !ok:
			t.Errorf("%s: no override found, want %s", test.inst, test.want)
		case ok && got.Target.String() != test.want:
			t.Errorf("%s: wrong override %s, want %s", test.inst, got.Target, test.want)
		}
	}

	moduleTests := []struct {
		inst       string
		want       string
		overridden bool
	}{
		{"module.child[1]", "module.child[1]", true},
		{"module.child[0]", "module.child", true},
		// overrides of an ancestor don't apply to the module itself, but
		// do mark it as overridden
		{"module.child[0].module.grandchild", "", true},
		{"module.unrelated", "", false},
	}
	for _, test := range moduleTests {
		inst, diags := addrs.ParseModuleInstanceStr(test.inst)
		if diags.HasErrors() {
			t.Fatal(diags.Err())
		}
		got, ok := overrides.GetModuleOverride(inst)
		switch {
		case test.want == "" && ok:
			t.Errorf("%s: unexpected override %s", test.inst, got.Target)
		case test.want != "" && !ok:
			t.Errorf("%s: no override found, want %s", test.inst, test.want)
		case ok && got.Target.String() != test.want:
			t.Errorf("%s: wrong override %s, want %s", test.inst, got.Target, test.want)
		}
		if got := overrides.IsOverridden(inst); got != test.overridden {
			t.Errorf("%s: IsOverridden returned %t, want %t", test.inst, got, test.overridden)
		}
	}
}
