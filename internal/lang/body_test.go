// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/format"
	"github.com/zclconf/go-cty/cty"
)

func TestVisitAttributes(t *testing.T) {
	leaf := configschema.Block{Attributes: map[string]*configschema.Attribute{
		"id": {Type: cty.String, Required: true},
	}}
	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{"settings": {Type: cty.Map(cty.String), Optional: true}},
		BlockTypes: map[string]*configschema.NestedBlock{
			"single": {Nesting: configschema.NestingSingle, Block: leaf},
			"group":  {Nesting: configschema.NestingGroup, Block: leaf},
			"list":   {Nesting: configschema.NestingList, Block: leaf},
			"map":    {Nesting: configschema.NestingMap, Block: leaf},
			"set":    {Nesting: configschema.NestingSet, Block: leaf},
			"other":  {Nesting: configschema.NestingList, Block: leaf},
			"outer": {Nesting: configschema.NestingList, Block: configschema.Block{BlockTypes: map[string]*configschema.NestedBlock{
				"list": {Nesting: configschema.NestingList, Block: leaf},
			}}},
		},
	}
	for name, tc := range map[string]struct {
		src  string
		want map[string]string
	}{
		"static paths": {
			src: `
settings = { id = "root" }
single { id = "single" }
group { id = "group" }
list { id = "first" }
other { id = "other-first" }
map "a.b" { id = "map" }
list { id = "second" }
other { id = "other-second" }
outer {
  list { id = "nested-first" }
  list { id = "nested-second" }
}
outer {
  list { id = "nested-reset" }
}
set { id = "unstable" }
`,
			want: map[string]string{
				".settings": `{ id = "root" }`, ".single.id": `"single"`, ".group.id": `"group"`,
				".list[0].id": `"first"`, `.map["a.b"].id`: `"map"`, ".list[1].id": `"second"`,
				".other[0].id": `"other-first"`, ".other[1].id": `"other-second"`,
				".outer[0].list[0].id": `"nested-first"`, ".outer[0].list[1].id": `"nested-second"`,
				".outer[1].list[0].id": `"nested-reset"`,
			},
		},
		"dynamic after static siblings": {
			src: `
list { id = "static" }
dynamic "list" {
  for_each = []
  content { id = "dynamic" }
}
single { id = "stable" }
`,
			want: map[string]string{".single.id": `"stable"`},
		},
		"dynamic before static siblings": {
			src: `
dynamic "list" {
  for_each = []
  content { id = "dynamic" }
}
list { id = "static" }
`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(tc.src), "test.tf", hcl.InitialPos)
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}
			got := make(map[string]string)
			var retained []cty.Path
			diags = VisitAttributes(file.Body, schema, func(path cty.Path, attr *hcl.Attribute) {
				if attr == nil || len(path) == 0 {
					t.Fatal("visitor must receive an attribute and a stable path")
				}
				got[format.CtyPath(path)] = string(attr.Expr.Range().SliceBytes([]byte(tc.src)))
				retained = append(retained, path)
			})
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got paths %v, want %v", got, tc.want)
			}
			for path, expr := range tc.want {
				if got[path] != expr {
					t.Errorf("%s: got expression %q, want %q", path, got[path], expr)
				}
			}
			// Retained callback paths must not share backing arrays with siblings.
			if len(retained) > 1 {
				before := format.CtyPath(retained[1])
				retained[0][0] = cty.GetAttrStep{Name: "changed"}
				if format.CtyPath(retained[1]) != before {
					t.Error("mutating a callback path changed a sibling's path")
				}
			}
		})
	}
}

func TestVisitAttributesBlockToAttribute(t *testing.T) {
	file, diags := hclsyntax.ParseConfig([]byte(`
items { id = "first" }
items { id = "second" }
`), "test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}
	schema := &configschema.Block{Attributes: map[string]*configschema.Attribute{
		"items": {Type: cty.List(cty.Object(map[string]cty.Type{"id": cty.String})), Optional: true},
	}}
	calls := 0
	diags = VisitAttributes(file.Body, schema, func(path cty.Path, attr *hcl.Attribute) {
		calls++
		if !path.Equals(cty.GetAttrPath("items")) {
			t.Fatalf("got path %#v, want the normalized attribute", path)
		}
		value, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			t.Fatal(diags.Error())
		}
		want := cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("first")}),
			cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("second")}),
		})
		if !value.RawEquals(want) {
			t.Errorf("got %#v, want %#v", value, want)
		}
	})
	if diags.HasErrors() || calls != 1 {
		t.Fatalf("got %d calls and %s, want one attribute", calls, diags.Error())
	}
}
