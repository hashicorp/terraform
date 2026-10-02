// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestSelfInstanceMissingIndexMatchesAggregate(t *testing.T) {
	for _, kind := range []string{"count", "for_each"} {
		for _, operation := range []walkOperation{walkEval, walkValidate, walkImport} {
			t.Run(kind+"/"+operation.String(), func(t *testing.T) {
				d, first := selfInstanceTestData(4, kind, false)
				d.Operation = operation
				var key addrs.InstanceKey = addrs.IntKey(8)
				index := cty.NumberIntVal(8)
				if kind == "for_each" {
					key = addrs.StringKey("missing")
					index = cty.StringVal("missing")
				}
				rng := tfdiags.SourceRange{Filename: "self.tf", Start: tfdiags.SourcePos{Line: 1, Column: 1}, End: tfdiags.SourcePos{Line: 1, Column: 5}}
				all, wantDiags := d.GetResource(first.Resource, rng)
				want, indexDiags := hcl.Index(all, index, rng.ToHCL().Ptr())
				wantDiags = wantDiags.Append(indexDiags)
				got, gotDiags := d.GetResourceInstance(first.Resource.Instance(key), rng)
				if !got.RawEquals(want) || !cmp.Equal(gotDiags.ForRPC(), wantDiags.ForRPC()) {
					t.Fatalf("missing index changed value or diagnostics: got %s %s; want %s %s", got.GoString(), gotDiags.Err(), want.GoString(), wantDiags.Err())
				}
			})
		}
	}
}

func TestSelfInstanceCountHoleMatchesAggregate(t *testing.T) {
	d, first := selfInstanceTestData(4, "count", false)
	d.Evaluator.State.ForgetResourceInstanceAll(first.Resource.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance))
	all, diags := d.GetResource(first.Resource, tfdiags.SourceRange{})
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
	got, diags := d.GetResourceInstance(first.Resource.Instance(addrs.IntKey(1)), tfdiags.SourceRange{})
	if diags.HasErrors() || !got.RawEquals(all.Index(cty.NumberIntVal(1))) {
		t.Fatalf("missing count instance changed: %s", diags.Err())
	}
}

func TestSelfInstancePlannedValueMatchesAggregate(t *testing.T) {
	d, first := selfInstanceTestData(4, "for_each", false)
	addr := first.Absolute(addrs.RootModuleInstance)
	provider := addrs.AbsProviderConfig{Provider: addrs.NewDefaultProvider("test"), Module: addrs.RootModule}
	d.Evaluator.State.SetResourceInstanceCurrent(addr, &states.ResourceInstanceObjectSrc{Status: states.ObjectPlanned}, provider)
	after := cty.ObjectVal(map[string]cty.Value{"id": cty.UnknownVal(cty.String), "secret": cty.StringVal("planned")})
	d.Evaluator.Changes.AppendResourceInstanceChange(&plans.ResourceInstanceChange{Addr: addr, Change: plans.Change{Action: plans.Update, After: after}})
	all, wantDiags := d.GetResource(first.Resource, tfdiags.SourceRange{})
	got, gotDiags := d.GetResourceInstance(first, tfdiags.SourceRange{})
	if wantDiags.HasErrors() || gotDiags.HasErrors() || !got.RawEquals(all.GetAttr(string(first.Key.(addrs.StringKey)))) {
		t.Fatalf("planned value differs: %s", gotDiags.Err())
	}
}
