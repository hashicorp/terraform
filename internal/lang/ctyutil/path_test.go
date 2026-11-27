// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ctyutil

import (
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestNormalizePathIgnoresKeyMarks(t *testing.T) {
	for _, key := range []cty.Value{cty.StringVal("id"), cty.NumberIntVal(0)} {
		marked := cty.GetAttrPath("settings").Index(key.Mark("selection"))
		unmarked := cty.GetAttrPath("settings").Index(key)
		if !NormalizePath(marked).Equals(NormalizePath(unmarked)) {
			t.Error("key marks changed path identity")
		}
		if !marked[1].(cty.IndexStep).Key.HasMark("selection") {
			t.Error("normalization mutated the input path")
		}
	}
}

func TestApplyPathSingleStep(t *testing.T) {
	object := cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("value")})
	mapping := cty.MapVal(map[string]cty.Value{"id": cty.StringVal("value")})
	for name, tc := range map[string]struct {
		value cty.Value
		step  cty.PathStep
		want  cty.Value
	}{
		"object attribute": {object, cty.GetAttrStep{Name: "id"}, cty.StringVal("value")},
		"object index":     {object, cty.IndexStep{Key: cty.StringVal("id")}, cty.StringVal("value")},
		"map attribute":    {mapping, cty.GetAttrStep{Name: "id"}, cty.StringVal("value")},
		"map index":        {mapping, cty.IndexStep{Key: cty.StringVal("id")}, cty.StringVal("value")},
		"dotted key": {
			cty.MapVal(map[string]cty.Value{"a.b": cty.StringVal("literal")}),
			cty.GetAttrStep{Name: "a.b"}, cty.StringVal("literal"),
		},
		"unknown object":  {cty.UnknownVal(object.Type()), cty.IndexStep{Key: cty.StringVal("id")}, cty.UnknownVal(cty.String)},
		"unknown map":     {cty.UnknownVal(mapping.Type()), cty.GetAttrStep{Name: "id"}, cty.UnknownVal(cty.String)},
		"unknown map key": {mapping, cty.IndexStep{Key: cty.UnknownVal(cty.String)}, cty.UnknownVal(cty.String)},
		"list":            {cty.ListVal([]cty.Value{cty.StringVal("value")}), cty.IndexStep{Key: cty.NumberIntVal(0)}, cty.StringVal("value")},
		"tuple":           {cty.TupleVal([]cty.Value{cty.StringVal("value")}), cty.IndexStep{Key: cty.NumberIntVal(0)}, cty.StringVal("value")},
		// Schema callers may reject set elements, but that is not this helper's decision.
		"set":           {cty.SetVal([]cty.Value{cty.StringVal("value")}), cty.IndexStep{Key: cty.StringVal("value")}, cty.StringVal("value")},
		"marked object": {object.Mark("sensitive"), cty.IndexStep{Key: cty.StringVal("id")}, cty.StringVal("value").Mark("sensitive")},
		"marked map":    {mapping.Mark("sensitive"), cty.GetAttrStep{Name: "id"}, cty.StringVal("value").Mark("sensitive")},
		"marked unknown object": {
			cty.UnknownVal(object.Type()).Mark("sensitive"), cty.IndexStep{Key: cty.StringVal("id")}, cty.UnknownVal(cty.String).Mark("sensitive"),
		},
		"missing attribute":  {value: object, step: cty.GetAttrStep{Name: "missing"}},
		"missing map key":    {value: mapping, step: cty.GetAttrStep{Name: "missing"}},
		"unknown object key": {value: object, step: cty.IndexStep{Key: cty.UnknownVal(cty.String)}},
		"null object key":    {value: object, step: cty.IndexStep{Key: cty.NullVal(cty.String)}},
		"null value":         {value: cty.NullVal(object.Type()), step: cty.GetAttrStep{Name: "id"}},
		"nil value":          {value: cty.NilVal, step: cty.GetAttrStep{Name: "id"}},
		"scalar value":       {value: cty.StringVal("value"), step: cty.GetAttrStep{Name: "id"}},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := ApplyPath(tc.value, cty.Path{tc.step})
			if tc.want == cty.NilVal {
				if err == nil || got != cty.NilVal {
					t.Fatalf("got (%#v, %v), want an error and cty.NilVal", got, err)
				}
				return
			}
			if err != nil || !got.RawEquals(tc.want) {
				t.Fatalf("got (%#v, %v), want %#v", got, err, tc.want)
			}
		})
	}
}

func TestApplyPathMarkedUnknownIndex(t *testing.T) {
	for name, tc := range map[string]struct {
		value cty.Value
		step  cty.PathStep
		index cty.PathStep
	}{
		"map": {
			cty.UnknownVal(cty.Map(cty.String)).Mark("sensitive"),
			cty.GetAttrStep{Name: "id"}, cty.IndexStep{Key: cty.StringVal("id")},
		},
		"list": {
			cty.UnknownVal(cty.List(cty.String)).Mark("sensitive"),
			cty.IndexStep{Key: cty.NumberIntVal(0)}, cty.IndexStep{Key: cty.NumberIntVal(0)},
		},
	} {
		t.Run(name, func(t *testing.T) {
			// cty currently drops marks when index existence is unknown. We
			// delegate that behavior rather than introduce new mark semantics.
			want, wantErr := tc.index.Apply(tc.value)
			got, err := ApplyPath(tc.value, cty.Path{tc.step})
			if err != nil || wantErr != nil || !got.RawEquals(want) {
				t.Fatalf("got (%#v, %v), want cty's result (%#v, %v)", got, err, want, wantErr)
			}
		})
	}
}

func TestApplyPath(t *testing.T) {
	value := cty.ObjectVal(map[string]cty.Value{
		"items": cty.MapVal(map[string]cty.Value{
			"key": cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("value").Mark("leaf")}),
		}),
	}).Mark("root")
	path := cty.IndexStringPath("items").GetAttr("key").IndexString("id")
	originalPath := path.Copy()
	got, err := ApplyPath(value, path)
	want := cty.StringVal("value").WithMarks(cty.NewValueMarks("root", "leaf"))
	if err != nil || !got.RawEquals(want) {
		t.Fatalf("got (%#v, %v), want %#v", got, err, want)
	}
	if !path.Equals(originalPath) {
		t.Error("applying a path must not modify it")
	}
	if !value.HasMark("root") || !value.GetAttr("items").Index(cty.StringVal("key")).GetAttr("id").HasMark("leaf") {
		t.Error("applying a path must not remove marks from the input")
	}

	t.Run("empty path", func(t *testing.T) {
		got, err := ApplyPath(value, nil)
		if err != nil || !got.RawEquals(value) {
			t.Fatalf("got (%#v, %v), want unchanged value", got, err)
		}
	})
	t.Run("error identifies step", func(t *testing.T) {
		got, err := ApplyPath(value, cty.GetAttrPath("items").GetAttr("missing"))
		if got != cty.NilVal || err == nil || !strings.HasPrefix(err.Error(), "at step 1: ") {
			t.Fatalf("got (%#v, %v), want an error at step 1 and cty.NilVal", got, err)
		}
	})
}
