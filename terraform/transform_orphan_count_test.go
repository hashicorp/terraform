package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestOrphanResourceCountTransformer(t *testing.T) {
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root"},
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo.2": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},
		},
	})

	g := Graph{Path: addrs.RootModuleInstance}

	{
		tf := &OrphanResourceCountTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			Count:    1,
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			State: state,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceCountBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanResourceCountTransformer_zero(t *testing.T) {
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root"},
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo.2": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},
		},
	})

	g := Graph{Path: addrs.RootModuleInstance}

	{
		tf := &OrphanResourceCountTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			Count:    0,
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			State: state,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceCountZeroStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanResourceCountTransformer_oneNoIndex(t *testing.T) {
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root"},
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo.2": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},
		},
	})

	g := Graph{Path: addrs.RootModuleInstance}

	{
		tf := &OrphanResourceCountTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			Count:    1,
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			State: state,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceCountOneNoIndexStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanResourceCountTransformer_oneIndex(t *testing.T) {
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root"},
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo.1": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},
		},
	})

	g := Graph{Path: addrs.RootModuleInstance}

	{
		tf := &OrphanResourceCountTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			Count:    1,
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			State: state,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceCountOneIndexStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanResourceCountTransformer_zeroAndNone(t *testing.T) {
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root"},
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},
		},
	})

	g := Graph{Path: addrs.RootModuleInstance}

	{
		tf := &OrphanResourceCountTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			Count:    -1,
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			State: state,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceCountZeroAndNoneStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestOrphanResourceCountTransformer_zeroAndNoneCount(t *testing.T) {
	state := MustShimLegacyState(&State{
		Modules: []*ModuleState{
			&ModuleState{
				Path: []string{"root"},
				Resources: map[string]*ResourceState{
					"aws_instance.web": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},

					"aws_instance.foo.0": &ResourceState{
						Type: "aws_instance",
						Primary: &InstanceState{
							ID: "foo",
						},
					},
				},
			},
		},
	})

	g := Graph{Path: addrs.RootModuleInstance}

	{
		tf := &OrphanResourceCountTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			Count:    2,
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			State: state,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceCountZeroAndNoneCountStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

const testTransformOrphanResourceCountBasicStr = `
aws_instance.foo[2] (orphan)
`

const testTransformOrphanResourceCountZeroStr = `
aws_instance.foo (orphan)
aws_instance.foo[2] (orphan)
`

const testTransformOrphanResourceCountOneNoIndexStr = `
aws_instance.foo[2] (orphan)
`

const testTransformOrphanResourceCountOneIndexStr = `
aws_instance.foo[1] (orphan)
`

const testTransformOrphanResourceCountZeroAndNoneStr = `
aws_instance.foo[0] (orphan)
`

const testTransformOrphanResourceCountZeroAndNoneCountStr = `
aws_instance.foo (orphan)
`
