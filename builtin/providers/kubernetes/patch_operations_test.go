package kubernetes

import (
	"fmt"
	"testing"
)

func TestDiffStringMap(t *testing.T) {
	testCases := []struct {
		Path        string
		Old         map[string]interface{}
		New         map[string]interface{}
		ExpectedOps PatchOperations
	}{
		{
			Path: "/parent/",
			Old: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			New: map[string]interface{}{
				"one":   "111",
				"two":   "222",
				"three": "333",
			},
			ExpectedOps: []PatchOperation{
				&AddOperation{
					Path:  "/parent/three",
					Value: "333",
				},
			},
		},
		{
			Path: "/parent/",
			Old: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			New: map[string]interface{}{
				"one": "111",
				"two": "abcd",
			},
			ExpectedOps: []PatchOperation{
				&ReplaceOperation{
					Path:  "/parent/two",
					Value: "abcd",
				},
			},
		},
		{
			Path: "/parent/",
			Old: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			New: map[string]interface{}{
				"two":   "abcd",
				"three": "333",
			},
			ExpectedOps: []PatchOperation{
				&RemoveOperation{Path: "/parent/one"},
				&ReplaceOperation{
					Path:  "/parent/two",
					Value: "abcd",
				},
				&AddOperation{
					Path:  "/parent/three",
					Value: "333",
				},
			},
		},
		{
			Path: "/parent/",
			Old: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			New: map[string]interface{}{
				"two": "222",
			},
			ExpectedOps: []PatchOperation{
				&RemoveOperation{Path: "/parent/one"},
			},
		},
		{
			Path: "/parent/",
			Old: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			New: map[string]interface{}{},
			ExpectedOps: []PatchOperation{
				&RemoveOperation{Path: "/parent/one"},
				&RemoveOperation{Path: "/parent/two"},
			},
		},
		{
			Path: "/parent/",
			Old:  map[string]interface{}{},
			New: map[string]interface{}{
				"one": "111",
				"two": "222",
			},
			ExpectedOps: []PatchOperation{
				&AddOperation{
					Path:  "/parent/one",
					Value: "111",
				},
				&AddOperation{
					Path:  "/parent/two",
					Value: "222",
				},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ops := diffStringMap(tc.Path, tc.Old, tc.New)
			if !tc.ExpectedOps.Equal(ops) {
				t.Fatalf("Operations don't match.\nExpected: %v\nGiven:    %v\n", tc.ExpectedOps, ops)
			}
		})
	}

}
