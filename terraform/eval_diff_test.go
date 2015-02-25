package terraform

import (
	"reflect"
	"testing"
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
