package terraform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/states"
)

func TestOrphanResourceTransformer(t *testing.T) {
	mod := testModule(t, "transform-orphan-basic")

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "web",
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

		// The orphan
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "db",
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
		tf := &ConfigTransformer{Config: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &OrphanResourceTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			State:    state,
			Config:   mod,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceBasicStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanResourceTransformer_countGood(t *testing.T) {
	mod := testModule(t, "transform-orphan-count")

	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
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
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance),
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
		tf := &ConfigTransformer{Config: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &OrphanResourceTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			State:    state,
			Config:   mod,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceCountStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanResourceTransformer_countBad(t *testing.T) {
	mod := testModule(t, "transform-orphan-count-empty")
	state := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
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
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "foo",
			}.Instance(addrs.IntKey(1)).Absolute(addrs.RootModuleInstance),
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
		tf := &ConfigTransformer{Config: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &OrphanResourceTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			State:    state,
			Config:   mod,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformOrphanResourceCountBadStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestOrphanResourceTransformer_modules(t *testing.T) {
	mod := testModule(t, "transform-orphan-modules")
	state := states.BuildState(func(s *states.SyncState) {
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
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "aws_instance",
				Name: "web",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("child", addrs.NoKey)),
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
		tf := &ConfigTransformer{Config: mod}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	{
		tf := &OrphanResourceTransformer{
			Concrete: testOrphanResourceConcreteFunc,
			State:    state,
			Config:   mod,
		}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	got := strings.TrimSpace(g.String())
	want := strings.TrimSpace(testTransformOrphanResourceModulesStr)
	if got != want {
		t.Fatalf("wrong state result\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

const testTransformOrphanResourceBasicStr = `
aws_instance.db (orphan)
aws_instance.web
`

const testTransformOrphanResourceCountStr = `
aws_instance.foo
`

const testTransformOrphanResourceCountBadStr = `
aws_instance.foo[0] (orphan)
aws_instance.foo[1] (orphan)
`

const testTransformOrphanResourceModulesStr = `
aws_instance.foo
module.child.aws_instance.web (orphan)
`

func testOrphanResourceConcreteFunc(a *NodeAbstractResourceInstance) dag.Vertex {
	return &testOrphanResourceInstanceConcrete{a}
}

type testOrphanResourceInstanceConcrete struct {
	*NodeAbstractResourceInstance
}

func (n *testOrphanResourceInstanceConcrete) Name() string {
	return fmt.Sprintf("%s (orphan)", n.NodeAbstractResourceInstance.Name())
}
