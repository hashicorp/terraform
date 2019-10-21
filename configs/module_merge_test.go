package configs

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
)

func TestModuleOverrideVariable(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/valid-modules/override-variable")
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
			Type:           cty.String,
			ParsingMode:    VariableParseLiteral,
			DeclRange: hcl.Range{
				Filename: "testdata/valid-modules/override-variable/primary.tf",
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
			Type:           cty.String,
			ParsingMode:    VariableParseLiteral,
			DeclRange: hcl.Range{
				Filename: "testdata/valid-modules/override-variable/primary.tf",
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

func TestModuleOverrideModule(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/valid-modules/override-module")
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatalf("module is nil")
	}

	if _, exists := mod.ModuleCalls["example"]; !exists {
		t.Fatalf("no module 'example'")
	}
	if len(mod.ModuleCalls) != 1 {
		t.Fatalf("wrong number of module calls in result %d; want 1", len(mod.ModuleCalls))
	}

	got := mod.ModuleCalls["example"]
	want := &ModuleCall{
		Name:       "example",
		SourceAddr: "./example2-a_override",
		SourceAddrRange: hcl.Range{
			Filename: "testdata/valid-modules/override-module/a_override.tf",
			Start: hcl.Pos{
				Line:   3,
				Column: 12,
				Byte:   31,
			},
			End: hcl.Pos{
				Line:   3,
				Column: 35,
				Byte:   54,
			},
		},
		SourceSet: true,
		DeclRange: hcl.Range{
			Filename: "testdata/valid-modules/override-module/primary.tf",
			Start: hcl.Pos{
				Line:   2,
				Column: 1,
				Byte:   1,
			},
			End: hcl.Pos{
				Line:   2,
				Column: 17,
				Byte:   17,
			},
		},
	}

	// We're going to extract and nil out our hcl.Body here because DeepEqual
	// is not a useful way to assert on that.
	gotConfig := got.Config
	got.Config = nil

	assertResultDeepEqual(t, got, want)

	type content struct {
		Kept  *string `hcl:"kept"`
		Foo   *string `hcl:"foo"`
		New   *string `hcl:"new"`
		Newer *string `hcl:"newer"`
	}
	var gotArgs content
	diags = gohcl.DecodeBody(gotConfig, nil, &gotArgs)
	assertNoDiagnostics(t, diags)

	wantArgs := content{
		Kept:  stringPtr("primary kept"),
		Foo:   stringPtr("a_override foo"),
		New:   stringPtr("b_override new"),
		Newer: stringPtr("b_override newer"),
	}

	assertResultDeepEqual(t, gotArgs, wantArgs)
}

func TestModuleOverrideDynamic(t *testing.T) {
	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "foo"},
			{Type: "dynamic", LabelNames: []string{"type"}},
		},
	}

	t.Run("base is dynamic", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-dynamic-block-base")
		assertNoDiagnostics(t, diags)
		if mod == nil {
			t.Fatalf("module is nil")
		}

		if _, exists := mod.ManagedResources["test.foo"]; !exists {
			t.Fatalf("no module 'example'")
		}
		if len(mod.ManagedResources) != 1 {
			t.Fatalf("wrong number of managed resources in result %d; want 1", len(mod.ManagedResources))
		}

		body := mod.ManagedResources["test.foo"].Config
		content, diags := body.Content(schema)
		assertNoDiagnostics(t, diags)

		if len(content.Blocks) != 1 {
			t.Fatalf("wrong number of blocks in result %d; want 1", len(content.Blocks))
		}
		if got, want := content.Blocks[0].Type, "foo"; got != want {
			t.Fatalf("wrong block type %q; want %q", got, want)
		}
	})
	t.Run("override is dynamic", func(t *testing.T) {
		mod, diags := testModuleFromDir("testdata/valid-modules/override-dynamic-block-override")
		assertNoDiagnostics(t, diags)
		if mod == nil {
			t.Fatalf("module is nil")
		}

		if _, exists := mod.ManagedResources["test.foo"]; !exists {
			t.Fatalf("no module 'example'")
		}
		if len(mod.ManagedResources) != 1 {
			t.Fatalf("wrong number of managed resources in result %d; want 1", len(mod.ManagedResources))
		}

		body := mod.ManagedResources["test.foo"].Config
		content, diags := body.Content(schema)
		assertNoDiagnostics(t, diags)

		if len(content.Blocks) != 1 {
			t.Fatalf("wrong number of blocks in result %d; want 1", len(content.Blocks))
		}
		if got, want := content.Blocks[0].Type, "dynamic"; got != want {
			t.Fatalf("wrong block type %q; want %q", got, want)
		}
		if got, want := content.Blocks[0].Labels[0], "foo"; got != want {
			t.Fatalf("wrong dynamic block label %q; want %q", got, want)
		}
	})
}
