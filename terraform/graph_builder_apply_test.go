package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestApplyGraphBuilder_impl(t *testing.T) {
	var _ GraphBuilder = new(ApplyGraphBuilder)
}

func TestApplyGraphBuilder(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					// Verify noop doesn't show up in graph
					"test_object.noop": &InstanceDiff{},

					"test_object.create": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"test_string": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},

					"test_object.other": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"test_string": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},
				},
			},

			&ModuleDiff{
				Path: []string{"root", "child"},
				Resources: map[string]*InstanceDiff{
					"test_object.create": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"test_string": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},

					"test_object.other": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"test_string": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:        testModule(t, "graph-builder-apply-basic"),
		Diff:          diff,
		Components:    simpleMockComponentFactory(),
		Schemas:       simpleTestSchemas(),
		DisableReduce: true,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong path %q", g.Path.String())
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testApplyGraphBuilderStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// This tests the ordering of two resources where a non-CBD depends
// on a CBD. GH-11349.
func TestApplyGraphBuilder_depCbd(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{"test_object.A": &InstanceDiff{Destroy: true,
					Attributes: map[string]*ResourceAttrDiff{
						"test_string": &ResourceAttrDiff{
							Old:         "",
							New:         "foo",
							RequiresNew: true,
						},
					},
				},

					"test_object.B": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"test_string": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:        testModule(t, "graph-builder-apply-dep-cbd"),
		Diff:          diff,
		Components:    simpleMockComponentFactory(),
		Schemas:       simpleTestSchemas(),
		DisableReduce: true,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Logf("Graph: %s", g.String())

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong path %q", g.Path.String())
	}

	// Create A, Modify B, Destroy A

	testGraphHappensBefore(
		t, g,
		"test_object.A",
		"test_object.A (destroy)")
	testGraphHappensBefore(
		t, g,
		"test_object.A",
		"test_object.B")
	testGraphHappensBefore(
		t, g,
		"test_object.B",
		"test_object.A (destroy)")
}

// This tests the ordering of two resources that are both CBD that
// require destroy/create.
func TestApplyGraphBuilder_doubleCBD(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					"test_object.A": &InstanceDiff{
						Destroy: true,
						Attributes: map[string]*ResourceAttrDiff{
							"test_string": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},

					"test_object.B": &InstanceDiff{
						Destroy: true,
						Attributes: map[string]*ResourceAttrDiff{
							"test_string": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:        testModule(t, "graph-builder-apply-double-cbd"),
		Diff:          diff,
		Components:    simpleMockComponentFactory(),
		Schemas:       simpleTestSchemas(),
		DisableReduce: true,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong path %q", g.Path.String())
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testApplyGraphBuilderDoubleCBDStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

// This tests the ordering of two resources being destroyed that depend
// on each other from only state. GH-11749
func TestApplyGraphBuilder_destroyStateOnly(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root", "child"},
				Resources: map[string]*InstanceDiff{
					"test_object.A": &InstanceDiff{
						Destroy: true,
					},

					"test_object.B": &InstanceDiff{
						Destroy: true,
					},
				},
			},
		},
	}

	state := &State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root", "child"},
				Resources: map[string]*ResourceState{
					"test_object.A": &ResourceState{
						Type: "test_object",
						Primary: &InstanceState{
							ID:         "foo",
							Attributes: map[string]string{},
						},
					},

					"test_object.B": &ResourceState{
						Type: "test_object",
						Primary: &InstanceState{
							ID:         "bar",
							Attributes: map[string]string{},
						},
						Dependencies: []string{"test_object.A"},
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:        testModule(t, "empty"),
		Diff:          diff,
		State:         state,
		Components:    simpleMockComponentFactory(),
		Schemas:       simpleTestSchemas(),
		DisableReduce: true,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Logf("Graph: %s", g.String())

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong path %q", g.Path.String())
	}

	testGraphHappensBefore(
		t, g,
		"module.child.test_object.B (destroy)",
		"module.child.test_object.A (destroy)")
}

// This tests the ordering of destroying a single count of a resource.
func TestApplyGraphBuilder_destroyCount(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					"test_object.A.1": &InstanceDiff{
						Destroy: true,
					},

					"test_object.B": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"name": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:        testModule(t, "graph-builder-apply-count"),
		Diff:          diff,
		Components:    simpleMockComponentFactory(),
		Schemas:       simpleTestSchemas(),
		DisableReduce: true,
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if g.Path.String() != addrs.RootModuleInstance.String() {
		t.Fatalf("wrong module path %q", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testApplyGraphBuilderDestroyCountStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestApplyGraphBuilder_moduleDestroy(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root", "A"},
				Resources: map[string]*InstanceDiff{
					"test_object.foo": &InstanceDiff{
						Destroy: true,
					},
				},
			},

			&ModuleDiff{
				Path: []string{"root", "B"},
				Resources: map[string]*InstanceDiff{
					"test_object.foo": &InstanceDiff{
						Destroy: true,
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:     testModule(t, "graph-builder-apply-module-destroy"),
		Diff:       diff,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphHappensBefore(
		t, g,
		"module.B.test_object.foo (destroy)",
		"module.A.test_object.foo (destroy)",
	)
}

func TestApplyGraphBuilder_provisioner(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					"test_object.foo": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"test_string": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:     testModule(t, "graph-builder-apply-provisioner"),
		Diff:       diff,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphContains(t, g, "provisioner.test")
	testGraphHappensBefore(
		t, g,
		"provisioner.test",
		"test_object.foo",
	)
}

func TestApplyGraphBuilder_provisionerDestroy(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					"test_object.foo": &InstanceDiff{
						Destroy: true,
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Destroy:    true,
		Config:     testModule(t, "graph-builder-apply-provisioner"),
		Diff:       diff,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphContains(t, g, "provisioner.test")
	testGraphHappensBefore(
		t, g,
		"provisioner.test",
		"test_object.foo (destroy)",
	)
}

func TestApplyGraphBuilder_targetModule(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					"test_object.foo": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"test_string": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},
				},
			},
			&ModuleDiff{
				Path: []string{"root", "child2"},
				Resources: map[string]*InstanceDiff{
					"test_object.foo": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"test_string": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Config:     testModule(t, "graph-builder-apply-target-module"),
		Diff:       diff,
		Components: simpleMockComponentFactory(),
		Schemas:    simpleTestSchemas(),
		Targets: []addrs.Targetable{
			addrs.RootModuleInstance.Child("child2", addrs.NoKey),
		},
	}

	g, err := b.Build(addrs.RootModuleInstance)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphNotContains(t, g, "module.child1.output.instance_id")
}

const testApplyGraphBuilderStr = `
meta.count-boundary (count boundary fixup)
  module.child.provisioner.test
  module.child.test_object.create
  module.child.test_object.other
  provider.test
  test_object.create
  test_object.other
module.child.provisioner.test
module.child.test_object.create
  module.child.provisioner.test
  provider.test
module.child.test_object.other
  module.child.test_object.create
  provider.test
provider.test
provider.test (close)
  module.child.test_object.create
  module.child.test_object.other
  provider.test
  test_object.create
  test_object.other
provisioner.test (close)
  module.child.test_object.create
root
  meta.count-boundary (count boundary fixup)
  provider.test (close)
  provisioner.test (close)
test_object.create
  provider.test
test_object.other
  provider.test
  test_object.create
`

const testApplyGraphBuilderDoubleCBDStr = `
meta.count-boundary (count boundary fixup)
  provider.test
  test_object.A
  test_object.A (destroy)
  test_object.B
  test_object.B (destroy)
provider.test
provider.test (close)
  provider.test
  test_object.A
  test_object.A (destroy)
  test_object.B
  test_object.B (destroy)
root
  meta.count-boundary (count boundary fixup)
  provider.test (close)
test_object.A
  provider.test
test_object.A (destroy)
  provider.test
  test_object.A
  test_object.B
  test_object.B (destroy)
test_object.B
  provider.test
  test_object.A
test_object.B (destroy)
  provider.test
  test_object.B
`

const testApplyGraphBuilderDestroyCountStr = `
meta.count-boundary (count boundary fixup)
  provider.test
  test_object.A[1] (destroy)
  test_object.B
provider.test
provider.test (close)
  provider.test
  test_object.A[1] (destroy)
  test_object.B
root
  meta.count-boundary (count boundary fixup)
  provider.test (close)
test_object.A[1] (destroy)
  provider.test
test_object.B
  provider.test
  test_object.A[1] (destroy)
`
