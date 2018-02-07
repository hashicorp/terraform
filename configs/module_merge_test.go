package configs

import (
	"testing"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
)

func TestModuleOverrideVariable(t *testing.T) {
	mod, diags := testModuleFromDir("test-fixtures/valid-modules/override-variable")
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatalf("module is nil")
	}

	got := mod.Variables
	want := map[string]*Variable{
		"fully_overridden": {
			Name:           "fully_overridden",
			Description:    "b_override description",
			DescriptionSet: true,
			Default:        cty.StringVal("b_override"),
			TypeHint:       TypeHintString,
			DeclRange: hcl.Range{
				Filename: "test-fixtures/valid-modules/override-variable/primary.tf",
				Start: hcl.Pos{
					Line:   1,
					Column: 1,
					Byte:   0,
				},
				End: hcl.Pos{
					Line:   1,
					Column: 28,
					Byte:   27,
				},
			},
		},
		"partially_overridden": {
			Name:           "partially_overridden",
			Description:    "base description",
			DescriptionSet: true,
			Default:        cty.StringVal("b_override partial"),
			TypeHint:       TypeHintString,
			DeclRange: hcl.Range{
				Filename: "test-fixtures/valid-modules/override-variable/primary.tf",
				Start: hcl.Pos{
					Line:   7,
					Column: 1,
					Byte:   103,
				},
				End: hcl.Pos{
					Line:   7,
					Column: 32,
					Byte:   134,
				},
			},
		},
	}
	assertResultDeepEqual(t, got, want)
}
