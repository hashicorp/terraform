package terraform

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/config"
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

func TestProcessIgnoreChangesOnResourceIgnoredWithRequiresNew(t *testing.T) {
	var evalDiff *EvalDiff
	var instanceDiff *InstanceDiff

	var testDiffs = func(ignoreChanges []string, newAttribute string) (*EvalDiff, *InstanceDiff) {
		return &EvalDiff{
				Resource: &config.Resource{
					Lifecycle: config.ResourceLifecycle{
						IgnoreChanges: ignoreChanges,
					},
				},
			},
			&InstanceDiff{
				Destroy: true,
				Attributes: map[string]*ResourceAttrDiff{
					"resource.changed": {
						RequiresNew: true,
						Type:        DiffAttrInput,
						Old:         "old",
						New:         "new",
					},
					"resource.unchanged": {
						Old: "unchanged",
						New: newAttribute,
					},
				},
			}
	}

	evalDiff, instanceDiff = testDiffs([]string{"resource.changed"}, "unchanged")
	err := evalDiff.processIgnoreChanges(instanceDiff)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(instanceDiff.Attributes) > 0 {
		t.Fatalf("Expected all resources to be ignored, found %d", len(instanceDiff.Attributes))
	}

	evalDiff, instanceDiff = testDiffs([]string{}, "unchanged")
	err = evalDiff.processIgnoreChanges(instanceDiff)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(instanceDiff.Attributes) != 2 {
		t.Fatalf("Expected 2 resources to be found, found %d", len(instanceDiff.Attributes))
	}

	evalDiff, instanceDiff = testDiffs([]string{"resource.changed"}, "changed")
	err = evalDiff.processIgnoreChanges(instanceDiff)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(instanceDiff.Attributes) != 1 {
		t.Fatalf("Expected 1 resource to be found, found %d", len(instanceDiff.Attributes))
	}

	evalDiff, instanceDiff = testDiffs([]string{}, "changed")
	err = evalDiff.processIgnoreChanges(instanceDiff)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if len(instanceDiff.Attributes) != 2 {
		t.Fatalf("Expected 2 resource to be found, found %d", len(instanceDiff.Attributes))
	}
}
