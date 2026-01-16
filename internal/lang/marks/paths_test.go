// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package marks

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestPathsWithMark(t *testing.T) {
	input := []cty.PathValueMarks{
		{
			Path:  cty.GetAttrPath("sensitive"),
			Marks: cty.NewValueMarks("sensitive"),
		},
		{
			Path:  cty.GetAttrPath("other"),
			Marks: cty.NewValueMarks("other"),
		},
		{
			Path:  cty.GetAttrPath("both"),
			Marks: cty.NewValueMarks("sensitive", "other"),
		},
		{
			Path:  cty.GetAttrPath("neither"),
			Marks: cty.NewValueMarks("x", "y"),
		},
		{
			Path:  cty.GetAttrPath("deprecated"),
			Marks: cty.NewValueMarks(NewDeprecation("this is deprecated", "")),
		},
		{
			Path:  cty.GetAttrPath("multipleDeprecations"),
			Marks: cty.NewValueMarks(NewDeprecation("this is deprecated", ""), NewDeprecation("this is also deprecated", "")),
		},
		{
			Path:  cty.GetAttrPath("multipleDeprecationsAndSensitive"),
			Marks: cty.NewValueMarks(NewDeprecation("this is deprecated", ""), NewDeprecation("this is also deprecated", ""), "sensitive"),
		},
	}

	gotPaths, gotOthers := PathsWithMark(input, "sensitive")
	wantPaths := []cty.Path{
		cty.GetAttrPath("sensitive"),
		cty.GetAttrPath("both"),
		cty.GetAttrPath("multipleDeprecationsAndSensitive"),
	}
	wantOthers := []cty.PathValueMarks{
		{
			Path:  cty.GetAttrPath("other"),
			Marks: cty.NewValueMarks("other"),
		},
		{
			Path:  cty.GetAttrPath("both"),
			Marks: cty.NewValueMarks("sensitive", "other"),
			// Note that this intentionally preserves the fact that the
			// attribute was both sensitive _and_ had another mark, since
			// that gives the caller the most possible information to
			// potentially handle this combination in a special way in
			// an error message, or whatever. It also conveniently avoids
			// allocating a new mark set, which is nice.
		},
		{
			Path:  cty.GetAttrPath("neither"),
			Marks: cty.NewValueMarks("x", "y"),
		},
		{
			Path:  cty.GetAttrPath("deprecated"),
			Marks: cty.NewValueMarks(NewDeprecation("this is deprecated", "")),
		},
		{
			Path:  cty.GetAttrPath("multipleDeprecations"),
			Marks: cty.NewValueMarks(NewDeprecation("this is deprecated", ""), NewDeprecation("this is also deprecated", "")),
		},
		{
			Path:  cty.GetAttrPath("multipleDeprecationsAndSensitive"),
			Marks: cty.NewValueMarks(NewDeprecation("this is deprecated", ""), NewDeprecation("this is also deprecated", ""), "sensitive"),
		},
	}

	if diff := cmp.Diff(wantPaths, gotPaths, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong matched paths\n%s", diff)
	}
	if diff := cmp.Diff(wantOthers, gotOthers, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong set of entries with other marks\n%s", diff)
	}

	gotPaths, gotOthers = PathsWithMark(input, Deprecation)

	wantPaths = []cty.Path{
		cty.GetAttrPath("deprecated"),
		cty.GetAttrPath("multipleDeprecations"),
		cty.GetAttrPath("multipleDeprecationsAndSensitive"),
	}
	wantOthers = []cty.PathValueMarks{
		{
			Path:  cty.GetAttrPath("sensitive"),
			Marks: cty.NewValueMarks("sensitive"),
		},
		{
			Path:  cty.GetAttrPath("other"),
			Marks: cty.NewValueMarks("other"),
		},
		{
			Path:  cty.GetAttrPath("both"),
			Marks: cty.NewValueMarks("sensitive", "other"),
		},
		{
			Path:  cty.GetAttrPath("neither"),
			Marks: cty.NewValueMarks("x", "y"),
		},
		{
			Path:  cty.GetAttrPath("multipleDeprecationsAndSensitive"),
			Marks: cty.NewValueMarks(NewDeprecation("this is deprecated", ""), NewDeprecation("this is also deprecated", ""), "sensitive"),
		},
	}

	if diff := cmp.Diff(wantPaths, gotPaths, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong matched deprecation paths\n%s", diff)
	}
	if diff := cmp.Diff(wantOthers, gotOthers, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong set of entries with other than deprecation marks\n%s", diff)
	}
}

func TestRemoveAll_valueMarks(t *testing.T) {
	input := []cty.PathValueMarks{
		{
			Path:  cty.GetAttrPath("sensitive"),
			Marks: cty.NewValueMarks("sensitive"),
		},
		{
			Path:  cty.GetAttrPath("other"),
			Marks: cty.NewValueMarks("other"),
		},
		{
			Path:  cty.GetAttrPath("both"),
			Marks: cty.NewValueMarks("sensitive", "other"),
		},
	}

	want := []cty.PathValueMarks{
		{
			Path:  cty.GetAttrPath("sensitive"),
			Marks: cty.NewValueMarks("sensitive"),
		},
		{
			Path:  cty.GetAttrPath("both"),
			Marks: cty.NewValueMarks("sensitive"),
		},
	}

	got := RemoveAll(input, "other")

	if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong matched paths\n%s", diff)
	}
}

func TestRemoveAll_dataMarks(t *testing.T) {
	input := []cty.PathValueMarks{
		{
			Path:  cty.GetAttrPath("deprecated"),
			Marks: cty.NewValueMarks(NewDeprecation("this is deprecated", "")),
		},
		{
			Path:  cty.GetAttrPath("multipleDeprecations"),
			Marks: cty.NewValueMarks(NewDeprecation("this is deprecated", ""), NewDeprecation("this is also deprecated", "")),
		},
		{
			Path:  cty.GetAttrPath("multipleDeprecationsAndSensitive"),
			Marks: cty.NewValueMarks(NewDeprecation("this is deprecated", ""), NewDeprecation("this is also deprecated", ""), "sensitive"),
		},
	}

	want := []cty.PathValueMarks{
		{
			Path:  cty.GetAttrPath("multipleDeprecationsAndSensitive"),
			Marks: cty.NewValueMarks("sensitive"),
		},
	}

	got := RemoveAll(input, Deprecation)

	if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong matched paths\n%s", diff)
	}
}

func TestMarkPaths(t *testing.T) {
	value := cty.ObjectVal(map[string]cty.Value{
		"s": cty.StringVal(".s"),
		"l": cty.ListVal([]cty.Value{
			cty.StringVal(".l[0]"),
			cty.StringVal(".l[1]"),
		}),
		"m": cty.MapVal(map[string]cty.Value{
			"a": cty.StringVal(`.m["a"]`),
			"b": cty.StringVal(`.m["b"]`),
		}),
		"o": cty.ObjectVal(map[string]cty.Value{
			"a": cty.StringVal(".o.a"),
			"b": cty.StringVal(".o.b"),
		}),
		"t": cty.TupleVal([]cty.Value{
			cty.StringVal(`.t[0]`),
			cty.StringVal(`.t[1]`),
		}),
	})
	sensitivePaths := []cty.Path{
		cty.GetAttrPath("s"),
		cty.GetAttrPath("l").IndexInt(1),
		cty.GetAttrPath("m").IndexString("a"),
		cty.GetAttrPath("o").GetAttr("b"),
		cty.GetAttrPath("t").IndexInt(0),
	}
	got := MarkPaths(value, Sensitive, sensitivePaths)
	want := cty.ObjectVal(map[string]cty.Value{
		"s": cty.StringVal(".s").Mark(Sensitive),
		"l": cty.ListVal([]cty.Value{
			cty.StringVal(".l[0]"),
			cty.StringVal(".l[1]").Mark(Sensitive),
		}),
		"m": cty.MapVal(map[string]cty.Value{
			"a": cty.StringVal(`.m["a"]`).Mark(Sensitive),
			"b": cty.StringVal(`.m["b"]`),
		}),
		"o": cty.ObjectVal(map[string]cty.Value{
			"a": cty.StringVal(".o.a"),
			"b": cty.StringVal(".o.b").Mark(Sensitive),
		}),
		"t": cty.TupleVal([]cty.Value{
			cty.StringVal(`.t[0]`).Mark(Sensitive),
			cty.StringVal(`.t[1]`),
		}),
	})
	if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}

	deprecatedPaths := []cty.Path{
		cty.GetAttrPath("s"),
		cty.GetAttrPath("l").IndexInt(1),
		cty.GetAttrPath("m").IndexString("a"),
		cty.GetAttrPath("o").GetAttr("b"),
		cty.GetAttrPath("t").IndexInt(0),
	}
	deprecationMark := NewDeprecation("this is deprecated", "")
	got = MarkPaths(value, deprecationMark, deprecatedPaths)
	want = cty.ObjectVal(map[string]cty.Value{
		"s": cty.StringVal(".s").Mark(deprecationMark),
		"l": cty.ListVal([]cty.Value{
			cty.StringVal(".l[0]"),
			cty.StringVal(".l[1]").Mark(deprecationMark),
		}),
		"m": cty.MapVal(map[string]cty.Value{
			"a": cty.StringVal(`.m["a"]`).Mark(deprecationMark),
			"b": cty.StringVal(`.m["b"]`),
		}),
		"o": cty.ObjectVal(map[string]cty.Value{
			"a": cty.StringVal(".o.a"),
			"b": cty.StringVal(".o.b").Mark(deprecationMark),
		}),
		"t": cty.TupleVal([]cty.Value{
			cty.StringVal(`.t[0]`).Mark(deprecationMark),
			cty.StringVal(`.t[1]`),
		}),
	})
	if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}
}

func TestMarksEqual(t *testing.T) {
	for i, tc := range []struct {
		a, b  []cty.PathValueMarks
		equal bool
	}{
		{
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(Sensitive)},
			},
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(Sensitive)},
			},
			true,
		},
		{
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(Sensitive)},
			},
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "A"}}, Marks: cty.NewValueMarks(Sensitive)},
			},
			false,
		},
		{
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(Sensitive)},
				{Path: cty.Path{cty.GetAttrStep{Name: "b"}}, Marks: cty.NewValueMarks(Sensitive)},
				{Path: cty.Path{cty.GetAttrStep{Name: "c"}}, Marks: cty.NewValueMarks(Sensitive)},
			},
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "b"}}, Marks: cty.NewValueMarks(Sensitive)},
				{Path: cty.Path{cty.GetAttrStep{Name: "c"}}, Marks: cty.NewValueMarks(Sensitive)},
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(Sensitive)},
			},
			true,
		},
		{
			[]cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "a"}, cty.GetAttrStep{Name: "b"}},
					Marks: cty.NewValueMarks(Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "a"}, cty.GetAttrStep{Name: "c"}},
					Marks: cty.NewValueMarks(Sensitive),
				},
			},
			[]cty.PathValueMarks{
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "a"}, cty.GetAttrStep{Name: "c"}},
					Marks: cty.NewValueMarks(Sensitive),
				},
				{
					Path:  cty.Path{cty.GetAttrStep{Name: "a"}, cty.GetAttrStep{Name: "b"}},
					Marks: cty.NewValueMarks(Sensitive),
				},
			},
			true,
		},
		{
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(Sensitive)},
			},
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "b"}}, Marks: cty.NewValueMarks(Sensitive)},
			},
			false,
		},
		{
			nil,
			nil,
			true,
		},
		{
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(Sensitive)},
			},
			nil,
			false,
		},
		{
			nil,
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(Sensitive)},
			},
			false,
		},
		{
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(NewDeprecation("same message", ""))},
			},
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(NewDeprecation("same message", ""))},
			},
			true,
		},
		{
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(NewDeprecation("different", ""))},
			},
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(NewDeprecation("message", ""))},
			},
			false,
		},
		{
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(NewDeprecation("same message", ""))},
			},
			[]cty.PathValueMarks{
				{Path: cty.Path{cty.GetAttrStep{Name: "a"}}, Marks: cty.NewValueMarks(NewDeprecation("same message", ""))},
			},
			true,
		},
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			if MarksEqual(tc.a, tc.b) != tc.equal {
				t.Fatalf("MarksEqual(\n%#v,\n%#v,\n) != %t\n", tc.a, tc.b, tc.equal)
			}
		})
	}
}
