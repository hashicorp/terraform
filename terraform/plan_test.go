package terraform

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/terraform/configs"
	"github.com/zclconf/go-cty/cty"
)

func TestPlanContextOpts(t *testing.T) {
	plan := &Plan{
		Diff: &Diff{
			Modules: []*ModuleDiff{
				{
					Path: []string{"test"},
				},
			},
		},
		Config: configs.NewEmptyConfig(),
		State: &State{
			TFVersion: "sigil",
		},
		Vars:    map[string]cty.Value{"foo": cty.StringVal("bar")},
		Targets: []string{"baz.bar"},

		TerraformVersion: VersionString(),
		ProviderSHA256s: map[string][]byte{
			"test": []byte("placeholder"),
		},
	}

	got, err := plan.contextOpts(&ContextOpts{})
	if err != nil {
		t.Fatalf("error creating context: %s", err)
	}

	want := &ContextOpts{
		Diff:   plan.Diff,
		Config: plan.Config,
		State:  plan.State,
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("bar"),
				SourceType: ValueFromPlan,
			},
		},
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(addrs.ManagedResourceMode, "baz", "bar"),
		},
		ProviderSHA256s: plan.ProviderSHA256s,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot:  %#v\nwant %#v", got, want)
	}
}

func TestReadWritePlan(t *testing.T) {
	plan := &Plan{
		Config: testModule(t, "new-good"),
		Diff: &Diff{
			Modules: []*ModuleDiff{
				&ModuleDiff{
					Path: rootModulePath,
					Resources: map[string]*InstanceDiff{
						"nodeA": &InstanceDiff{
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

							Meta: map[string]interface{}{
								"foo": []interface{}{1, 2, 3},
							},
						},
					},
				},
			},
		},
		State: &State{
			Modules: []*ModuleState{
				&ModuleState{
					Path: rootModulePath,
					Resources: map[string]*ResourceState{
						"foo": &ResourceState{
							Primary: &InstanceState{
								ID: "bar",
							},
						},
					},
				},
			},
		},
		Vars: map[string]cty.Value{
			"foo": cty.StringVal("bar"),
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

	actualStr := strings.TrimSpace(actual.String())
	expectedStr := strings.TrimSpace(plan.String())
	if actualStr != expectedStr {
		t.Fatalf("bad:\n\n%s\n\nexpected:\n\n%s", actualStr, expectedStr)
	}
}

func TestPlanContextOptsOverrideStateGood(t *testing.T) {
	plan := &Plan{
		Diff: &Diff{
			Modules: []*ModuleDiff{
				{
					Path: []string{"test"},
				},
			},
		},
		Config: configs.NewEmptyConfig(),
		State: &State{
			TFVersion: "sigil",
			Serial:    1,
		},
		Vars:    map[string]cty.Value{"foo": cty.StringVal("bar")},
		Targets: []string{"baz.bar"},

		TerraformVersion: VersionString(),
		ProviderSHA256s: map[string][]byte{
			"test": []byte("placeholder"),
		},
	}

	base := &ContextOpts{
		State: &State{
			TFVersion: "sigil",
			Serial:    2,
		},
	}

	got, err := plan.contextOpts(base)
	if err != nil {
		t.Fatalf("error creating context: %s", err)
	}

	want := &ContextOpts{
		Diff:   plan.Diff,
		Config: plan.Config,
		State:  base.State,
		Variables: InputValues{
			"foo": &InputValue{
				Value:      cty.StringVal("bar"),
				SourceType: ValueFromPlan,
			},
		},
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Resource(addrs.ManagedResourceMode, "baz", "bar"),
		},
		ProviderSHA256s: plan.ProviderSHA256s,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot:  %#v\nwant %#v", got, want)
	}
}
