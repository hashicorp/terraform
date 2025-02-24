// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package backendbase

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestBase_coerceError(t *testing.T) {
	// This tests that we return errors if type coersion fails.
	// This doesn't thoroughly test all cases because we're just delegating
	// to the configschema package's coersion function, which is already
	// tested in its own package.

	b := Base{
		Schema: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {
					Type:     cty.String,
					Optional: true,
				},
			},
		},
	}
	// This is a fake body just to give us something to correlate the
	// diagnostic attribute paths against so we can test that the
	// errors are properly annotated. In the real implementation
	// the command package logic would evaluate the diagnostics against
	// the real HCL body written by the end-user.
	//
	// Because we're using MockExprLiteral for the expressions here,
	// the source range for each expression is just the fake filename
	// "MockExprLiteral". If the PrepareConfig function fails to properly
	// annotate its diagnostics then the source range won't be populated
	// at all.
	body := hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcl.Attributes{
			"foo": {
				Expr: hcltest.MockExprLiteral(cty.StringVal("")),
			},
		},
	})

	t.Run("error", func(t *testing.T) {
		_, diags := b.PrepareConfig(cty.ObjectVal(map[string]cty.Value{
			// This is incorrect because the schema wants a string
			"foo": cty.MapValEmpty(cty.String),
		}))
		gotDiags := diags.InConfigBody(body, "")
		var wantDiags tfdiags.Diagnostics
		wantDiags = wantDiags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid backend configuration",
			Detail:   "The backend configuration is incorrect: .foo: string required.",
			Subject:  &hcl.Range{Filename: "MockExprLiteral"},
		})

		tfdiags.AssertDiagnosticsMatch(t, gotDiags, wantDiags)
	})
}

func TestBase_deprecatedArg(t *testing.T) {
	b := Base{
		Schema: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"not_deprecated": {
					Type:     cty.String,
					Optional: true,
				},
				"deprecated": {
					Type:       cty.String,
					Optional:   true,
					Deprecated: true,
				},
			},
			BlockTypes: map[string]*configschema.NestedBlock{
				"nested": {
					Nesting: configschema.NestingList,
					Block: configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"deprecated": {
								Type:       cty.String,
								Optional:   true,
								Deprecated: true,
							},
						},
					},
				},
			},
		},
	}
	// This is a fake body just to give us something to correlate the
	// diagnostic attribute paths against so we can test that the
	// warnings are properly annotated. In the real implementation
	// the command package logic would evaluate the diagnostics against
	// the real HCL body written by the end-user.
	//
	// Because we're using MockExprLiteral for the expressions here,
	// the source range for each expression is just the fake filename
	// "MockExprLiteral". If the PrepareConfig function fails to properly
	// annotate its diagnostics then the source range won't be populated
	// at all.
	body := hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcl.Attributes{
			"deprecated": {
				Expr: hcltest.MockExprLiteral(cty.StringVal("")),
			},
		},
		Blocks: hcl.Blocks{
			{
				Type: "nested",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"deprecated": {
							Expr: hcltest.MockExprLiteral(cty.StringVal("")),
						},
					},
				}),
			},
			{
				Type: "nested",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"deprecated": {
							Expr: hcltest.MockExprLiteral(cty.StringVal("")),
						},
					},
				}),
			},
		},
	})

	t.Run("nothing deprecated", func(t *testing.T) {
		got, diags := b.PrepareConfig(cty.ObjectVal(map[string]cty.Value{
			"not_deprecated": cty.StringVal("hello"),
		}))
		if len(diags) != 0 {
			t.Errorf("unexpected diagnostics: %s", diags.ErrWithWarnings().Error())
		}
		want := cty.ObjectVal(map[string]cty.Value{
			"deprecated":     cty.NullVal(cty.String),
			"not_deprecated": cty.StringVal("hello"),
			"nested": cty.ListValEmpty(cty.Object(map[string]cty.Type{
				"deprecated": cty.String,
			})),
		})
		if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result\n%s", diff)
		}
	})
	t.Run("toplevel deprecated", func(t *testing.T) {
		_, diags := b.PrepareConfig(cty.ObjectVal(map[string]cty.Value{
			"deprecated": cty.StringVal("hello"),
		}))

		gotDiags := diags.InConfigBody(body, "")
		var wantDiags tfdiags.Diagnostics
		wantDiags = wantDiags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Deprecated provider argument",
			Detail:   "The argument .deprecated is deprecated. Refer to the backend documentation for more information.",
			Subject:  &hcl.Range{Filename: "MockExprLiteral"},
		})
		tfdiags.AssertDiagnosticsMatch(t, wantDiags, gotDiags)
	})
	t.Run("nested deprecated", func(t *testing.T) {
		_, diags := b.PrepareConfig(cty.ObjectVal(map[string]cty.Value{
			"nested": cty.TupleVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{
					"deprecated": cty.StringVal("hello"),
				}),
				cty.ObjectVal(map[string]cty.Value{
					"deprecated": cty.StringVal("hello"),
				}),
			}),
		}))
		gotDiags := diags.InConfigBody(body, "")
		var wantDiags tfdiags.Diagnostics
		wantDiags = wantDiags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Deprecated provider argument",
			Detail:   "The argument .nested[0].deprecated is deprecated. Refer to the backend documentation for more information.",
			Subject:  &hcl.Range{Filename: "MockExprLiteral"},
		})
		wantDiags = wantDiags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Deprecated provider argument",
			Detail:   "The argument .nested[1].deprecated is deprecated. Refer to the backend documentation for more information.",
			Subject:  &hcl.Range{Filename: "MockExprLiteral"},
		})
		tfdiags.AssertDiagnosticsMatch(t, wantDiags, gotDiags)
	})
}

func TestBase_nullCrash(t *testing.T) {
	// This test ensures that we don't crash while applying defaults to
	// a null value

	b := Base{
		Schema: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"foo": {
					Type:     cty.String,
					Required: true,
				},
			},
		},
		SDKLikeDefaults: SDKLikeDefaults{
			"foo": {
				Fallback: "fallback",
			},
		},
	}

	t.Run("error", func(t *testing.T) {
		// We pass an explicit null value here to simulate an interrupt
		_, gotDiags := b.PrepareConfig(cty.NullVal(cty.Object(map[string]cty.Type{
			"foo": cty.String,
		})))
		var wantDiags tfdiags.Diagnostics
		wantDiags = wantDiags.Append(
			&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid backend configuration",
				Detail:   "The backend configuration is incorrect: attribute \"foo\" is required.",
			})
		if diff := cmp.Diff(wantDiags.ForRPC(), gotDiags.ForRPC()); diff != "" {
			t.Errorf("wrong diagnostics\n%s", diff)
		}
	})
}
