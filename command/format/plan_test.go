package format

import (
	"reflect"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/colorstring"
)

var disabledColorize = &colorstring.Colorize{
	Colors:  colorstring.DefaultColors,
	Disable: true,
}

func TestNewPlan(t *testing.T) {
	tests := map[string]struct {
		Input *terraform.Plan
		Want  *Plan
	}{
		"nil input": {
			Input: nil,
			Want: &Plan{
				Resources: nil,
			},
		},
		"nil diff": {
			Input: &terraform.Plan{},
			Want: &Plan{
				Resources: nil,
			},
		},
		"empty diff": {
			Input: &terraform.Plan{
				Diff: &terraform.Diff{
					Modules: []*terraform.ModuleDiff{
						{
							Path:      []string{"root"},
							Resources: map[string]*terraform.InstanceDiff{},
						},
					},
				},
			},
			Want: &Plan{
				Resources: nil,
			},
		},
		"create managed resource": {
			Input: &terraform.Plan{
				Diff: &terraform.Diff{
					Modules: []*terraform.ModuleDiff{
						{
							Path: []string{"root"},
							Resources: map[string]*terraform.InstanceDiff{
								"test_resource.foo": {
									Attributes: map[string]*terraform.ResourceAttrDiff{
										"id": {
											NewComputed: true,
											RequiresNew: true,
										},
									},
								},
							},
						},
					},
				},
			},
			Want: &Plan{
				Resources: []*InstanceDiff{
					{
						Addr:   mustParseResourceAddress("test_resource.foo"),
						Action: terraform.DiffCreate,
						Attributes: []*AttributeDiff{
							{
								Path:        "id",
								Action:      terraform.DiffCreate,
								NewComputed: true,
								ForcesNew:   true,
							},
						},
					},
				},
			},
		},
		"create managed resource in child module": {
			Input: &terraform.Plan{
				Diff: &terraform.Diff{
					Modules: []*terraform.ModuleDiff{
						{
							Path: []string{"root"},
							Resources: map[string]*terraform.InstanceDiff{
								"test_resource.foo": {
									Attributes: map[string]*terraform.ResourceAttrDiff{
										"id": {
											NewComputed: true,
											RequiresNew: true,
										},
									},
								},
							},
						},
						{
							Path: []string{"root", "foo"},
							Resources: map[string]*terraform.InstanceDiff{
								"test_resource.foo": {
									Attributes: map[string]*terraform.ResourceAttrDiff{
										"id": {
											NewComputed: true,
											RequiresNew: true,
										},
									},
								},
							},
						},
					},
				},
			},
			Want: &Plan{
				Resources: []*InstanceDiff{
					{
						Addr:   mustParseResourceAddress("test_resource.foo"),
						Action: terraform.DiffCreate,
						Attributes: []*AttributeDiff{
							{
								Path:        "id",
								Action:      terraform.DiffCreate,
								NewComputed: true,
								ForcesNew:   true,
							},
						},
					},
					{
						Addr:   mustParseResourceAddress("module.foo.test_resource.foo"),
						Action: terraform.DiffCreate,
						Attributes: []*AttributeDiff{
							{
								Path:        "id",
								Action:      terraform.DiffCreate,
								NewComputed: true,
								ForcesNew:   true,
							},
						},
					},
				},
			},
		},
		"create data resource": {
			Input: &terraform.Plan{
				Diff: &terraform.Diff{
					Modules: []*terraform.ModuleDiff{
						{
							Path: []string{"root"},
							Resources: map[string]*terraform.InstanceDiff{
								"data.test_data_source.foo": {
									Attributes: map[string]*terraform.ResourceAttrDiff{
										"id": {
											NewComputed: true,
											RequiresNew: true,
										},
									},
								},
							},
						},
					},
				},
			},
			Want: &Plan{
				Resources: []*InstanceDiff{
					{
						Addr:   mustParseResourceAddress("data.test_data_source.foo"),
						Action: terraform.DiffRefresh,
						Attributes: []*AttributeDiff{
							{
								Path:        "id",
								Action:      terraform.DiffUpdate,
								NewComputed: true,
								ForcesNew:   true,
							},
						},
					},
				},
			},
		},
		"destroy managed resource": {
			Input: &terraform.Plan{
				Diff: &terraform.Diff{
					Modules: []*terraform.ModuleDiff{
						{
							Path: []string{"root"},
							Resources: map[string]*terraform.InstanceDiff{
								"test_resource.foo": {
									Destroy: true,
								},
							},
						},
					},
				},
			},
			Want: &Plan{
				Resources: []*InstanceDiff{
					{
						Addr:   mustParseResourceAddress("test_resource.foo"),
						Action: terraform.DiffDestroy,
					},
				},
			},
		},
		"destroy data resource": {
			Input: &terraform.Plan{
				Diff: &terraform.Diff{
					Modules: []*terraform.ModuleDiff{
						{
							Path: []string{"root"},
							Resources: map[string]*terraform.InstanceDiff{
								"data.test_data_source.foo": {
									Destroy: true,
								},
							},
						},
					},
				},
			},
			Want: &Plan{
				// Data source destroys are not shown
				Resources: nil,
			},
		},
		"destroy many instances of a resource": {
			Input: &terraform.Plan{
				Diff: &terraform.Diff{
					Modules: []*terraform.ModuleDiff{
						{
							Path: []string{"root"},
							Resources: map[string]*terraform.InstanceDiff{
								"test_resource.foo.0": {
									Destroy: true,
								},
								"test_resource.foo.1": {
									Destroy: true,
								},
								"test_resource.foo.10": {
									Destroy: true,
								},
								"test_resource.foo.2": {
									Destroy: true,
								},
								"test_resource.foo.3": {
									Destroy: true,
								},
								"test_resource.foo.4": {
									Destroy: true,
								},
								"test_resource.foo.5": {
									Destroy: true,
								},
								"test_resource.foo.6": {
									Destroy: true,
								},
								"test_resource.foo.7": {
									Destroy: true,
								},
								"test_resource.foo.8": {
									Destroy: true,
								},
								"test_resource.foo.9": {
									Destroy: true,
								},
							},
						},
					},
				},
			},
			Want: &Plan{
				Resources: []*InstanceDiff{
					{
						Addr:   mustParseResourceAddress("test_resource.foo[0]"),
						Action: terraform.DiffDestroy,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.foo[1]"),
						Action: terraform.DiffDestroy,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.foo[2]"),
						Action: terraform.DiffDestroy,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.foo[3]"),
						Action: terraform.DiffDestroy,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.foo[4]"),
						Action: terraform.DiffDestroy,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.foo[5]"),
						Action: terraform.DiffDestroy,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.foo[6]"),
						Action: terraform.DiffDestroy,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.foo[7]"),
						Action: terraform.DiffDestroy,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.foo[8]"),
						Action: terraform.DiffDestroy,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.foo[9]"),
						Action: terraform.DiffDestroy,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.foo[10]"),
						Action: terraform.DiffDestroy,
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := NewPlan(test.Input)
			if !reflect.DeepEqual(got, test.Want) {
				t.Errorf(
					"wrong result\ninput: %sgot: %swant:%s",
					spew.Sdump(test.Input),
					spew.Sdump(got),
					spew.Sdump(test.Want),
				)
			}
		})
	}
}

func TestPlanStats(t *testing.T) {
	tests := map[string]struct {
		Input *Plan
		Want  PlanStats
	}{
		"empty": {
			&Plan{},
			PlanStats{},
		},
		"destroy": {
			&Plan{
				Resources: []*InstanceDiff{
					{
						Addr:   mustParseResourceAddress("test_resource.foo"),
						Action: terraform.DiffDestroy,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.bar"),
						Action: terraform.DiffDestroy,
					},
				},
			},
			PlanStats{
				ToDestroy: 2,
			},
		},
		"create": {
			&Plan{
				Resources: []*InstanceDiff{
					{
						Addr:   mustParseResourceAddress("test_resource.foo"),
						Action: terraform.DiffCreate,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.bar"),
						Action: terraform.DiffCreate,
					},
				},
			},
			PlanStats{
				ToAdd: 2,
			},
		},
		"update": {
			&Plan{
				Resources: []*InstanceDiff{
					{
						Addr:   mustParseResourceAddress("test_resource.foo"),
						Action: terraform.DiffUpdate,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.bar"),
						Action: terraform.DiffUpdate,
					},
				},
			},
			PlanStats{
				ToChange: 2,
			},
		},
		"data source refresh": {
			&Plan{
				Resources: []*InstanceDiff{
					{
						Addr:   mustParseResourceAddress("data.test.foo"),
						Action: terraform.DiffRefresh,
					},
				},
			},
			PlanStats{
			// data resource refreshes are not counted in our stats
			},
		},
		"replace": {
			&Plan{
				Resources: []*InstanceDiff{
					{
						Addr:   mustParseResourceAddress("test_resource.foo"),
						Action: terraform.DiffDestroyCreate,
					},
					{
						Addr:   mustParseResourceAddress("test_resource.bar"),
						Action: terraform.DiffDestroyCreate,
					},
				},
			},
			PlanStats{
				ToDestroy: 2,
				ToAdd:     2,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := test.Input.Stats()
			if !reflect.DeepEqual(got, test.Want) {
				t.Errorf(
					"wrong result\ninput: %sgot: %swant:%s",
					spew.Sdump(test.Input),
					spew.Sdump(got),
					spew.Sdump(test.Want),
				)
			}
		})
	}
}

// Test that deposed instances are marked as such
func TestPlan_destroyDeposed(t *testing.T) {
	plan := &terraform.Plan{
		Diff: &terraform.Diff{
			Modules: []*terraform.ModuleDiff{
				&terraform.ModuleDiff{
					Path: []string{"root"},
					Resources: map[string]*terraform.InstanceDiff{
						"aws_instance.foo": &terraform.InstanceDiff{
							DestroyDeposed: true,
						},
					},
				},
			},
		},
	}
	dispPlan := NewPlan(plan)
	actual := dispPlan.Format(disabledColorize)

	expected := strings.TrimSpace(`
- aws_instance.foo (deposed)
	`)
	if actual != expected {
		t.Fatalf("expected:\n\n%s\n\ngot:\n\n%s", expected, actual)
	}
}

// Test that computed fields with an interpolation string get displayed
func TestPlan_displayInterpolations(t *testing.T) {
	plan := &terraform.Plan{
		Diff: &terraform.Diff{
			Modules: []*terraform.ModuleDiff{
				&terraform.ModuleDiff{
					Path: []string{"root"},
					Resources: map[string]*terraform.InstanceDiff{
						"aws_instance.foo": &terraform.InstanceDiff{
							Attributes: map[string]*terraform.ResourceAttrDiff{
								"computed_field": &terraform.ResourceAttrDiff{
									New:         "${aws_instance.other.id}",
									NewComputed: true,
								},
							},
						},
					},
				},
			},
		},
	}
	dispPlan := NewPlan(plan)
	out := dispPlan.Format(disabledColorize)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatal("expected 2 lines of output, got:\n", out)
	}

	actual := strings.TrimSpace(lines[1])
	expected := `computed_field: "" => "${aws_instance.other.id}"`

	if actual != expected {
		t.Fatalf("expected:\n\n%s\n\ngot:\n\n%s", expected, actual)
	}
}

// Ensure that (forces new resource) text is included
// https://github.com/hashicorp/terraform/issues/16035
func TestPlan_forcesNewResource(t *testing.T) {
	plan := &terraform.Plan{
		Diff: &terraform.Diff{
			Modules: []*terraform.ModuleDiff{
				&terraform.ModuleDiff{
					Path: []string{"root"},
					Resources: map[string]*terraform.InstanceDiff{
						"test_resource.foo": &terraform.InstanceDiff{
							Destroy: true,
							Attributes: map[string]*terraform.ResourceAttrDiff{
								"A": &terraform.ResourceAttrDiff{
									New:         "B",
									RequiresNew: true,
								},
							},
						},
					},
				},
			},
		},
	}
	dispPlan := NewPlan(plan)
	actual := dispPlan.Format(disabledColorize)

	expected := strings.TrimSpace(`
-/+ test_resource.foo (new resource required)
      A: "" => "B" (forces new resource)
	`)
	if actual != expected {
		t.Fatalf("expected:\n\n%s\n\ngot:\n\n%s", expected, actual)
	}
}

// Test that a root level data source gets a special plan output on create
func TestPlan_rootDataSource(t *testing.T) {
	plan := &terraform.Plan{
		Diff: &terraform.Diff{
			Modules: []*terraform.ModuleDiff{
				&terraform.ModuleDiff{
					Path: []string{"root"},
					Resources: map[string]*terraform.InstanceDiff{
						"data.type.name": &terraform.InstanceDiff{
							Attributes: map[string]*terraform.ResourceAttrDiff{
								"A": &terraform.ResourceAttrDiff{
									New:         "B",
									RequiresNew: true,
								},
							},
						},
					},
				},
			},
		},
	}
	dispPlan := NewPlan(plan)
	actual := dispPlan.Format(disabledColorize)

	expected := strings.TrimSpace(`
 <= data.type.name
      A: "B"
	`)
	if actual != expected {
		t.Fatalf("expected:\n\n%s\n\ngot:\n\n%s", expected, actual)
	}
}

// Test that data sources nested in modules get the same plan output
func TestPlan_nestedDataSource(t *testing.T) {
	plan := &terraform.Plan{
		Diff: &terraform.Diff{
			Modules: []*terraform.ModuleDiff{
				&terraform.ModuleDiff{
					Path: []string{"root", "nested"},
					Resources: map[string]*terraform.InstanceDiff{
						"data.type.name": &terraform.InstanceDiff{
							Attributes: map[string]*terraform.ResourceAttrDiff{
								"A": &terraform.ResourceAttrDiff{
									New:         "B",
									RequiresNew: true,
								},
							},
						},
					},
				},
			},
		},
	}
	dispPlan := NewPlan(plan)
	actual := dispPlan.Format(disabledColorize)

	expected := strings.TrimSpace(`
 <= module.nested.data.type.name
      A: "B"
	`)
	if actual != expected {
		t.Fatalf("expected:\n\n%s\n\ngot:\n\n%s", expected, actual)
	}
}

func mustParseResourceAddress(s string) *terraform.ResourceAddress {
	addr, err := terraform.ParseResourceAddress(s)
	if err != nil {
		panic(err)
	}
	return addr
}
