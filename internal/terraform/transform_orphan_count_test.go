package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

func TestOrphanResourceCountTransformer(t *testing.T) {
	state := states.NewState()
	root := state.RootModule()
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.web").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[2]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	g := Graph{Path: addrs.RootModuleInstance}

	{
		tf := &OrphanResourceInstanceCountTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			InstanceAddrs: []addrs.AbsResourceInstance{mustResourceInstanceAddr("aws_instance.foo[0]")},
			State:         state,
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
	state := states.NewState()
	root := state.RootModule()
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.web").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[2]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	g := Graph{Path: addrs.RootModuleInstance}

	{
		tf := &OrphanResourceInstanceCountTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			InstanceAddrs: []addrs.AbsResourceInstance{},
			State:         state,
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

func TestOrphanResourceCountTransformer_oneIndex(t *testing.T) {
	state := states.NewState()
	root := state.RootModule()
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.web").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[0]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)
	root.SetResourceInstanceCurrent(
		mustResourceInstanceAddr("aws_instance.foo[1]").Resource,
		&states.ResourceInstanceObjectSrc{
			Status:    states.ObjectReady,
			AttrsJSON: []byte(`{"id":"foo"}`),
		},
		mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
	)

	g := Graph{Path: addrs.RootModuleInstance}

	{
		tf := &OrphanResourceInstanceCountTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			InstanceAddrs: []addrs.AbsResourceInstance{mustResourceInstanceAddr("aws_instance.foo[0]")},
			State:         state,
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
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
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
			mustProviderConfig(`provider["registry.terraform.io/hashicorp/aws"]`),
		)
	})

	g := Graph{Path: addrs.RootModuleInstance}

	{
		tf := &OrphanResourceInstanceCountTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			Addr: addrs.RootModuleInstance.Resource(
				addrs.ManagedResourceMode, "aws_instance", "foo",
			),
			InstanceAddrs: []addrs.AbsResourceInstance{},
			State:         state,
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
aws_instance.foo[0] (orphan)
aws_instance.foo[2] (orphan)
`

const testTransformOrphanResourceCountOneIndexStr = `
aws_instance.foo[1] (orphan)
`

const testTransformOrphanResourceForEachStr = `
aws_instance.foo (orphan)
aws_instance.foo["bar"] (orphan)
`
