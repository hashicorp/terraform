package checks

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/initwd"
)

func TestChecksHappyPath(t *testing.T) {
	const fixtureDir = "testdata/happypath"
	loader, close := configload.NewLoaderForTests(t)
	loader.AllowLanguageExperiments(true)
	defer close()
	inst := initwd.NewModuleInstaller(loader.ModulesDir(), nil)
	_, instDiags := inst.InstallModules(context.Background(), fixtureDir, true, initwd.ModuleInstallHooksImpl{})
	if instDiags.HasErrors() {
		t.Fatal(instDiags.Err())
	}
	if err := loader.RefreshModules(); err != nil {
		t.Fatalf("failed to refresh modules after installation: %s", err)
	}

	/////////////////////////////////////////////////////////////////////////

	cfg, hclDiags := loader.LoadConfig(fixtureDir)
	if hclDiags.HasErrors() {
		t.Fatalf("invalid configuration: %s", hclDiags.Error())
	}

	resourceA := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "null_resource",
		Name: "a",
	}.InModule(addrs.RootModule)
	resourceNoChecks := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "null_resource",
		Name: "no_checks",
	}.InModule(addrs.RootModule)
	resourceNonExist := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "null_resource",
		Name: "nonexist",
	}.InModule(addrs.RootModule)
	rootOutput := addrs.OutputValue{
		Name: "a",
	}.InModule(addrs.RootModule)
	moduleChild := addrs.RootModule.Child("child")
	resourceB := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "null_resource",
		Name: "b",
	}.InModule(moduleChild)
	resourceC := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "null_resource",
		Name: "c",
	}.InModule(moduleChild)
	childOutput := addrs.OutputValue{
		Name: "b",
	}.InModule(moduleChild)
	smokeTestA := addrs.SmokeTest{Name: "a"}.InModule(addrs.RootModule)

	// First some consistency checks to make sure our configuration is the
	// shape we are relying on it to be.
	if addr := resourceA; cfg.Module.ResourceByAddr(addr.Resource) == nil {
		t.Fatalf("configuration does not include %s", addr)
	}
	if addr := resourceB; cfg.Children["child"].Module.ResourceByAddr(addr.Resource) == nil {
		t.Fatalf("configuration does not include %s", addr)
	}
	if addr := resourceNoChecks; cfg.Module.ResourceByAddr(addr.Resource) == nil {
		t.Fatalf("configuration does not include %s", addr)
	}
	if addr := resourceNonExist; cfg.Module.ResourceByAddr(addr.Resource) != nil {
		t.Fatalf("configuration includes %s, which is not supposed to exist", addr)
	}
	if cfg.Module.SmokeTests["a"] == nil {
		t.Fatalf("configuration does not include %s", smokeTestA)
	}

	/////////////////////////////////////////////////////////////////////////

	checks := NewState(cfg)

	missing := 0
	if addr := resourceA; !checks.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
	}
	if addr := resourceB; !checks.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
	}
	if addr := resourceC; !checks.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
	}
	if addr := rootOutput; !checks.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
	}
	if addr := childOutput; !checks.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
	}
	if addr := smokeTestA; !checks.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
	}
	if addr := resourceNoChecks; checks.ConfigHasChecks(addr) {
		t.Errorf("checks detected for %s, even though it has none", addr)
	}
	if addr := resourceNonExist; checks.ConfigHasChecks(addr) {
		t.Errorf("checks detected for %s, even though it doesn't exist", addr)
	}
	if missing > 0 {
		t.Fatalf("missing some configuration objects we'd need for subsequent testing")
	}

	/////////////////////////////////////////////////////////////////////////

	// Everything should start with status unknown.

	{
		wantConfigAddrs := addrs.MakeSet[addrs.ConfigCheckable](
			resourceA,
			resourceB,
			resourceC,
			rootOutput,
			childOutput,
			smokeTestA,
		)
		gotConfigAddrs := checks.AllConfigAddrs()
		if diff := cmp.Diff(wantConfigAddrs, gotConfigAddrs); diff != "" {
			t.Errorf("wrong detected config addresses\n%s", diff)
		}

		for _, configAddr := range gotConfigAddrs {
			if got, want := checks.AggregateCheckStatus(configAddr), StatusUnknown; got != want {
				t.Errorf("incorrect initial aggregate check status for %s: %s, but want %s", configAddr, got, want)
			}
		}
	}

	/////////////////////////////////////////////////////////////////////////

	// The following are steps that would normally be done by Terraform Core
	// as part of visiting checkable objects during the graph walk. We're
	// simulating a likely sequence of calls here for testing purposes, but
	// Terraform Core won't necessarily visit all of these in exactly the
	// same order every time and so this is just one possible valid ordering
	// of calls.

	resourceInstA := resourceA.Resource.Absolute(addrs.RootModuleInstance).Instance(addrs.NoKey)
	rootOutputInst := rootOutput.OutputValue.Absolute(addrs.RootModuleInstance)
	moduleChildInst := addrs.RootModuleInstance.Child("child", addrs.NoKey)
	resourceInstB := resourceB.Resource.Absolute(moduleChildInst).Instance(addrs.NoKey)
	resourceInstC0 := resourceC.Resource.Absolute(moduleChildInst).Instance(addrs.IntKey(0))
	resourceInstC1 := resourceC.Resource.Absolute(moduleChildInst).Instance(addrs.IntKey(1))
	childOutputInst := childOutput.OutputValue.Absolute(moduleChildInst)
	smokeTestInstA := smokeTestA.SmokeTest.Absolute(addrs.RootModuleInstance)

	checks.ReportCheckableObjects(resourceA, addrs.MakeSet[addrs.Checkable](resourceInstA))
	checks.ReportCheckResult(resourceInstA, addrs.ResourcePrecondition, 0, StatusPass)
	checks.ReportCheckResult(resourceInstA, addrs.ResourcePrecondition, 1, StatusPass)
	checks.ReportCheckResult(resourceInstA, addrs.ResourcePostcondition, 0, StatusPass)

	checks.ReportCheckableObjects(resourceB, addrs.MakeSet[addrs.Checkable](resourceInstB))
	checks.ReportCheckResult(resourceInstB, addrs.ResourcePrecondition, 0, StatusPass)

	checks.ReportCheckableObjects(resourceC, addrs.MakeSet[addrs.Checkable](resourceInstC0, resourceInstC1))
	checks.ReportCheckResult(resourceInstC0, addrs.ResourcePostcondition, 0, StatusPass)
	checks.ReportCheckResult(resourceInstC1, addrs.ResourcePostcondition, 0, StatusPass)

	checks.ReportCheckableObjects(childOutput, addrs.MakeSet[addrs.Checkable](childOutputInst))
	checks.ReportCheckResult(childOutputInst, addrs.OutputPrecondition, 0, StatusPass)

	checks.ReportCheckableObjects(rootOutput, addrs.MakeSet[addrs.Checkable](rootOutputInst))
	checks.ReportCheckResult(rootOutputInst, addrs.OutputPrecondition, 0, StatusPass)

	checks.ReportCheckableObjects(smokeTestA, addrs.MakeSet[addrs.Checkable](smokeTestInstA))
	checks.ReportCheckResult(smokeTestInstA, addrs.SmokeTestPrecondition, 0, StatusPass)
	checks.ReportCheckResult(smokeTestInstA, addrs.SmokeTestDataResource, 0, StatusPass)
	checks.ReportCheckResult(smokeTestInstA, addrs.SmokeTestPostcondition, 0, StatusPass)

	/////////////////////////////////////////////////////////////////////////

	// This "section" is simulating what we might do to report the results
	// of the checks after a run completes.

	{
		configCount := 0
		for _, configAddr := range checks.AllConfigAddrs() {
			configCount++
			if got, want := checks.AggregateCheckStatus(configAddr), StatusPass; got != want {
				t.Errorf("incorrect final aggregate check status for %s: %s, but want %s", configAddr, got, want)
			}
		}
		if got, want := configCount, 6; got != want {
			t.Errorf("incorrect number of known config addresses %d; want %d", got, want)
		}
	}

	{
		objAddrs := addrs.MakeSet[addrs.Checkable](
			resourceInstA,
			rootOutputInst,
			resourceInstB,
			resourceInstC0,
			resourceInstC1,
			childOutputInst,
			smokeTestInstA,
		)
		for _, addr := range objAddrs {
			if got, want := checks.ObjectCheckStatus(addr), StatusPass; got != want {
				t.Errorf("incorrect final check status for object %s: %s, but want %s", addr, got, want)
			}
		}
	}
}
