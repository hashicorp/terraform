package terraform

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// The tests in this file are for the extra warnings we produce when running
// validation in hint mode. All of the tests in here should be calling
// validate with Hints: true at some point, or else they'd be better
// placed in context_validate_test.go instead.

func TestContextValidateHint_redundantResourceDepends(t *testing.T) {
	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_thing": {
				Attributes: map[string]*configschema.Attribute{
					"name": {
						Type:     cty.String,
						Optional: true,
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"single_block": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"id": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
					"map_block": {
						Nesting: configschema.NestingMap,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"doodad": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
			},
		},
	})

	m := testModule(t, "validate-hint-redundant-resource-depends")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	gotDiags := c.Validate(m, &ValidateOpts{
		Hints: true,
	})
	var wantDiags tfdiags.Diagnostics
	wantDiags = wantDiags.Append(&tfdiags.HintMessage{
		Summary: "Redundant explicit dependency",
		Detail:  "There is already an implied dependency for test_thing.a at testdata/validate-hint-redundant-resource-depends/redundant-resource-depends.tf:23,10, so this declaration is redundant.",
		SourceRange: tfdiags.SourceRange{
			Filename: "testdata/validate-hint-redundant-resource-depends/redundant-resource-depends.tf",
			Start:    tfdiags.SourcePos{Line: 34, Column: 5, Byte: 404},
			End:      tfdiags.SourcePos{Line: 34, Column: 17, Byte: 416},
		},
	})
	wantDiags = wantDiags.Append(&tfdiags.HintMessage{
		Summary: "Redundant explicit dependency",
		Detail:  "There is already an explicit dependency for test_thing.b at testdata/validate-hint-redundant-resource-depends/redundant-resource-depends.tf:35,5, so this declaration is redundant.",
		SourceRange: tfdiags.SourceRange{
			Filename: "testdata/validate-hint-redundant-resource-depends/redundant-resource-depends.tf",
			Start:    tfdiags.SourcePos{Line: 36, Column: 5, Byte: 440},
			End:      tfdiags.SourcePos{Line: 36, Column: 17, Byte: 452},
		},
	})
	wantDiags = wantDiags.Append(&tfdiags.HintMessage{
		Summary: "Redundant explicit dependency",
		Detail:  "There is already an implied dependency for test_thing.c at testdata/validate-hint-redundant-resource-depends/redundant-resource-depends.tf:26,10, so this declaration is redundant.",
		SourceRange: tfdiags.SourceRange{
			Filename: "testdata/validate-hint-redundant-resource-depends/redundant-resource-depends.tf",
			Start:    tfdiags.SourcePos{Line: 37, Column: 5, Byte: 458},
			End:      tfdiags.SourcePos{Line: 37, Column: 17, Byte: 470},
		},
	})
	wantDiags = wantDiags.Append(&tfdiags.HintMessage{
		Summary: "Redundant explicit dependency",
		Detail:  "There is already an implied dependency for test_thing.d at testdata/validate-hint-redundant-resource-depends/redundant-resource-depends.tf:30,14, so this declaration is redundant.",
		SourceRange: tfdiags.SourceRange{
			Filename: "testdata/validate-hint-redundant-resource-depends/redundant-resource-depends.tf",
			Start:    tfdiags.SourcePos{Line: 38, Column: 5, Byte: 476},
			End:      tfdiags.SourcePos{Line: 38, Column: 17, Byte: 488},
		},
	})
	wantDiags = wantDiags.Append(&tfdiags.HintMessage{
		Summary: "Over-specified explicit dependency",
		Detail:  "Terraform references are between resources as a whole, and don't consider individual resource instances. The [0] instance key in this address has no effect.",
		SourceRange: tfdiags.SourceRange{
			Filename: "testdata/validate-hint-redundant-resource-depends/redundant-resource-depends.tf",
			Start:    tfdiags.SourcePos{Line: 39, Column: 5, Byte: 494},
			End:      tfdiags.SourcePos{Line: 39, Column: 20, Byte: 509},
		},
	})
	wantDiags = wantDiags.Append(&tfdiags.HintMessage{
		Summary: "Redundant explicit dependency",
		Detail:  "There is already an explicit dependency for test_thing.f at testdata/validate-hint-redundant-resource-depends/redundant-resource-depends.tf:39,5, so this declaration is redundant.",
		SourceRange: tfdiags.SourceRange{
			Filename: "testdata/validate-hint-redundant-resource-depends/redundant-resource-depends.tf",
			Start:    tfdiags.SourcePos{Line: 40, Column: 5, Byte: 515},
			End:      tfdiags.SourcePos{Line: 40, Column: 17, Byte: 527},
		},
	})
	assertDiagnosticsMatch(t, gotDiags, wantDiags)

}

func TestContextValidateHint_unnecessarySplatIndex(t *testing.T) {
	p := testProvider("test")
	p.GetProviderSchemaResponse = getProviderSchemaResponseFromProviderSchema(&ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_thing": {
				Attributes: map[string]*configschema.Attribute{
					"name": {
						Type:     cty.String,
						Optional: true,
					},
				},
			},
		},
	})

	m := testModule(t, "validate-hint-unnecessary-splat-index")
	c := testContext2(t, &ContextOpts{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	gotDiags := c.Validate(m, &ValidateOpts{
		Hints: true,
	})
	var wantDiags tfdiags.Diagnostics
	wantDiags = wantDiags.Append(&tfdiags.HintMessage{
		Summary: "Unnecessary splat expression with index",
		Detail:  "Looking up a particular index of a splat expression result is the same as just directly using the index instead of the splat operator.",
		SourceRange: tfdiags.SourceRange{
			Filename: "testdata/validate-hint-unnecessary-splat-index/unnecessary-splat-index.tf",
			Start:    tfdiags.SourcePos{Line: 34, Column: 5, Byte: 404},
			End:      tfdiags.SourcePos{Line: 34, Column: 17, Byte: 416},
		},
	})
	assertDiagnosticsMatch(t, gotDiags, wantDiags)

}
