// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/e2e"
)

func TestTerraformProviderFunctions(t *testing.T) {
	// This test ensures that the terraform.io/builtin/terraform provider
	// remains available and that its three functions are available to be
	// called. This test is here because builtin providers are a bit of a
	// special case in the CLI layer which could in principle get accidentally
	// broken there even with deeper tests in the provider package itself
	// still passing.
	//
	// The tests in the provider's own package are authoritative for the
	// expected behavior of the functions. This test is focused on whether
	// the functions can be called at all, though it does some very light
	// testing of results for one specific input each. If the functions
	// are intentionally changed to produce different results for those
	// inputs in future then it may be appropriate to just update these
	// tests to match.

	t.Parallel()
	fixturePath := filepath.Join("testdata", "terraform-provider-funcs")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	//// INIT
	_, stderr, err := tf.Run("init")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	//// PLAN
	_, stderr, err = tf.Run("plan", "-out=tfplan")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}

	// The saved plan should include three planned output values containing
	// results from our function calls.
	plan, err := tf.Plan("tfplan")
	if err != nil {
		t.Fatalf("can't reload saved plan: %s", err)
	}

	gotOutputs := make(map[string]cty.Value, 3)
	for _, outputSrc := range plan.Changes.Outputs {
		output, err := outputSrc.Decode()
		if err != nil {
			t.Fatalf("can't decode planned change for %s: %s", outputSrc.Addr, err)
		}
		gotOutputs[output.Addr.String()] = output.After
	}
	wantOutputs := map[string]cty.Value{
		"output.exprencode": cty.StringVal(`[1, 2, 3]`),
		"output.tfvarsdecode": cty.ObjectVal(map[string]cty.Value{
			"baaa": cty.StringVal("🐑"),
			"boop": cty.StringVal("👃"),
		}),
		"output.tfvarsencode": cty.StringVal(`a = "👋"
b = "🐝"
c = "👓"
`),
	}
	if diff := cmp.Diff(wantOutputs, gotOutputs, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong output values\n%s", diff)
	}
}
