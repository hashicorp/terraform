package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/zclconf/go-cty/cty"
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

// When converting from a NoEach mode to an EachMap via a switch to for_each,
// an edge is necessary to ensure that the map-key'd instances
// are evaluated after the NoKey resource, because the final instance evaluated
// sets the whole resource's EachMode.
func TestOrphanResourceCountTransformer_ForEachEdgesAdded(t *testing.T) {
	state := states.BuildState(func(s *states.SyncState) {
		// "bar" key'd resource
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			}.Instance(addrs.StringKey("bar")).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsFlat: map[string]string{
					"id": "foo",
				},
				Status: states.ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(addrs.RootModuleInstance),
		)

		// NoKey'd resource
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsFlat: map[string]string{
					"id": "foo",
				},
				Status: states.ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "aws",
			}.Absolute(addrs.RootModuleInstance),
		)
	})

	g := Graph{Path: addrs.RootModuleInstance}

	{
		tf := &OrphanResourceCountTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			// No keys in this ForEach ensure both our resources end
			// up orphaned in this test
			ForEach: map[string]cty.Value{},
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
	expected := strings.TrimSpace(testTransformOrphanResourceForEachStr)
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

const testTransformOrphanResourceForEachStr = `
aws_instance.foo (orphan)
aws_instance.foo["bar"] (orphan)
  aws_instance.foo (orphan)
`
