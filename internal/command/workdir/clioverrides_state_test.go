// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty/cty"
)

func TestConfigOverrideState_OverrideConfig(t *testing.T) {
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	b := configs.SynthBody("synth", map[string]cty.Value{
		"foo": cty.StringVal("bar"),
	})

	expectedOverride := "value from overrides"
	o := &ConfigOverrideState{
		Overrides: []byte(`{
			"foo": "value from overrides"
		}`),
	}

	// Use the ConfigOverrideState to override the provided hcl.Body
	ob, err := o.OverrideConfig(schema, b)
	if err != nil {
		t.Fatal()
	}

	// Assert that the override is in effect
	attrs, _ := ob.JustAttributes()
	foo, ok := attrs["foo"]
	if !ok {
		t.Fatal()
	}

	evalCtx := &hcl.EvalContext{
		Variables: map[string]cty.Value{},
	}
	val, diags := foo.Expr.Value(evalCtx)
	if diags.HasErrors() {
		t.Fatal()
	}
	if val.Equals(cty.StringVal(expectedOverride)) == cty.False {
		t.Fatalf("expected value to be %q, got %q", expectedOverride, val.AsString())
	}
}
