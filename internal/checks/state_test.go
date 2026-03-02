// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package checks_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
)

func TestChecksHappyPath(t *testing.T) {
	const fixtureDir = "testdata/happypath"

	cfg := LoadConfigForTests(t, fixtureDir, "tests")

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
	checkBlock := addrs.Check{
		Name: "check",
	}.InModule(addrs.RootModule)

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
	if addr := checkBlock; cfg.Module.Checks[addr.Check.Name] == nil {
		t.Fatalf("configuration does not include %s", addr)
	}

	/////////////////////////////////////////////////////////////////////////

	state := checks.NewState(cfg)

	missing := 0
	if addr := resourceA; !state.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
	}
	if addr := resourceB; !state.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
	}
	if addr := resourceC; !state.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
	}
	if addr := rootOutput; !state.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
	}
	if addr := childOutput; !state.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
	}
	if addr := resourceNoChecks; state.ConfigHasChecks(addr) {
		t.Errorf("checks detected for %s, even though it has none", addr)
	}
	if addr := resourceNonExist; state.ConfigHasChecks(addr) {
		t.Errorf("checks detected for %s, even though it doesn't exist", addr)
	}
	if addr := checkBlock; !state.ConfigHasChecks(addr) {
		t.Errorf("checks not detected for %s", addr)
		missing++
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
			checkBlock,
		)
		gotConfigAddrs := state.AllConfigAddrs()
		if diff := cmp.Diff(wantConfigAddrs, gotConfigAddrs); diff != "" {
			t.Errorf("wrong detected config addresses\n%s", diff)
		}

		for _, configAddr := range gotConfigAddrs {
			if got, want := state.AggregateCheckStatus(configAddr), checks.StatusUnknown; got != want {
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
	checkBlockInst := checkBlock.Check.Absolute(addrs.RootModuleInstance)

	state.ReportCheckableObjects(resourceA, addrs.MakeSet[addrs.Checkable](resourceInstA))
	state.ReportCheckResult(resourceInstA, addrs.ResourcePrecondition, 0, checks.StatusPass)
	state.ReportCheckResult(resourceInstA, addrs.ResourcePrecondition, 1, checks.StatusPass)
	state.ReportCheckResult(resourceInstA, addrs.ResourcePostcondition, 0, checks.StatusPass)

	state.ReportCheckableObjects(resourceB, addrs.MakeSet[addrs.Checkable](resourceInstB))
	state.ReportCheckResult(resourceInstB, addrs.ResourcePrecondition, 0, checks.StatusPass)

	state.ReportCheckableObjects(resourceC, addrs.MakeSet[addrs.Checkable](resourceInstC0, resourceInstC1))
	state.ReportCheckResult(resourceInstC0, addrs.ResourcePostcondition, 0, checks.StatusPass)
	state.ReportCheckResult(resourceInstC1, addrs.ResourcePostcondition, 0, checks.StatusPass)

	state.ReportCheckableObjects(childOutput, addrs.MakeSet[addrs.Checkable](childOutputInst))
	state.ReportCheckResult(childOutputInst, addrs.OutputPrecondition, 0, checks.StatusPass)

	state.ReportCheckableObjects(rootOutput, addrs.MakeSet[addrs.Checkable](rootOutputInst))
	state.ReportCheckResult(rootOutputInst, addrs.OutputPrecondition, 0, checks.StatusPass)

	state.ReportCheckableObjects(checkBlock, addrs.MakeSet[addrs.Checkable](checkBlockInst))
	state.ReportCheckResult(checkBlockInst, addrs.CheckAssertion, 0, checks.StatusPass)

	/////////////////////////////////////////////////////////////////////////

	// This "section" is simulating what we might do to report the results
	// of the checks after a run completes.

	{
		configCount := 0
		for _, configAddr := range state.AllConfigAddrs() {
			configCount++
			if got, want := state.AggregateCheckStatus(configAddr), checks.StatusPass; got != want {
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
			checkBlockInst,
		)
		for _, addr := range objAddrs {
			if got, want := state.ObjectCheckStatus(addr), checks.StatusPass; got != want {
				t.Errorf("incorrect final check status for object %s: %s, but want %s", addr, got, want)
			}
		}
	}
}
