// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package refactoring

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
)

// FakeExternalModuleSource is used in tests to simulate an external module source.
var FakeExternalModuleSource = addrs.ModuleSourceRemote{
	Package: addrs.ModulePackage("example.com/test/fake"),
}

// StaticPopulateExpanderModule populates an expander for testing by statically
// evaluating count and for_each expressions in a configuration.
//
// This is exported so that test code in package refactoring_test can use it
// without creating an import cycle with the terraform package.
func StaticPopulateExpanderModule(t *testing.T, rootCfg *configs.Config, moduleAddr addrs.ModuleInstance, expander *instances.Expander) {
	t.Helper()

	modCfg := rootCfg.DescendantForInstance(moduleAddr)
	if modCfg == nil {
		t.Fatalf("no configuration for %s", moduleAddr)
	}

	if len(modCfg.Path) > 0 && modCfg.Path[len(modCfg.Path)-1] == "fake_external" {
		modCfg.SourceAddr = FakeExternalModuleSource
	}

	for _, call := range modCfg.Module.ModuleCalls {
		callAddr := addrs.ModuleCall{Name: call.Name}

		if call.Name == "fake_external" {
			call.SourceExpr = hcltest.MockExprLiteral(cty.StringVal(FakeExternalModuleSource.String()))
		}

		switch {
		case call.ForEach != nil:
			val, diags := call.ForEach.Value(nil)
			if diags.HasErrors() {
				t.Fatalf("invalid for_each: %s", diags.Error())
			}
			expander.SetModuleForEach(moduleAddr, callAddr, val.AsValueMap())
		case call.Count != nil:
			val, diags := call.Count.Value(nil)
			if diags.HasErrors() {
				t.Fatalf("invalid count: %s", diags.Error())
			}
			var count int
			err := gocty.FromCtyValue(val, &count)
			if err != nil {
				t.Fatalf("invalid count at %s: %s", call.Count.Range(), err)
			}
			expander.SetModuleCount(moduleAddr, callAddr, count)
		default:
			expander.SetModuleSingle(moduleAddr, callAddr)
		}

		calledMod := modCfg.Path.Child(call.Name)
		for _, inst := range expander.ExpandModule(calledMod, false) {
			StaticPopulateExpanderModule(t, rootCfg, inst, expander)
		}
	}

	for _, rc := range modCfg.Module.ManagedResources {
		StaticPopulateExpanderResource(t, moduleAddr, rc, expander)
	}
	for _, rc := range modCfg.Module.DataResources {
		StaticPopulateExpanderResource(t, moduleAddr, rc, expander)
	}
}

// StaticPopulateExpanderResource populates resource instances in an expander for testing.
func StaticPopulateExpanderResource(t *testing.T, moduleAddr addrs.ModuleInstance, rCfg *configs.Resource, expander *instances.Expander) {
	t.Helper()

	addr := rCfg.Addr()
	switch {
	case rCfg.ForEach != nil:
		val, diags := rCfg.ForEach.Value(nil)
		if diags.HasErrors() {
			t.Fatalf("invalid for_each: %s", diags.Error())
		}
		expander.SetResourceForEach(moduleAddr, addr, val.AsValueMap())
	case rCfg.Count != nil:
		val, diags := rCfg.Count.Value(nil)
		if diags.HasErrors() {
			t.Fatalf("invalid count: %s", diags.Error())
		}
		var count int
		err := gocty.FromCtyValue(val, &count)
		if err != nil {
			t.Fatalf("invalid count at %s: %s", rCfg.Count.Range(), err)
		}
		expander.SetResourceCount(moduleAddr, addr, count)
	default:
		expander.SetResourceSingle(moduleAddr, addr)
	}
}
