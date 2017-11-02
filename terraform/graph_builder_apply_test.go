package terraform

import (
	"reflect"
	"strings"
	"testing"
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
					"aws_instance.noop": &InstanceDiff{},

					"aws_instance.create": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"name": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},

					"aws_instance.other": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"name": &ResourceAttrDiff{
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
					"aws_instance.create": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"name": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},

					"aws_instance.other": &InstanceDiff{
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
		Module:        testModule(t, "graph-builder-apply-basic"),
		Diff:          diff,
		Providers:     []string{"aws"},
		Provisioners:  []string{"exec"},
		DisableReduce: true,
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(g.Path, RootModulePath) {
		t.Fatalf("bad: %#v", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testApplyGraphBuilderStr)
	if actual != expected {
		t.Fatalf("expected:\n%s\n\ngot:\n%s", expected, actual)
	}
}

// This tests the ordering of two resources where a non-CBD depends
// on a CBD. GH-11349.
func TestApplyGraphBuilder_depCbd(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{"aws_instance.A": &InstanceDiff{Destroy: true,
					Attributes: map[string]*ResourceAttrDiff{
						"name": &ResourceAttrDiff{
							Old:         "",
							New:         "foo",
							RequiresNew: true,
						},
					},
				},

					"aws_instance.B": &InstanceDiff{
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
		Module:        testModule(t, "graph-builder-apply-dep-cbd"),
		Diff:          diff,
		Providers:     []string{"aws"},
		Provisioners:  []string{"exec"},
		DisableReduce: true,
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Logf("Graph: %s", g.String())

	if !reflect.DeepEqual(g.Path, RootModulePath) {
		t.Fatalf("bad: %#v", g.Path)
	}

	// Create A, Modify B, Destroy A

	testGraphHappensBefore(
		t, g,
		"aws_instance.A",
		"aws_instance.A (destroy)")
	testGraphHappensBefore(
		t, g,
		"aws_instance.A",
		"aws_instance.B")
	testGraphHappensBefore(
		t, g,
		"aws_instance.B",
		"aws_instance.A (destroy)")
}

// This tests the ordering of two resources that are both CBD that
// require destroy/create.
func TestApplyGraphBuilder_doubleCBD(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					"aws_instance.A": &InstanceDiff{
						Destroy: true,
						Attributes: map[string]*ResourceAttrDiff{
							"name": &ResourceAttrDiff{
								Old: "",
								New: "foo",
							},
						},
					},

					"aws_instance.B": &InstanceDiff{
						Destroy: true,
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
		Module:        testModule(t, "graph-builder-apply-double-cbd"),
		Diff:          diff,
		Providers:     []string{"aws"},
		Provisioners:  []string{"exec"},
		DisableReduce: true,
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(g.Path, RootModulePath) {
		t.Fatalf("bad: %#v", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testApplyGraphBuilderDoubleCBDStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
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
					"aws_instance.A": &InstanceDiff{
						Destroy: true,
					},

					"aws_instance.B": &InstanceDiff{
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
					"aws_instance.A": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:         "foo",
							Attributes: map[string]string{},
						},
					},

					"aws_instance.B": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID:         "bar",
							Attributes: map[string]string{},
						},
						Dependencies: []string{"aws_instance.A"},
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Module:        testModule(t, "empty"),
		Diff:          diff,
		State:         state,
		Providers:     []string{"aws"},
		DisableReduce: true,
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	t.Logf("Graph: %s", g.String())

	if !reflect.DeepEqual(g.Path, RootModulePath) {
		t.Fatalf("bad: %#v", g.Path)
	}

	testGraphHappensBefore(
		t, g,
		"module.child.aws_instance.B (destroy)",
		"module.child.aws_instance.A (destroy)")
}

// This tests the ordering of destroying a single count of a resource.
func TestApplyGraphBuilder_destroyCount(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					"aws_instance.A.1": &InstanceDiff{
						Destroy: true,
					},

					"aws_instance.B": &InstanceDiff{
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
		Module:        testModule(t, "graph-builder-apply-count"),
		Diff:          diff,
		Providers:     []string{"aws"},
		Provisioners:  []string{"exec"},
		DisableReduce: true,
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if !reflect.DeepEqual(g.Path, RootModulePath) {
		t.Fatalf("bad: %#v", g.Path)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testApplyGraphBuilderDestroyCountStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestApplyGraphBuilder_moduleDestroy(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root", "A"},
				Resources: map[string]*InstanceDiff{
					"null_resource.foo": &InstanceDiff{
						Destroy: true,
					},
				},
			},

			&ModuleDiff{
				Path: []string{"root", "B"},
				Resources: map[string]*InstanceDiff{
					"null_resource.foo": &InstanceDiff{
						Destroy: true,
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Module:    testModule(t, "graph-builder-apply-module-destroy"),
		Diff:      diff,
		Providers: []string{"null"},
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphHappensBefore(
		t, g,
		"module.B.null_resource.foo (destroy)",
		"module.A.null_resource.foo (destroy)")
}

func TestApplyGraphBuilder_provisioner(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					"null_resource.foo": &InstanceDiff{
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
		Module:       testModule(t, "graph-builder-apply-provisioner"),
		Diff:         diff,
		Providers:    []string{"null"},
		Provisioners: []string{"local"},
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphContains(t, g, "provisioner.local")
	testGraphHappensBefore(
		t, g,
		"provisioner.local",
		"null_resource.foo")
}

func TestApplyGraphBuilder_provisionerDestroy(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					"null_resource.foo": &InstanceDiff{
						Destroy: true,
					},
				},
			},
		},
	}

	b := &ApplyGraphBuilder{
		Destroy:      true,
		Module:       testModule(t, "graph-builder-apply-provisioner"),
		Diff:         diff,
		Providers:    []string{"null"},
		Provisioners: []string{"local"},
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphContains(t, g, "provisioner.local")
	testGraphHappensBefore(
		t, g,
		"provisioner.local",
		"null_resource.foo (destroy)")
}

func TestApplyGraphBuilder_targetModule(t *testing.T) {
	diff := &Diff{
		Modules: []*ModuleDiff{
			&ModuleDiff{
				Path: []string{"root"},
				Resources: map[string]*InstanceDiff{
					"null_resource.foo": &InstanceDiff{
						Attributes: map[string]*ResourceAttrDiff{
							"name": &ResourceAttrDiff{
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
					"null_resource.foo": &InstanceDiff{
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
		Module:    testModule(t, "graph-builder-apply-target-module"),
		Diff:      diff,
		Providers: []string{"null"},
		Targets:   []string{"module.child2"},
	}

	g, err := b.Build(RootModulePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	testGraphNotContains(t, g, "module.child1.output.instance_id")
}

const testApplyGraphBuilderStr = `
aws_instance.create
  provider.aws
aws_instance.other
  aws_instance.create
  provider.aws
meta.count-boundary (count boundary fixup)
  aws_instance.create
  aws_instance.other
  module.child.aws_instance.create
  module.child.aws_instance.other
  module.child.provisioner.exec
  provider.aws
module.child.aws_instance.create
  module.child.provisioner.exec
  provider.aws
module.child.aws_instance.other
  module.child.aws_instance.create
  provider.aws
module.child.provisioner.exec
provider.aws
provider.aws (close)
  aws_instance.create
  aws_instance.other
  module.child.aws_instance.create
  module.child.aws_instance.other
  provider.aws
provisioner.exec (close)
  module.child.aws_instance.create
root
  meta.count-boundary (count boundary fixup)
  provider.aws (close)
  provisioner.exec (close)
`

const testApplyGraphBuilderDoubleCBDStr = `
aws_instance.A
  provider.aws
aws_instance.A (destroy)
  aws_instance.A
  aws_instance.B
  aws_instance.B (destroy)
  provider.aws
aws_instance.B
  aws_instance.A
  provider.aws
aws_instance.B (destroy)
  aws_instance.B
  provider.aws
meta.count-boundary (count boundary fixup)
  aws_instance.A
  aws_instance.A (destroy)
  aws_instance.B
  aws_instance.B (destroy)
  provider.aws
provider.aws
provider.aws (close)
  aws_instance.A
  aws_instance.A (destroy)
  aws_instance.B
  aws_instance.B (destroy)
  provider.aws
root
  meta.count-boundary (count boundary fixup)
  provider.aws (close)
`

const testApplyGraphBuilderDestroyCountStr = `
aws_instance.A[1] (destroy)
  provider.aws
aws_instance.B
  aws_instance.A[1] (destroy)
  provider.aws
meta.count-boundary (count boundary fixup)
  aws_instance.A[1] (destroy)
  aws_instance.B
  provider.aws
provider.aws
provider.aws (close)
  aws_instance.A[1] (destroy)
  aws_instance.B
  provider.aws
root
  meta.count-boundary (count boundary fixup)
  provider.aws (close)
`
