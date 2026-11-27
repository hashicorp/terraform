// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configschema

import (
	"errors"
	"fmt"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestTypeAtPathHCLTraversal(t *testing.T) {
	stringMap := cty.Map(cty.String)
	stringList := cty.List(cty.String)
	object := cty.Object(map[string]cty.Type{"id": cty.String, "other": cty.DynamicPseudoType})
	for name, tc := range map[string]struct {
		typ  cty.Type
		path cty.Path
		want cty.Type
	}{
		"map dot":                  {stringMap, cty.GetAttrPath("id"), cty.String},
		"object brackets":          {object, cty.IndexPath(cty.StringVal("id")), cty.String},
		"dynamic selected value":   {object, cty.GetAttrPath("other"), cty.DynamicPseudoType},
		"dynamic descendant":       {object, cty.GetAttrPath("other").GetAttr("id"), cty.NilType},
		"nested collection member": {cty.Map(cty.List(object)), cty.GetAttrPath("key").IndexInt(0).IndexString("id"), cty.String},
		"marked list index":        {stringList, cty.IndexPath(cty.NumberIntVal(0).Mark("sensitive")), cty.String},
		"marked tuple index":       {cty.Tuple([]cty.Type{cty.String, cty.Number}), cty.IndexPath(cty.NumberIntVal(1).Mark("sensitive")), cty.Number},
		"marked fractional index":  {stringList, cty.IndexPath(cty.NumberFloatVal(0.5).Mark("sensitive")), cty.NilType},
		"marked unknown index":     {stringList, cty.IndexPath(cty.UnknownVal(cty.Number).Mark("sensitive")), cty.NilType},
		"numeric string list key":  {stringList, cty.IndexPath(cty.StringVal("0")), cty.NilType},
		"numeric map key":          {stringMap, cty.IndexPath(cty.NumberIntVal(0)), cty.NilType},
		"nested set":               {cty.Object(map[string]cty.Type{"ids": cty.Set(cty.String)}), cty.GetAttrPath("ids").IndexString("id"), cty.NilType},
	} {
		t.Run(name, func(t *testing.T) {
			typ, ok := TypeAtPath(tc.typ, tc.path)
			if ok != (tc.want != cty.NilType) || !typ.Equals(tc.want) {
				t.Fatalf("got (%#v, %t), want type %#v", typ, ok, tc.want)
			}
		})
	}
}

func TestBlockWalkPath(t *testing.T) {
	id := &Attribute{Type: cty.String, Optional: true, Computed: true, Sensitive: true}
	dynamic := &Attribute{Type: cty.DynamicPseudoType, Optional: true}
	fields := map[string]*Attribute{"id": id, "dynamic": dynamic}
	fieldsType := cty.Object(map[string]cty.Type{"id": cty.String, "dynamic": cty.DynamicPseudoType})
	rawType := cty.Object(map[string]cty.Type{"items": cty.Map(cty.List(cty.String))})
	raw := &Attribute{Type: rawType, Optional: true}
	schema := &Block{
		Attributes: map[string]*Attribute{"raw": raw},
		BlockTypes: map[string]*NestedBlock{},
	}
	for name, mode := range map[string]NestingMode{
		"single": NestingSingle, "group": NestingGroup, "list": NestingList, "map": NestingMap, "set": NestingSet,
	} {
		schema.BlockTypes[name] = &NestedBlock{Nesting: mode, Block: Block{Attributes: fields}}
	}
	for name, mode := range map[string]NestingMode{
		"single_attr": NestingSingle, "list_attr": NestingList, "map_attr": NestingMap,
	} {
		schema.Attributes[name] = &Attribute{Optional: true, NestedType: &Object{Nesting: mode, Attributes: fields}}
	}
	// Exercise multiple structured attribute ancestors, not just a collection's leaf.
	schema.Attributes["deep"] = &Attribute{Computed: true, NestedType: &Object{
		Nesting: NestingSingle, Attributes: map[string]*Attribute{"child": schema.Attributes["list_attr"]},
	}}

	type visit struct {
		prefix int
		attr   *Attribute
		block  *NestedBlock
	}
	for name, tc := range map[string]struct {
		path     cty.Path
		visits   []visit
		wantType cty.Type
	}{
		"raw whole":               {cty.GetAttrPath("raw"), []visit{{1, raw, nil}}, rawType},
		"raw suffix":              {cty.GetAttrPath("raw").IndexString("items").IndexString("literal.key").IndexInt(2), []visit{{1, raw, nil}}, cty.String},
		"single child brackets":   {cty.IndexStringPath("single").IndexString("id"), []visit{{1, nil, schema.BlockTypes["single"]}, {2, id, nil}}, cty.String},
		"list block":              {cty.GetAttrPath("list"), []visit{{1, nil, schema.BlockTypes["list"]}}, cty.List(fieldsType)},
		"marked list index":       {cty.GetAttrPath("list").Index(cty.NumberIntVal(3).Mark("sensitive")).GetAttr("id"), []visit{{1, nil, schema.BlockTypes["list"]}, {3, id, nil}}, cty.String},
		"marked map key":          {cty.GetAttrPath("map").Index(cty.StringVal("key").Mark("sensitive")).GetAttr("id"), []visit{{1, nil, schema.BlockTypes["map"]}, {3, id, nil}}, cty.String},
		"map block":               {cty.GetAttrPath("map"), []visit{{1, nil, schema.BlockTypes["map"]}}, cty.Map(fieldsType)},
		"map child literal key":   {cty.GetAttrPath("map").IndexString("literal.key").GetAttr("id"), []visit{{1, nil, schema.BlockTypes["map"]}, {3, id, nil}}, cty.String},
		"map dot key":             {cty.GetAttrPath("map").GetAttr("key").GetAttr("dynamic"), []visit{{1, nil, schema.BlockTypes["map"]}, {3, dynamic, nil}}, cty.DynamicPseudoType},
		"set whole":               {cty.GetAttrPath("set"), []visit{{1, nil, schema.BlockTypes["set"]}}, cty.Set(fieldsType)},
		"single attr child":       {cty.GetAttrPath("single_attr").GetAttr("id"), []visit{{1, schema.Attributes["single_attr"], nil}, {2, id, nil}}, cty.String},
		"list attr child":         {cty.GetAttrPath("list_attr").IndexInt(2).IndexString("id"), []visit{{1, schema.Attributes["list_attr"], nil}, {3, id, nil}}, cty.String},
		"map attr child":          {cty.GetAttrPath("map_attr").IndexString("literal.key").GetAttr("id"), []visit{{1, schema.Attributes["map_attr"], nil}, {3, id, nil}}, cty.String},
		"deep attr":               {cty.GetAttrPath("deep").GetAttr("child").IndexInt(1).GetAttr("id"), []visit{{1, schema.Attributes["deep"], nil}, {2, schema.Attributes["list_attr"], nil}, {4, id, nil}}, cty.String},
		"empty":                   {},
		"missing":                 {path: cty.GetAttrPath("missing")},
		"numeric root":            {path: cty.IndexIntPath(0)},
		"missing child":           {path: cty.GetAttrPath("single").GetAttr("missing")},
		"list without index":      {path: cty.GetAttrPath("list").GetAttr("id")},
		"list string index":       {path: cty.GetAttrPath("list").IndexString("key").GetAttr("id")},
		"marked unknown index":    {path: cty.GetAttrPath("list_attr").Index(cty.UnknownVal(cty.Number).Mark("sensitive")).GetAttr("id")},
		"null key":                {path: cty.GetAttrPath("map").Index(cty.NullVal(cty.String)).GetAttr("id")},
		"map numeric index":       {path: cty.GetAttrPath("map").IndexInt(0).GetAttr("id")},
		"set traversal":           {path: cty.GetAttrPath("set").IndexInt(0).GetAttr("id")},
		"nested attr wrong index": {path: cty.GetAttrPath("list_attr").IndexString("key").GetAttr("id")},
		"raw missing field":       {path: cty.GetAttrPath("raw").GetAttr("missing")},
	} {
		t.Run(name, func(t *testing.T) {
			original := tc.path.Copy()
			var got []visit
			typ, err := schema.WalkPath(tc.path, func(prefix cty.Path, attr *Attribute, block *NestedBlock) error {
				if (attr == nil) == (block == nil) {
					t.Error("callback must receive exactly one schema pointer")
				}
				if !prefix.Equals(original[:len(prefix)]) {
					t.Errorf("callback changed path representation: %#v", prefix)
				}
				got = append(got, visit{len(prefix), attr, block})
				return nil
			})
			if tc.visits == nil {
				if err == nil || typ != cty.NilType {
					t.Fatalf("got (%#v, %v), want (cty.NilType, error)", typ, err)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if !typ.Equals(tc.wantType) {
					t.Errorf("got type %#v, want %#v", typ, tc.wantType)
				}
				if len(got) != len(tc.visits) {
					t.Fatalf("got %d callbacks, want %d", len(got), len(tc.visits))
				}
				for i, want := range tc.visits {
					if got[i] != want {
						t.Errorf("callback %d: got %#v, want %#v", i, got[i], want)
					}
				}
			}
			if !tc.path.Equals(original) {
				t.Error("WalkPath mutated its input path")
			}
			t.Run("nil visitor", func(t *testing.T) {
				nilType, nilErr := schema.WalkPath(tc.path, nil)
				if !nilType.Equals(typ) || (nilErr == nil) != (err == nil) {
					t.Errorf("nil visitor got (%#v, %v), want type %#v and same success", nilType, nilErr, typ)
				}
			})
		})
	}
	if !id.Optional || !id.Computed || !id.Sensitive || !schema.Attributes["deep"].Computed {
		t.Error("WalkPath mutated schema flags")
	}
}

func TestBlockWalkPathCallbackError(t *testing.T) {
	schema := &Block{BlockTypes: map[string]*NestedBlock{"child": {
		Nesting: NestingSingle, Block: Block{Attributes: map[string]*Attribute{"id": {Type: cty.String}}},
	}}}
	sentinel := errors.New("stop walking")
	wantErr := fmt.Errorf("visitor: %w", sentinel)
	for _, stopAt := range []int{1, 2} {
		t.Run(fmt.Sprintf("callback %d", stopAt), func(t *testing.T) {
			calls := 0
			typ, err := schema.WalkPath(cty.GetAttrPath("child").GetAttr("id"), func(cty.Path, *Attribute, *NestedBlock) error {
				calls++
				if calls == stopAt {
					return wantErr
				}
				return nil
			})
			if typ != cty.NilType || err != wantErr || !errors.Is(err, sentinel) {
				t.Errorf("got (%#v, %v), want (cty.NilType, original error)", typ, err)
			}
			if calls != stopAt {
				t.Errorf("got %d callbacks, want %d", calls, stopAt)
			}
		})
	}
}

func TestBlockWalkPathInvalidIndex(t *testing.T) {
	schema := &Block{Attributes: map[string]*Attribute{"ids": {Type: cty.List(cty.String), Optional: true}}}
	for name, key := range map[string]cty.Value{
		"negative": cty.NumberIntVal(-1), "fractional": cty.NumberFloatVal(0.5),
		"overflow": cty.MustParseNumberVal("9223372036854775808"),
		"unknown":  cty.UnknownVal(cty.Number), "null": cty.NullVal(cty.Number),
	} {
		t.Run(name, func(t *testing.T) {
			path := cty.GetAttrPath("ids").Index(key)
			if typ, err := schema.WalkPath(path, nil); typ != cty.NilType || err == nil {
				t.Fatalf("got (%#v, %v), want invalid-index rejection", typ, err)
			}
			if _, ok := schema.TypeAtPath(path); ok {
				t.Fatal("type-only lookup accepted the invalid index")
			}
		})
	}
}

func TestBlockWalkPathNilSchema(t *testing.T) {
	var schema *Block
	path := cty.GetAttrPath("id")
	if typ, err := schema.WalkPath(path, nil); typ != cty.NilType || err == nil {
		t.Fatalf("got (%#v, %v), want nil-schema rejection", typ, err)
	}
	if _, ok := schema.TypeAtPath(path); ok {
		t.Fatal("type-only lookup accepted a nil schema")
	}
}

func TestAttributeByPath(t *testing.T) {
	schema := &Block{
		Attributes: map[string]*Attribute{
			"a1": {Description: "a1"},
			"a2": {Description: "a2"},
			"a3": {
				Description: "a3",
				NestedType: &Object{
					Nesting: NestingList,
					Attributes: map[string]*Attribute{
						"nt1": {Description: "nt1"},
						"nt2": {
							Description: "nt2",
							NestedType: &Object{
								Nesting: NestingSingle,
								Attributes: map[string]*Attribute{
									"deeply_nested": {Description: "deeply_nested"},
								},
							},
						},
					},
				},
			},
			"a4": {
				Description: "a3",
				NestedType: &Object{
					Nesting: NestingMap,
					Attributes: map[string]*Attribute{
						"nt1": {Description: "nt1"},
						"nt2": {
							Description: "nt2",
							NestedType: &Object{
								Nesting: NestingSingle,
								Attributes: map[string]*Attribute{
									"deeply_nested": {Description: "deeply_nested"},
								},
							},
						},
					},
				},
			},
		},
		BlockTypes: map[string]*NestedBlock{
			"b1": {
				Nesting: NestingList,
				Block: Block{
					Attributes: map[string]*Attribute{
						"a3": {Description: "a3"},
						"a4": {Description: "a4"},
					},
					BlockTypes: map[string]*NestedBlock{
						"b2": {
							Nesting: NestingMap,
							Block: Block{
								Attributes: map[string]*Attribute{
									"a5": {Description: "a5"},
									"a6": {Description: "a6"},
								},
							},
						},
					},
				},
			},
			"b3": {
				Nesting: NestingMap,
				Block: Block{
					Attributes: map[string]*Attribute{
						"a7": {Description: "a7"},
						"a8": {Description: "a8"},
					},
					BlockTypes: map[string]*NestedBlock{
						"b4": {
							Nesting: NestingSet,
							Block: Block{
								Attributes: map[string]*Attribute{
									"a9":  {Description: "a9"},
									"a10": {Description: "a10"},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range []struct {
		path            cty.Path
		attrDescription string
		exists          bool
	}{
		{
			cty.GetAttrPath("a2"),
			"a2",
			true,
		},
		{
			cty.GetAttrPath("a3").IndexInt(1).GetAttr("nt2"),
			"nt2",
			true,
		},
		{
			cty.GetAttrPath("a3").IndexInt(1).GetAttr("b2").IndexString("foo").GetAttr("no"),
			"missing",
			false,
		},
		{
			cty.GetAttrPath("a3").IndexInt(1).GetAttr("nt2").GetAttr("deeply_nested"),
			"deeply_nested",
			true,
		},
		{
			cty.GetAttrPath("b1"),
			"block",
			false,
		},
		{
			cty.GetAttrPath("b1").IndexInt(1).GetAttr("a3"),
			"a3",
			true,
		},
		{
			cty.GetAttrPath("b1").IndexInt(1).GetAttr("b2").IndexString("foo").GetAttr("a7"),
			"missing",
			false,
		},
		{
			cty.GetAttrPath("b1").IndexInt(1).GetAttr("b2").IndexString("foo").GetAttr("a6"),
			"a6",
			true,
		},
		{
			cty.GetAttrPath("b3").IndexString("foo").GetAttr("b2").IndexString("foo").GetAttr("a7"),
			"missing_block",
			false,
		},
		{
			cty.GetAttrPath("b3").IndexString("foo").GetAttr("a7"),
			"a7",
			true,
		},
		{
			// Index steps don't apply to the schema, so the set Index value doesn't matter.
			cty.GetAttrPath("b3").IndexString("foo").GetAttr("b4").Index(cty.EmptyObjectVal).GetAttr("a9"),
			"a9",
			true,
		},
		{
			cty.GetAttrPath("a4").IndexString("test").GetAttr("nt2").GetAttr("deeply_nested"),
			"deeply_nested",
			true,
		},
	} {
		t.Run(tc.attrDescription, func(t *testing.T) {
			attr := schema.AttributeByPath(tc.path)
			if !tc.exists && attr == nil {
				return
			}

			if attr == nil {
				t.Fatalf("missing attribute from path %#v\n", tc.path)
			}

			if attr.Description != tc.attrDescription {
				t.Fatalf("expected Attribute for %q, got %#v\n", tc.attrDescription, attr)
			}
		})
	}
}

func TestObject_AttributeByPath(t *testing.T) {
	obj := &Object{
		Nesting: NestingList,
		Attributes: map[string]*Attribute{
			"a1": {Description: "a1"},
			"a2": {
				Description: "a2",
				NestedType: &Object{
					Nesting: NestingSingle,
					Attributes: map[string]*Attribute{
						"n1": {Description: "n1"},
						"n2": {
							Description: "n2",
							NestedType: &Object{
								Attributes: map[string]*Attribute{
									"dn1": {Description: "dn1"},
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		path            cty.Path
		attrDescription string
		exists          bool
	}{
		{
			cty.GetAttrPath("a2"),
			"a2",
			true,
		},
		{
			cty.GetAttrPath("a3"),
			"missing",
			false,
		},
		{
			cty.GetAttrPath("a2").IndexString("foo").GetAttr("n1"),
			"n1",
			true,
		},
		{
			cty.GetAttrPath("a2").IndexString("foo").GetAttr("n2").IndexInt(11).GetAttr("dn1"),
			"dn1",
			true,
		},
		{
			cty.GetAttrPath("a2").IndexString("foo").GetAttr("n2").IndexInt(11).GetAttr("dn1").IndexString("hello").GetAttr("nope"),
			"missing_nested",
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.attrDescription, func(t *testing.T) {
			attr := obj.AttributeByPath(tc.path)
			if !tc.exists && attr == nil {
				return
			}

			if !tc.exists && attr != nil {
				t.Fatalf("found Attribute, expected nil from path %#v\n", tc.path)
			}

			if attr == nil {
				t.Fatalf("missing attribute from path %#v\n", tc.path)
			}

			if attr.Description != tc.attrDescription {
				t.Fatalf("expected Attribute for %q, got %#v\n", tc.attrDescription, attr)
			}
		})
	}
}

func TestBlockByPath(t *testing.T) {
	schema := &Block{
		BlockTypes: map[string]*NestedBlock{
			"b1": {
				Nesting: NestingList,
				Block: Block{
					Attributes: map[string]*Attribute{
						"a3": {Description: "a3"},
						"a4": {Description: "a4"},
					},
					BlockTypes: map[string]*NestedBlock{
						"b2": {
							Nesting: NestingMap,
							Block: Block{
								Attributes: map[string]*Attribute{
									"a5": {Description: "a5"},
									"a6": {Description: "a6"},
								},
							},
						},
					},
				},
			},
			"b3": {
				Nesting: NestingMap,
				Block: Block{
					Attributes: map[string]*Attribute{
						"a7": {Description: "a7"},
						"a8": {Description: "a8"},
					},
					BlockTypes: map[string]*NestedBlock{
						"b4": {
							Nesting: NestingSet,
							Block: Block{
								Attributes: map[string]*Attribute{
									"a9":  {Description: "a9"},
									"a10": {Description: "a10"},
								},
							},
						},
					},
				},
			},
		},
	}

	for i, tc := range []struct {
		path   cty.Path
		exists bool
	}{
		{
			cty.GetAttrPath("b1").IndexInt(1).GetAttr("b2"),
			true,
		},
		{
			cty.GetAttrPath("b1"),
			true,
		},
		{
			cty.GetAttrPath("b2"),
			false,
		},
		{
			cty.GetAttrPath("b3").IndexString("foo").GetAttr("b2"),
			false,
		},
		{
			cty.GetAttrPath("b3").IndexString("foo").GetAttr("b4"),
			true,
		},
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			block := schema.NestedBlockByPath(tc.path)
			if !tc.exists && block == nil {
				return
			}

			if block == nil {
				t.Fatalf("missing block from path %#v\n", tc.path)
			}
		})
	}
}
