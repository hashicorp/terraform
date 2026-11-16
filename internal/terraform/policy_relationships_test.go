// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

func TestPolicyStateValue(t *testing.T) {
	schema := providers.Schema{Body: &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":     {Type: cty.String, Computed: true},
			"secret": {Type: cty.String, Optional: true, Sensitive: true},
			"other":  {Type: cty.String, Optional: true},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"nested": {
				Nesting: configschema.NestingList,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"key": {Type: cty.String, Optional: true, Sensitive: true},
					},
				},
			},
		},
	}}
	obj := &states.ResourceInstanceObjectSrc{
		AttrsJSON: []byte(`{"id":"i","secret":"s","other":"o","nested":[{"key":"k"}]}`),
		// A path that was marked sensitive in configuration.
		AttrSensitivePaths: []cty.Path{cty.GetAttrPath("other")},
		Status:             states.ObjectReady,
	}

	// The policy client redacts the paths the schema declares sensitive.
	val, err := policyStateValue(obj, schema)
	if err != nil {
		t.Fatal(err)
	}
	_, pvms := val.UnmarkDeepWithPaths()
	paths, _ := marks.PathsWithMark(pvms, marks.Sensitive)
	if len(paths) != 1 || !paths[0].Equals(cty.GetAttrPath("other")) {
		t.Fatalf("wrong sensitive paths: %#v", paths)
	}

	t.Run("schema version mismatch", func(t *testing.T) {
		old := *obj
		old.SchemaVersion = 1
		if _, err := policyStateValue(&old, schema); err == nil {
			t.Fatal("expected an error for a state object with another schema version")
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		bad := *obj
		bad.AttrsJSON = []byte(`{"unknown_attr":true}`)
		if _, err := policyStateValue(&bad, schema); err == nil {
			t.Fatal("expected an error for a state object that doesn't conform to the schema")
		}
	})
}

// A key holds between two reads as contract §3.5 says, and certainly fails
// only if one side is empty or both are known.
func TestPolicyReadHolds(t *testing.T) {
	str := cty.StringVal
	unknown := cty.UnknownVal(cty.String)
	list := func(vs ...cty.Value) cty.Value { return cty.ListVal(vs) }
	obj := func(v cty.Value) *policyObject {
		return &policyObject{addr: "test_vm.v", op: proto.Operation_CREATE, value: cty.ObjectVal(map[string]cty.Value{"k": v})}
	}
	withOrigin := func(v cty.Value, origin string) *policyObject {
		o := obj(v)
		o.origins = map[string][]string{"k": {origin}}
		return o
	}
	const (
		holds = iota
		fails
		undecided
	)
	tests := map[string]struct {
		a, b *policyObject
		want int
	}{
		"equal scalars":                 {obj(str("a")), obj(str("a")), holds},
		"different scalars":             {obj(str("a")), obj(str("b")), fails},
		"different types":               {obj(str("1")), obj(cty.NumberIntVal(1)), fails},
		"null":                          {obj(cty.NullVal(cty.String)), obj(cty.NullVal(cty.String)), fails},
		"empty string":                  {obj(str("")), obj(str("")), fails},
		"empty list":                    {obj(cty.ListValEmpty(cty.String)), obj(cty.ListValEmpty(cty.String)), fails},
		"unknown scalar":                {obj(unknown), obj(str("a")), undecided},
		"unknown scalar, empty other":   {obj(unknown), obj(cty.NullVal(cty.String)), fails},
		"scalar in list":                {obj(str("b")), obj(list(str("a"), str("b"))), holds},
		"scalar not in list":            {obj(str("c")), obj(list(str("a"), str("b"))), fails},
		"scalar maybe in list":          {obj(str("c")), obj(list(str("a"), unknown)), undecided},
		"equal lists":                   {obj(list(str("a"), str("b"))), obj(list(str("a"), str("b"))), holds},
		"lists in another order":        {obj(list(str("a"), str("b"))), obj(list(str("b"), str("a"))), fails},
		"lists of different lengths":    {obj(list(str("a"))), obj(list(str("a"), str("b"))), fails},
		"list with an unknown element":  {obj(list(str("a"), unknown)), obj(list(str("a"), str("b"))), undecided},
		"list with an empty element":    {obj(list(str("a"), str(""))), obj(list(str("a"))), holds},
		"wholly unknown list":           {obj(cty.UnknownVal(cty.List(cty.String))), obj(list(str("a"))), undecided},
		"origins of a delete are unset": {&policyObject{addr: "test_vm.v", op: proto.Operation_DELETE, value: cty.ObjectVal(map[string]cty.Value{"k": unknown}), origins: map[string][]string{"k": {"x"}}}, withOrigin(unknown, "x"), undecided},
		"shared origins":                {withOrigin(unknown, "x"), withOrigin(unknown, "x"), holds},
		"shared origin, known values":   {withOrigin(str("a"), "x"), withOrigin(str("b"), "x"), fails},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			other := *test.b
			other.addr = "test_net.n"
			a, b := test.a.read([]string{"k"}), other.read([]string{"k"})
			got := undecided
			switch {
			case a.holds(b):
				got = holds
			case a.fails(b):
				got = fails
			}
			if got != test.want {
				t.Errorf("got %d, want %d", got, test.want)
			}
			if b.holds(a) != a.holds(b) {
				t.Error("holds isn't symmetric")
			}
		})
	}
}
