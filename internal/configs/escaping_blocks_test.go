// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

// "Escaping Blocks" are a special mechanism we have inside our block types
// that accept a mixture of meta-arguments and externally-defined arguments,
// which allow an author to force particular argument names to be interpreted
// as externally-defined even if they have the same name as a meta-argument.
//
// An escaping block is a block with the special type name "_" (just an
// underscore), and is allowed at the top-level of any resource, data, or
// module block. It intentionally has a rather "odd" look so that it stands
// out as something special and rare.
//
// This is not something we expect to see used a lot, but it's an important
// part of our strategy to evolve the Terraform language in future using
// editions, so that later editions can define new meta-arguments without
// blocking access to externally-defined arguments of the same name.
//
// We should still define new meta-arguments with care to avoid squatting on
// commonly-used names, but we can't see all modules and all providers in
// the world and so this is an escape hatch for edge cases. Module migration
// tools for future editions that define new meta-arguments should detect
// collisions and automatically migrate existing arguments into an escaping
// block.

func TestEscapingBlockResource(t *testing.T) {
	// (this also tests escaping blocks in provisioner blocks, because
	// they only appear nested inside resource blocks.)

	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir("testdata/escaping-blocks/resource")
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	rc := mod.ManagedResources["foo.bar"]
	if rc == nil {
		t.Fatal("no managed resource named foo.bar")
	}

	t.Run("resource body", func(t *testing.T) {
		if got := rc.Count; got == nil {
			t.Errorf("count not set; want count = 2")
		} else {
			got, diags := got.Value(nil)
			assertNoDiagnostics(t, diags)
			if want := cty.NumberIntVal(2); !want.RawEquals(got) {
				t.Errorf("wrong count\ngot:  %#v\nwant: %#v", got, want)
			}
		}
		if got, want := rc.ForEach, hcl.Expression(nil); got != want {
			// Shouldn't have any count because our test fixture only has
			// for_each in the escaping block.
			t.Errorf("wrong for_each\ngot:  %#v\nwant: %#v", got, want)
		}

		schema := &hcl.BodySchema{
			Attributes: []hcl.AttributeSchema{
				{Name: "normal", Required: true},
				{Name: "count", Required: true},
				{Name: "for_each", Required: true},
			},
			Blocks: []hcl.BlockHeaderSchema{
				{Type: "normal_block"},
				{Type: "lifecycle"},
				{Type: "_"},
			},
		}
		content, diags := rc.Config.Content(schema)
		assertNoDiagnostics(t, diags)

		normalVal, diags := content.Attributes["normal"].Expr.Value(nil)
		assertNoDiagnostics(t, diags)
		if got, want := normalVal, cty.StringVal("yes"); !want.RawEquals(got) {
			t.Errorf("wrong value for 'normal'\ngot:  %#v\nwant: %#v", got, want)
		}

		countVal, diags := content.Attributes["count"].Expr.Value(nil)
		assertNoDiagnostics(t, diags)
		if got, want := countVal, cty.StringVal("not actually count"); !want.RawEquals(got) {
			t.Errorf("wrong value for 'count'\ngot:  %#v\nwant: %#v", got, want)
		}

		var gotBlockTypes []string
		for _, block := range content.Blocks {
			gotBlockTypes = append(gotBlockTypes, block.Type)
		}
		wantBlockTypes := []string{"normal_block", "lifecycle", "_"}
		if diff := cmp.Diff(gotBlockTypes, wantBlockTypes); diff != "" {
			t.Errorf("wrong block types\n%s", diff)
		}
	})
	t.Run("provisioner body", func(t *testing.T) {
		if got, want := len(rc.Managed.Provisioners), 1; got != want {
			t.Fatalf("wrong number of provisioners %d; want %d", got, want)
		}
		pc := rc.Managed.Provisioners[0]

		schema := &hcl.BodySchema{
			Attributes: []hcl.AttributeSchema{
				{Name: "when", Required: true},
				{Name: "normal", Required: true},
			},
			Blocks: []hcl.BlockHeaderSchema{
				{Type: "normal_block"},
				{Type: "lifecycle"},
				{Type: "_"},
			},
		}
		content, diags := pc.Config.Content(schema)
		assertNoDiagnostics(t, diags)

		normalVal, diags := content.Attributes["normal"].Expr.Value(nil)
		assertNoDiagnostics(t, diags)
		if got, want := normalVal, cty.StringVal("yep"); !want.RawEquals(got) {
			t.Errorf("wrong value for 'normal'\ngot:  %#v\nwant: %#v", got, want)
		}
		whenVal, diags := content.Attributes["when"].Expr.Value(nil)
		assertNoDiagnostics(t, diags)
		if got, want := whenVal, cty.StringVal("hell freezes over"); !want.RawEquals(got) {
			t.Errorf("wrong value for 'normal'\ngot:  %#v\nwant: %#v", got, want)
		}
	})
}

func TestEscapingBlockData(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir("testdata/escaping-blocks/data")
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	rc := mod.DataResources["data.foo.bar"]
	if rc == nil {
		t.Fatal("no data resource named data.foo.bar")
	}

	if got := rc.Count; got == nil {
		t.Errorf("count not set; want count = 2")
	} else {
		got, diags := got.Value(nil)
		assertNoDiagnostics(t, diags)
		if want := cty.NumberIntVal(2); !want.RawEquals(got) {
			t.Errorf("wrong count\ngot:  %#v\nwant: %#v", got, want)
		}
	}
	if got, want := rc.ForEach, hcl.Expression(nil); got != want {
		// Shouldn't have any count because our test fixture only has
		// for_each in the escaping block.
		t.Errorf("wrong for_each\ngot:  %#v\nwant: %#v", got, want)
	}

	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "normal", Required: true},
			{Name: "count", Required: true},
			{Name: "for_each", Required: true},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "normal_block"},
			{Type: "lifecycle"},
			{Type: "_"},
		},
	}
	content, diags := rc.Config.Content(schema)
	assertNoDiagnostics(t, diags)

	normalVal, diags := content.Attributes["normal"].Expr.Value(nil)
	assertNoDiagnostics(t, diags)
	if got, want := normalVal, cty.StringVal("yes"); !want.RawEquals(got) {
		t.Errorf("wrong value for 'normal'\ngot:  %#v\nwant: %#v", got, want)
	}

	countVal, diags := content.Attributes["count"].Expr.Value(nil)
	assertNoDiagnostics(t, diags)
	if got, want := countVal, cty.StringVal("not actually count"); !want.RawEquals(got) {
		t.Errorf("wrong value for 'count'\ngot:  %#v\nwant: %#v", got, want)
	}

	var gotBlockTypes []string
	for _, block := range content.Blocks {
		gotBlockTypes = append(gotBlockTypes, block.Type)
	}
	wantBlockTypes := []string{"normal_block", "lifecycle", "_"}
	if diff := cmp.Diff(gotBlockTypes, wantBlockTypes); diff != "" {
		t.Errorf("wrong block types\n%s", diff)
	}

}

func TestEscapingBlockModule(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir("testdata/escaping-blocks/module")
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	mc := mod.ModuleCalls["foo"]
	if mc == nil {
		t.Fatal("no module call named foo")
	}

	if got := mc.Count; got == nil {
		t.Errorf("count not set; want count = 2")
	} else {
		got, diags := got.Value(nil)
		assertNoDiagnostics(t, diags)
		if want := cty.NumberIntVal(2); !want.RawEquals(got) {
			t.Errorf("wrong count\ngot:  %#v\nwant: %#v", got, want)
		}
	}
	if got, want := mc.ForEach, hcl.Expression(nil); got != want {
		// Shouldn't have any count because our test fixture only has
		// for_each in the escaping block.
		t.Errorf("wrong for_each\ngot:  %#v\nwant: %#v", got, want)
	}

	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "normal", Required: true},
			{Name: "count", Required: true},
			{Name: "for_each", Required: true},
		},
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "normal_block"},
			{Type: "lifecycle"},
			{Type: "_"},
		},
	}
	content, diags := mc.Config.Content(schema)
	assertNoDiagnostics(t, diags)

	normalVal, diags := content.Attributes["normal"].Expr.Value(nil)
	assertNoDiagnostics(t, diags)
	if got, want := normalVal, cty.StringVal("yes"); !want.RawEquals(got) {
		t.Errorf("wrong value for 'normal'\ngot:  %#v\nwant: %#v", got, want)
	}

	countVal, diags := content.Attributes["count"].Expr.Value(nil)
	assertNoDiagnostics(t, diags)
	if got, want := countVal, cty.StringVal("not actually count"); !want.RawEquals(got) {
		t.Errorf("wrong value for 'count'\ngot:  %#v\nwant: %#v", got, want)
	}

	var gotBlockTypes []string
	for _, block := range content.Blocks {
		gotBlockTypes = append(gotBlockTypes, block.Type)
	}
	wantBlockTypes := []string{"normal_block", "lifecycle", "_"}
	if diff := cmp.Diff(gotBlockTypes, wantBlockTypes); diff != "" {
		t.Errorf("wrong block types\n%s", diff)
	}

}

func TestEscapingBlockProvider(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDir("testdata/escaping-blocks/provider")
	assertNoDiagnostics(t, diags)
	if mod == nil {
		t.Fatal("got nil root module; want non-nil")
	}

	pc := mod.ProviderConfigs["foo.bar"]
	if pc == nil {
		t.Fatal("no provider configuration named foo.bar")
	}

	if got, want := pc.Alias, "bar"; got != want {
		t.Errorf("wrong alias\ngot:  %#v\nwant: %#v", got, want)
	}

	schema := &hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{
			{Name: "normal", Required: true},
			{Name: "alias", Required: true},
			{Name: "version", Required: true},
		},
	}
	content, diags := pc.Config.Content(schema)
	assertNoDiagnostics(t, diags)

	normalVal, diags := content.Attributes["normal"].Expr.Value(nil)
	assertNoDiagnostics(t, diags)
	if got, want := normalVal, cty.StringVal("yes"); !want.RawEquals(got) {
		t.Errorf("wrong value for 'normal'\ngot:  %#v\nwant: %#v", got, want)
	}
	aliasVal, diags := content.Attributes["alias"].Expr.Value(nil)
	assertNoDiagnostics(t, diags)
	if got, want := aliasVal, cty.StringVal("not actually alias"); !want.RawEquals(got) {
		t.Errorf("wrong value for 'alias'\ngot:  %#v\nwant: %#v", got, want)
	}
	versionVal, diags := content.Attributes["version"].Expr.Value(nil)
	assertNoDiagnostics(t, diags)
	if got, want := versionVal, cty.StringVal("not actually version"); !want.RawEquals(got) {
		t.Errorf("wrong value for 'version'\ngot:  %#v\nwant: %#v", got, want)
	}
}
