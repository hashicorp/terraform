// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
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
			Nullable:       false,
			NullableSet:    true,
			Type:           cty.String,
			ConstraintType: cty.String,
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
			Nullable:       true,
			NullableSet:    false,
			Type:           cty.String,
			ConstraintType: cty.String,
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
		Name:          "example",
		SourceAddr:    addrs.ModuleSourceLocal("./example2-a_override"),
		SourceAddrRaw: "./example2-a_override",
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
		DependsOn: []hcl.Traversal{
			{
				hcl.TraverseRoot{
					Name: "null_resource",
					SrcRange: hcl.Range{
						Filename: "testdata/valid-modules/override-module/primary.tf",
						Start:    hcl.Pos{Line: 11, Column: 17, Byte: 149},
						End:      hcl.Pos{Line: 11, Column: 30, Byte: 162},
					},
				},
				hcl.TraverseAttr{
					Name: "test",
					SrcRange: hcl.Range{
						Filename: "testdata/valid-modules/override-module/primary.tf",
						Start:    hcl.Pos{Line: 11, Column: 30, Byte: 162},
						End:      hcl.Pos{Line: 11, Column: 35, Byte: 167},
					},
				},
			},
		},
		Providers: []PassedProviderConfig{
			{
				InChild: &ProviderConfigRef{
					Name: "test",
					NameRange: hcl.Range{
						Filename: "testdata/valid-modules/override-module/b_override.tf",
						Start:    hcl.Pos{Line: 7, Column: 5, Byte: 97},
						End:      hcl.Pos{Line: 7, Column: 9, Byte: 101},
					},
				},
				InParent: &ProviderConfigRef{
					Name: "test",
					NameRange: hcl.Range{
						Filename: "testdata/valid-modules/override-module/b_override.tf",
						Start:    hcl.Pos{Line: 7, Column: 12, Byte: 104},
						End:      hcl.Pos{Line: 7, Column: 16, Byte: 108},
					},
					Alias: "b_override",
					AliasRange: &hcl.Range{
						Filename: "testdata/valid-modules/override-module/b_override.tf",
						Start:    hcl.Pos{Line: 7, Column: 16, Byte: 108},
						End:      hcl.Pos{Line: 7, Column: 27, Byte: 119},
					},
				},
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

func TestModuleOverrideSensitiveVariable(t *testing.T) {
	type testCase struct {
		sensitive    bool
		sensitiveSet bool
	}
	cases := map[string]testCase{
		"false_true": {
			sensitive:    true,
			sensitiveSet: true,
		},
		"true_false": {
			sensitive:    false,
			sensitiveSet: true,
		},
		"false_false_true": {
			sensitive:    true,
			sensitiveSet: true,
		},
		"true_true_false": {
			sensitive:    false,
			sensitiveSet: true,
		},
		"false_true_false": {
			sensitive:    false,
			sensitiveSet: true,
		},
		"true_false_true": {
			sensitive:    true,
			sensitiveSet: true,
		},
	}

	mod, diags := testModuleFromDir("testdata/valid-modules/override-variable-sensitive")

	assertNoDiagnostics(t, diags)

	if mod == nil {
		t.Fatalf("module is nil")
	}

	got := mod.Variables

	for v, want := range cases {
		t.Run(fmt.Sprintf("variable %s", v), func(t *testing.T) {
			if got[v].Sensitive != want.sensitive {
				t.Errorf("wrong result for sensitive\ngot: %t want: %t", got[v].Sensitive, want.sensitive)
			}

			if got[v].SensitiveSet != want.sensitiveSet {
				t.Errorf("wrong result for sensitive set\ngot: %t want: %t", got[v].Sensitive, want.sensitive)
			}
		})
	}
}

func TestModuleOverrideResourceFQNs(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/valid-modules/override-resource-provider")
	assertNoDiagnostics(t, diags)

	got := mod.ManagedResources["test_instance.explicit"]
	wantProvider := addrs.NewProvider(addrs.DefaultProviderRegistryHost, "bar", "test")
	wantProviderCfg := &ProviderConfigRef{
		Name: "bar-test",
		NameRange: hcl.Range{
			Filename: "testdata/valid-modules/override-resource-provider/a_override.tf",
			Start:    hcl.Pos{Line: 2, Column: 14, Byte: 51},
			End:      hcl.Pos{Line: 2, Column: 22, Byte: 59},
		},
	}

	if !got.Provider.Equals(wantProvider) {
		t.Fatalf("wrong provider %s, want %s", got.Provider, wantProvider)
	}
	assertResultDeepEqual(t, got.ProviderConfigRef, wantProviderCfg)

	// now verify that a resource with no provider config falls back to default
	got = mod.ManagedResources["test_instance.default"]
	wantProvider = addrs.NewDefaultProvider("test")
	if !got.Provider.Equals(wantProvider) {
		t.Fatalf("wrong provider %s, want %s", got.Provider, wantProvider)
	}
	if got.ProviderConfigRef != nil {
		t.Fatalf("wrong result: found provider config ref %s, expected nil", got.ProviderConfigRef)
	}
}

func TestModuleOverrideIgnoreAllChanges(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/valid-modules/override-ignore-changes")
	assertNoDiagnostics(t, diags)

	r := mod.ManagedResources["test_instance.foo"]
	if !r.Managed.IgnoreAllChanges {
		t.Fatalf("wrong result: expected r.Managed.IgnoreAllChanges to be true")
	}
}

// This tests the override behavior of action blocks and action_triggers inside resources.
func TestModuleOverride_action_and_trigger(t *testing.T) {
	mod, diags := testModuleFromDirWithExperiments("testdata/valid-modules/override-action-and-trigger")
	assertNoDiagnostics(t, diags)

	if len(mod.Actions) != 2 {
		t.Fatalf("wrong number of actions: %d\n", len(mod.Actions))
	}

	// verify that the action has attr foo = baz (override)
	got := mod.Actions["action.test_action.test"]
	want := &Action{
		Name:              "test",
		Type:              "test_action",
		Config:            nil,
		Count:             nil,
		ForEach:           nil,
		DependsOn:         nil,
		ProviderConfigRef: nil,
		Provider:          addrs.NewProvider(addrs.DefaultProviderRegistryHost, "hashicorp", "test"),
		DeclRange: hcl.Range{
			Filename: "testdata/valid-modules/override-action-and-trigger/main.tf",
			Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
			End:      hcl.Pos{Line: 1, Column: 28, Byte: 27},
		},
		TypeRange: hcl.Range{
			Filename: "testdata/valid-modules/override-action-and-trigger/main.tf",
			Start:    hcl.Pos{Line: 1, Column: 8, Byte: 7},
			End:      hcl.Pos{Line: 1, Column: 21, Byte: 20},
		},
	}

	// We're going to extract and nil out our hcl.Body here because DeepEqual
	// is not a useful way to assert on that.
	gotConfig := got.Config
	got.Config = nil

	assertResultDeepEqual(t, got, want)

	// now to check that config
	type content struct {
		Foo *string `hcl:"foo"`
	}
	var gotArgs content
	diags = gohcl.DecodeBody(gotConfig, nil, &gotArgs)
	assertNoDiagnostics(t, diags)

	wantArgs := content{
		Foo: stringPtr("baz"),
	}
	assertResultDeepEqual(t, gotArgs, wantArgs)

	if _, exists := mod.ManagedResources["test_instance.test"]; !exists {
		t.Fatalf("no resource 'test_instance.test'")
	}
	if len(mod.ManagedResources) != 1 {
		t.Fatalf("wrong number of managed resources in result %d; want 1", len(mod.ManagedResources))
	}

	r := mod.ManagedResources["test_instance.test"].Managed
	assertResultDeepEqual(t, len(r.ActionTriggers), 1)

	// verify the resource action trigger event changed
	at := mod.ManagedResources["test_instance.test"].Managed.ActionTriggers[0]
	assertResultDeepEqual(t, at.Events, []ActionTriggerEvent{AfterDestroy})
}
