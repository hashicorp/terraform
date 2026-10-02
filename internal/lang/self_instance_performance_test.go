// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type instanceDataForTests struct {
	*dataForTests
	aggregateCalls, instanceCalls int
}

func (d *instanceDataForTests) GetResource(addr addrs.Resource, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	d.aggregateCalls++
	return d.dataForTests.GetResource(addr, rng)
}
func (d *instanceDataForTests) GetResourceInstance(addr addrs.ResourceInstance, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	d.instanceCalls++
	return cty.ObjectVal(map[string]cty.Value{"attr": cty.StringVal("value")}), nil
}

func TestScopeSelfOnlyResolvesTargetInstance(t *testing.T) {
	resource := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "null_resource", Name: "each"}
	d := &instanceDataForTests{dataForTests: &dataForTests{Resources: map[string]cty.Value{"null_resource.each": cty.ObjectVal(map[string]cty.Value{"target": cty.ObjectVal(map[string]cty.Value{"attr": cty.StringVal("value")})})}}}
	scope := &Scope{Data: d, ParseRef: addrs.ParseRef, SelfAddr: resource.Instance(addrs.StringKey("target"))}
	expr, err := hclsyntax.ParseExpression([]byte(`self.attr == "value" && self.attr == "value"`), "self.tf", hcl.InitialPos)
	if err.HasErrors() {
		t.Fatal(err)
	}
	got, diags := scope.EvalExpr(expr, cty.Bool)
	if diags.HasErrors() || !got.RawEquals(cty.True) {
		t.Fatalf("value %s diagnostics %s", got.GoString(), diags.Err())
	}
	if d.aggregateCalls != 0 || d.instanceCalls != 2 {
		t.Fatalf("self resolved aggregate %d times and target %d times; want aggregate 0 and target 2", d.aggregateCalls, d.instanceCalls)
	}
}
