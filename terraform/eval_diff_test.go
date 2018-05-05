package terraform

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/hcl2/hcl/hclsyntax"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/configs"
)

func TestEvalFilterDiff(t *testing.T) {
	ctx := new(MockEvalContext)

	cases := []struct {
		Node   *EvalFilterDiff
		Input  *InstanceDiff
		Output *InstanceDiff
	}{
		// With no settings, it returns an empty diff
		{
			&EvalFilterDiff{},
			&InstanceDiff{Destroy: true},
			&InstanceDiff{},
		},

		// Destroy
		{
			&EvalFilterDiff{Destroy: true},
			&InstanceDiff{Destroy: false},
			&InstanceDiff{Destroy: false},
		},
		{
			&EvalFilterDiff{Destroy: true},
			&InstanceDiff{Destroy: true},
			&InstanceDiff{Destroy: true},
		},
		{
			&EvalFilterDiff{Destroy: true},
			&InstanceDiff{
				Destroy: true,
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{},
				},
			},
			&InstanceDiff{Destroy: true},
		},
		{
			&EvalFilterDiff{Destroy: true},
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{
						RequiresNew: true,
					},
				},
			},
			&InstanceDiff{Destroy: true},
		},
		{
			&EvalFilterDiff{Destroy: true},
			&InstanceDiff{
				Attributes: map[string]*ResourceAttrDiff{
					"foo": &ResourceAttrDiff{},
				},
			},
			&InstanceDiff{Destroy: false},
		},
	}

	for i, tc := range cases {
		var output *InstanceDiff
		tc.Node.Diff = &tc.Input
		tc.Node.Output = &output
		if _, err := tc.Node.Eval(ctx); err != nil {
			t.Fatalf("err: %s", err)
		}

		if !reflect.DeepEqual(output, tc.Output) {
			t.Fatalf("bad: %d\n\n%#v", i, output)
		}
	}
}

func TestProcessIgnoreChanges(t *testing.T) {
	var evalDiff *EvalDiff
	var instanceDiff *InstanceDiff

	var testDiffs = func(t *testing.T, ignoreChanges []string, newAttribute string) (*EvalDiff, *InstanceDiff) {
		ignoreChangesTravs := make([]hcl.Traversal, len(ignoreChanges))
		for i, s := range ignoreChanges {
			traversal, travDiags := hclsyntax.ParseTraversalAbs([]byte(s), "", hcl.Pos{Line: 1, Column: 1})
			if travDiags.HasErrors() {
				t.Fatal(travDiags.Error())
			}
			ignoreChangesTravs[i] = traversal
		}

		return &EvalDiff{
				Config: &configs.Resource{
					Managed: &configs.ManagedResource{
						IgnoreChanges: ignoreChangesTravs,
					},
				},
			},
			&InstanceDiff{
				Destroy: true,
				Attributes: map[string]*ResourceAttrDiff{
					"resource.%": {
						Old: "3",
						New: "3",
					},
					"resource.changed": {
						RequiresNew: true,
						Type:        DiffAttrInput,
						Old:         "old",
						New:         "new",
					},
					"resource.maybe": {
						Old: "",
						New: newAttribute,
					},
					"resource.same": {
						Old: "same",
						New: "same",
					},
				},
			}
	}

	for i, tc := range []struct {
		ignore    []string
		newAttr   string
		attrDiffs int
	}{
		// attr diffs should be all (4), or nothing
		{
			ignore:    []string{"resource.changed"},
			attrDiffs: 0,
		},
		{
			ignore:    []string{"resource.changed"},
			newAttr:   "new",
			attrDiffs: 4,
		},
		{
			attrDiffs: 4,
		},
		{
			ignore:    []string{"resource.maybe"},
			newAttr:   "new",
			attrDiffs: 4,
		},
		{
			newAttr:   "new",
			attrDiffs: 4,
		},
		{
			ignore:    []string{"resource"},
			newAttr:   "new",
			attrDiffs: 0,
		},
		// extra ignored values shouldn't effect the diff
		{
			ignore:    []string{"resource.missing", "resource.maybe"},
			newAttr:   "new",
			attrDiffs: 4,
		},
		// this isn't useful, but make sure it doesn't break
		{
			ignore:    []string{"resource.%"},
			attrDiffs: 4,
		},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			evalDiff, instanceDiff = testDiffs(t, tc.ignore, tc.newAttr)
			err := evalDiff.processIgnoreChanges(instanceDiff)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			if len(instanceDiff.Attributes) != tc.attrDiffs {
				t.Errorf("expected %d diffs, found %d", tc.attrDiffs, len(instanceDiff.Attributes))
				for k, attr := range instanceDiff.Attributes {
					fmt.Printf("  %s:%#v\n", k, attr)
				}
			}
		})
	}
}
