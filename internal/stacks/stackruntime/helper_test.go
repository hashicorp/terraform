// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// This file has helper functions used by other tests. It doesn't contain any
// test cases of its own.

// loadConfigForTest is a test helper that tries to open bundleRoot as a
// source bundle, and then if successful tries to load the given source address
// from it as a stack configuration. If any part of the operation fails then
// it halts execution of the test and doesn't return.
func loadConfigForTest(t *testing.T, bundleRoot string, configSourceAddr string) *stackconfig.Config {
	t.Helper()
	sources, err := sourcebundle.OpenDir(bundleRoot)
	if err != nil {
		t.Fatalf("cannot load source bundle: %s", err)
	}

	// We force using remote source addresses here because that avoids
	// us having to deal with the extra version constraints argument
	// that registry sources require. Exactly what source address type
	// we use isn't relevant for tests in this package, since it's
	// the sourcebundle package's responsibility to make sure its
	// abstraction works for all of the source types.
	sourceAddr, err := sourceaddrs.ParseRemoteSource(configSourceAddr)
	if err != nil {
		t.Fatalf("invalid config source address: %s", err)
	}

	cfg, diags := stackconfig.LoadConfigDir(sourceAddr, sources)
	reportDiagnosticsForTest(t, diags)
	return cfg
}

func mainBundleSourceAddrStr(dirName string) string {
	return "git::https://example.com/test.git//" + dirName
}

// loadMainBundleConfigForTest is a convenience wrapper around
// loadConfigForTest that knows the location and package address of our
// "main" source bundle, in ./testdata/mainbundle, so that we can use that
// conveniently without duplicating its location and synthetic package address
// in every single test function.
//
// dirName should begin with the name of a subdirectory that's present in
// ./testdata/mainbundle/test . It can optionally refer to subdirectories
// thereof, using forward slashes as the path separator just as we'd do
// in the subdirectory portion of a remote source address (which is exactly
// what we're using this as.)
func loadMainBundleConfigForTest(t *testing.T, dirName string) *stackconfig.Config {
	t.Helper()
	fullSourceAddr := mainBundleSourceAddrStr(dirName)
	return loadConfigForTest(t, "./testdata/mainbundle", fullSourceAddr)
}

// reportDiagnosticsForTest creates a test log entry for every diagnostic in
// the given diags, and halts the test if any of them are error diagnostics.
func reportDiagnosticsForTest(t *testing.T, diags tfdiags.Diagnostics) {
	t.Helper()
	for _, diag := range diags {
		var b strings.Builder
		desc := diag.Description()
		locs := diag.Source()

		switch sev := diag.Severity(); sev {
		case tfdiags.Error:
			b.WriteString("Error: ")
		case tfdiags.Warning:
			b.WriteString("Warning: ")
		default:
			t.Errorf("unsupported diagnostic type %s", sev)
		}
		b.WriteString(desc.Summary)
		if desc.Address != "" {
			b.WriteString("\nwith ")
			b.WriteString(desc.Summary)
		}
		if locs.Subject != nil {
			b.WriteString("\nat ")
			b.WriteString(locs.Subject.StartString())
		}
		if desc.Detail != "" {
			b.WriteString("\n\n")
			b.WriteString(desc.Detail)
		}
		t.Log(b.String())
	}
	if diags.HasErrors() {
		t.FailNow()
	}
}

// appliedChangeSortKey returns a string that can be used to sort applied
// changes in a predictable order for testing purposes. This is used to
// ensure that we can compare applied changes in a consistent way across
// different test runs.
func appliedChangeSortKey(change stackstate.AppliedChange) string {
	switch change := change.(type) {
	case *stackstate.AppliedChangeResourceInstanceObject:
		return change.ResourceInstanceObjectAddr.String()
	case *stackstate.AppliedChangeComponentInstance:
		return change.ComponentInstanceAddr.String()
	case *stackstate.AppliedChangeDiscardKeys:
		// There should only be a single discard keys in a plan, so we can just
		// return a static string here.
		return "discard"
	default:
		// This is only going to happen during tests, so we can panic here.
		panic(fmt.Errorf("unrecognized applied change type: %T", change))
	}
}

// plannedChangeSortKey returns a string that can be used to sort planned
// changes in a predictable order for testing purposes. This is used to
// ensure that we can compare planned changes in a consistent way across
// different test runs.
func plannedChangeSortKey(change stackplan.PlannedChange) string {
	switch change := change.(type) {
	case *stackplan.PlannedChangeRootInputValue:
		return change.Addr.String()
	case *stackplan.PlannedChangeComponentInstance:
		return change.Addr.String()
	case *stackplan.PlannedChangeResourceInstancePlanned:
		return change.ResourceInstanceObjectAddr.String()
	case *stackplan.PlannedChangeOutputValue:
		return change.Addr.String()
	case *stackplan.PlannedChangeHeader:
		// There should only be a single header in a plan, so we can just return
		// a static string here.
		return "header"
	case *stackplan.PlannedChangeApplyable:
		// There should only be a single applyable marker in a plan, so we can
		// just return a static string here.
		return "applyable"
	default:
		// This is only going to happen during tests, so we can panic here.
		panic(fmt.Errorf("unrecognized planned change type: %T", change))
	}
}

func mustPlanDynamicValue(v cty.Value) plans.DynamicValue {
	ret, err := plans.NewDynamicValue(v, v.Type())
	if err != nil {
		panic(err)
	}
	return ret
}

func mustPlanDynamicValueDynamicType(v cty.Value) plans.DynamicValue {
	ret, err := plans.NewDynamicValue(v, cty.DynamicPseudoType)
	if err != nil {
		panic(err)
	}
	return ret
}

func mustPlanDynamicValueSchema(v cty.Value, block *configschema.Block) plans.DynamicValue {
	ty := block.ImpliedType()
	ret, err := plans.NewDynamicValue(v, ty)
	if err != nil {
		panic(err)
	}
	return ret
}

func mustMarshalJSONAttrs(attrs map[string]interface{}) []byte {
	jsonAttrs, err := json.Marshal(attrs)
	if err != nil {
		panic(err)
	}
	return jsonAttrs
}
