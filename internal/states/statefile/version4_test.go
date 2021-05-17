package statefile

import (
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// This test verifies that modules are sorted before resources:
// https://github.com/hashicorp/terraform/issues/21552
func TestVersion4_sort(t *testing.T) {
	resources := sortResourcesV4{
		{
			Module: "module.child",
			Type:   "test_instance",
			Name:   "foo",
		},
		{
			Type: "test_instance",
			Name: "foo",
		},
		{
			Module: "module.kinder",
			Type:   "test_instance",
			Name:   "foo",
		},
		{
			Module: "module.child.grandchild",
			Type:   "test_instance",
			Name:   "foo",
		},
	}
	sort.Stable(resources)

	moduleOrder := []string{"", "module.child", "module.child.grandchild", "module.kinder"}

	for i, resource := range resources {
		if resource.Module != moduleOrder[i] {
			t.Errorf("wrong sort order: expected %q, got %q\n", moduleOrder[i], resource.Module)
		}
	}
}

func TestVersion4_unmarshalPaths(t *testing.T) {
	testCases := map[string]struct {
		json  string
		paths []cty.Path
		diags []string
	}{
		"no paths": {
			json:  `[]`,
			paths: []cty.Path{},
		},
		"attribute path": {
			json: `[
  [
    {
      "type": "get_attr",
	  "value": "password"
    }
  ]
]`,
			paths: []cty.Path{cty.GetAttrPath("password")},
		},
		"attribute and string index": {
			json: `[
  [
    {
      "type": "get_attr",
	  "value": "triggers"
    },
    {
      "type": "index",
      "value": {
        "value": "secret",
		"type": "string"
      }
    }
  ]
]`,
			paths: []cty.Path{cty.GetAttrPath("triggers").IndexString("secret")},
		},
		"attribute, number index, attribute": {
			json: `[
  [
    {
      "type": "get_attr",
	  "value": "identities"
    },
    {
      "type": "index",
      "value": {
        "value": 2,
		"type": "number"
      }
    },
    {
      "type": "get_attr",
      "value": "private_key"
    }
  ]
]`,
			paths: []cty.Path{cty.GetAttrPath("identities").IndexInt(2).GetAttr("private_key")},
		},
		"multiple paths": {
			json: `[
  [
    {
      "type": "get_attr",
	  "value": "alpha"
    }
  ],
  [
    {
      "type": "get_attr",
	  "value": "beta"
    }
  ],
  [
    {
      "type": "get_attr",
	  "value": "gamma"
    }
  ]
]`,
			paths: []cty.Path{cty.GetAttrPath("alpha"), cty.GetAttrPath("beta"), cty.GetAttrPath("gamma")},
		},
		"errors": {
			json: `[
  [
    {
      "type": "get_attr",
	  "value": 5
    }
  ],
  [
    {
      "type": "index",
	  "value": "test"
    }
  ],
  [
    {
      "type": "invalid_type",
	  "value": ["this is invalid too"]
    }
  ]
]`,
			paths: []cty.Path{},
			diags: []string{
				"Failed to unmarshal get attr step name",
				"Failed to unmarshal index step key",
				"Unsupported path step",
			},
		},
		"one invalid path, others valid": {
			json: `[
  [
    {
      "type": "get_attr",
	  "value": "alpha"
    }
  ],
  [
    {
      "type": "invalid_type",
	  "value": ["this is invalid too"]
    }
  ],
  [
    {
      "type": "get_attr",
	  "value": "gamma"
    }
  ]
]`,
			paths: []cty.Path{cty.GetAttrPath("alpha"), cty.GetAttrPath("gamma")},
			diags: []string{"Unsupported path step"},
		},
		"invalid structure": {
			json:  `{}`,
			paths: []cty.Path{},
			diags: []string{"Error unmarshaling path steps"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			paths, diags := unmarshalPaths([]byte(tc.json))

			if len(tc.diags) == 0 {
				if len(diags) != 0 {
					t.Errorf("expected no diags, got: %#v", diags)
				}
			} else {
				if got, want := len(diags), len(tc.diags); got != want {
					t.Fatalf("got %d diags, want %d\n%s", got, want, diags.Err())
				}
				for i := range tc.diags {
					got := tfdiags.Diagnostics{diags[i]}.Err().Error()
					if !strings.Contains(got, tc.diags[i]) {
						t.Errorf("expected diag %d to contain %q, but was:\n%s", i, tc.diags[i], got)
					}
				}
			}

			if len(paths) != len(tc.paths) {
				t.Fatalf("got %d paths, want %d", len(paths), len(tc.paths))
			}
			for i, path := range paths {
				if !path.Equals(tc.paths[i]) {
					t.Errorf("wrong paths\n got: %#v\nwant: %#v", path, tc.paths[i])
				}
			}
		})
	}
}

func TestVersion4_marshalPaths(t *testing.T) {
	testCases := map[string]struct {
		paths []cty.Path
		json  string
	}{
		"no paths": {
			paths: []cty.Path{},
			json:  `[]`,
		},
		"attribute path": {
			paths: []cty.Path{cty.GetAttrPath("password")},
			json:  `[[{"type":"get_attr","value":"password"}]]`,
		},
		"attribute, number index, attribute": {
			paths: []cty.Path{cty.GetAttrPath("identities").IndexInt(2).GetAttr("private_key")},
			json:  `[[{"type":"get_attr","value":"identities"},{"type":"index","value":{"value":2,"type":"number"}},{"type":"get_attr","value":"private_key"}]]`,
		},
		"multiple paths": {
			paths: []cty.Path{cty.GetAttrPath("a"), cty.GetAttrPath("b"), cty.GetAttrPath("c")},
			json:  `[[{"type":"get_attr","value":"a"}],[{"type":"get_attr","value":"b"}],[{"type":"get_attr","value":"c"}]]`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			json, diags := marshalPaths(tc.paths)

			if len(diags) != 0 {
				t.Fatalf("expected no diags, got: %#v", diags)
			}

			if got, want := string(json), tc.json; got != want {
				t.Fatalf("wrong JSON output\n got: %s\nwant: %s\n", got, want)
			}
		})
	}
}
