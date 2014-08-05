package terraform

import (
	"bytes"
	"reflect"

	"testing"
)

func TestReadWritePlan(t *testing.T) {
	plan := &Plan{
		Config: testConfig(t, "new-good"),
		Diff: &Diff{
			Resources: map[string]*ResourceDiff{
				"nodeA": &ResourceDiff{
					Attributes: map[string]*ResourceAttrDiff{
						"foo": &ResourceAttrDiff{
							Old: "foo",
							New: "bar",
						},
						"bar": &ResourceAttrDiff{
							Old:         "foo",
							NewComputed: true,
						},
						"longfoo": &ResourceAttrDiff{
							Old:         "foo",
							New:         "bar",
							RequiresNew: true,
						},
					},
				},
			},
		},
		State: &State{
			Resources: map[string]*ResourceState{
				"foo": &ResourceState{
					ID: "bar",
				},
			},
		},
		Vars: map[string]string{
			"foo": "bar",
		},
	}

	buf := new(bytes.Buffer)
	if err := WritePlan(plan, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual, err := ReadPlan(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	println(reflect.DeepEqual(actual.Config.Resources, plan.Config.Resources))

	if !reflect.DeepEqual(actual, plan) {
		t.Fatalf("bad: %#v", actual)
	}
}
