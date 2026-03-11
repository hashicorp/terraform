// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
)

func TestVariableValidationTransformer(t *testing.T) {
	// This is a unit test focused on just the validation transformer's
	// behavior, which assumes that the caller correctly arranges for
	// its invariants to be met:
	//  1. all input variable evaluation nodes must already be present
	//     in the graph before running the transformer.
	//  2. the reference transformer must run after this transformer.
	//
	// To avoid this depending on transformers other than the one we're
	// testing we'll arrange for those to be met in a rather artificial
	// way. Our integration tests complement this by verifying that
	// the variable validation feature as a whole is working. For
	// example: [TestContext2Plan_variableValidation].

	g := &Graph{}
	fooNode := &nodeTestOnlyInputVariable{
		configAddr: addrs.InputVariable{Name: "foo"}.InModule(addrs.RootModule),
		rules: []*configs.CheckRule{
			{
				// The condition contains a self-reference, which is required
				// for a realistic input variable validation because otherwise
				// it wouldn't actually be checking the variable it's
				// supposed to be validating. (This transformer is not the
				// one responsible for validating that though, so it's
				// okay for the examples below to not meet that requirement.)
				Condition:    hcltest.MockExprTraversalSrc("var.foo"),
				ErrorMessage: hcltest.MockExprLiteral(cty.StringVal("wrong")),
			},
		},
	}
	barNode := &nodeTestOnlyInputVariable{
		configAddr: addrs.InputVariable{Name: "bar"}.InModule(addrs.RootModule),
		rules: []*configs.CheckRule{
			{
				// The condition of this one refers to var.foo
				Condition:    hcltest.MockExprTraversalSrc("var.foo"),
				ErrorMessage: hcltest.MockExprLiteral(cty.StringVal("wrong")),
			},
		},
	}
	bazNode := &nodeTestOnlyInputVariable{
		configAddr: addrs.InputVariable{Name: "baz"}.InModule(addrs.RootModule),
		rules: []*configs.CheckRule{
			{
				// The error message of this one refers to var.foo
				Condition:    hcltest.MockExprLiteral(cty.False),
				ErrorMessage: hcltest.MockExprTraversalSrc("var.foo"),
			},
		},
	}
	g.Add(fooNode)
	g.Add(barNode)
	g.Add(bazNode)

	transformer := &variableValidationTransformer{}
	err := transformer.Transform(g)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	gotStr := strings.TrimSpace(g.String())
	wantStr := strings.TrimSpace(`
var.bar (test fake)
var.bar (validation)
  var.bar (test fake)
var.baz (test fake)
var.baz (validation)
  var.baz (test fake)
var.foo (test fake)
var.foo (validation)
  var.foo (test fake)
`)
	if diff := cmp.Diff(wantStr, gotStr); diff != "" {
		t.Errorf("wrong graph after transform\n%s", diff)
	}

	// This transformer is not responsible for wiring up dependencies based
	// on references -- that's ReferenceTransformer's job -- but we'll
	// verify that the nodes that were added by this transformer do at least
	// report the references we expect them to report, in the way that
	// ReferenceTransformer would expect.
	gotRefs := map[string]map[string]struct{}{}
	for _, v := range g.Vertices() {
		v, ok := v.(*nodeVariableValidation) // the type of all nodes that this transformer adds
		if !ok {
			continue
		}
		var _ GraphNodeReferencer = v // static assertion just to make sure we'll fail to compile if GraphNodeReferencer changes later

		refs := v.References()
		gotRefs[v.Name()] = map[string]struct{}{}
		for _, ref := range refs {
			gotRefs[v.Name()][ref.Subject.String()] = struct{}{}
		}
	}
	wantRefs := map[string]map[string]struct{}{
		"var.bar (validation)": {
			"var.foo": struct{}{},
		},
		"var.baz (validation)": {
			"var.foo": struct{}{},
		},
		"var.foo (validation)": {},
	}
	if diff := cmp.Diff(wantRefs, gotRefs); diff != "" {
		t.Errorf("wrong references for the added nodes\n%s", diff)
	}
}

type nodeTestOnlyInputVariable struct {
	configAddr addrs.ConfigInputVariable
	rules      []*configs.CheckRule
}

var _ graphNodeValidatableVariable = (*nodeTestOnlyInputVariable)(nil)

func (n *nodeTestOnlyInputVariable) Name() string {
	return fmt.Sprintf("%s (test fake)", n.configAddr)
}

// variableValidationRules implements [graphNodeValidatableVariable].
func (n *nodeTestOnlyInputVariable) variableValidationRules() (addrs.ConfigInputVariable, []*configs.CheckRule, hcl.Range) {
	return n.configAddr, n.rules, hcl.Range{
		Filename: "test",
		Start:    hcl.InitialPos,
		End:      hcl.InitialPos,
	}
}
