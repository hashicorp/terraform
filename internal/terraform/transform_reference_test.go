package terraform

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
)

func TestReferenceTransformer_simple(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeRefParentTest{
		NameValue: "A",
		Names:     []string{"A"},
	})
	g.Add(&graphNodeRefChildTest{
		NameValue: "B",
		Refs:      []string{"A"},
	})

	tf := &ReferenceTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformRefBasicStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestReferenceTransformer_self(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeRefParentTest{
		NameValue: "A",
		Names:     []string{"A"},
	})
	g.Add(&graphNodeRefChildTest{
		NameValue: "B",
		Refs:      []string{"A", "B"},
	})

	tf := &ReferenceTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformRefBasicStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestReferenceTransformer_path(t *testing.T) {
	g := Graph{Path: addrs.RootModuleInstance}
	g.Add(&graphNodeRefParentTest{
		NameValue: "A",
		Names:     []string{"A"},
	})
	g.Add(&graphNodeRefChildTest{
		NameValue: "B",
		Refs:      []string{"A"},
	})
	g.Add(&graphNodeRefParentTest{
		NameValue: "child.A",
		PathValue: addrs.ModuleInstance{addrs.ModuleInstanceStep{Name: "child"}},
		Names:     []string{"A"},
	})
	g.Add(&graphNodeRefChildTest{
		NameValue: "child.B",
		PathValue: addrs.ModuleInstance{addrs.ModuleInstanceStep{Name: "child"}},
		Refs:      []string{"A"},
	})

	tf := &ReferenceTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformRefPathStr)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestReferenceTransformer_resourceInstances(t *testing.T) {
	// Our reference analyses are all done based on unexpanded addresses
	// so that we can use this transformer both in the plan graph (where things
	// are not expanded yet) and the apply graph (where resource instances are
	// pre-expanded but nothing else is.)
	// However, that would make the result too conservative about instances
	// of the same resource in different instances of the same module, so we
	// make an exception for that situation in particular, keeping references
	// between resource instances segregated by their containing module
	// instance.
	g := Graph{Path: addrs.RootModuleInstance}
	moduleInsts := []addrs.ModuleInstance{
		{
			{
				Name: "foo", InstanceKey: addrs.IntKey(0),
			},
		},
		{
			{
				Name: "foo", InstanceKey: addrs.IntKey(1),
			},
		},
	}
	resourceAs := make([]addrs.AbsResourceInstance, len(moduleInsts))
	for i, moduleInst := range moduleInsts {
		resourceAs[i] = addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "thing",
			Name: "a",
		}.Instance(addrs.NoKey).Absolute(moduleInst)
	}
	resourceBs := make([]addrs.AbsResourceInstance, len(moduleInsts))
	for i, moduleInst := range moduleInsts {
		resourceBs[i] = addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "thing",
			Name: "b",
		}.Instance(addrs.NoKey).Absolute(moduleInst)
	}
	g.Add(&graphNodeFakeResourceInstance{
		Addr: resourceAs[0],
	})
	g.Add(&graphNodeFakeResourceInstance{
		Addr: resourceBs[0],
		Refs: []*addrs.Reference{
			{
				Subject: resourceAs[0].Resource,
			},
		},
	})
	g.Add(&graphNodeFakeResourceInstance{
		Addr: resourceAs[1],
	})
	g.Add(&graphNodeFakeResourceInstance{
		Addr: resourceBs[1],
		Refs: []*addrs.Reference{
			{
				Subject: resourceAs[1].Resource,
			},
		},
	})

	tf := &ReferenceTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Resource B should be connected to resource A in each module instance,
	// but there should be no connections between the two module instances.
	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(`
module.foo[0].thing.a
module.foo[0].thing.b
  module.foo[0].thing.a
module.foo[1].thing.a
module.foo[1].thing.b
  module.foo[1].thing.a
`)
	if actual != expected {
		t.Fatalf("wrong result\n\ngot:\n%s\n\nwant:\n%s", actual, expected)
	}
}

func TestReferenceMapReferences(t *testing.T) {
	cases := map[string]struct {
		Nodes  []dag.Vertex
		Check  dag.Vertex
		Result []string
	}{
		"simple": {
			Nodes: []dag.Vertex{
				&graphNodeRefParentTest{
					NameValue: "A",
					Names:     []string{"A"},
				},
			},
			Check: &graphNodeRefChildTest{
				NameValue: "foo",
				Refs:      []string{"A"},
			},
			Result: []string{"A"},
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			rm := NewReferenceMap(tc.Nodes)
			result := rm.References(tc.Check)

			var resultStr []string
			for _, v := range result {
				resultStr = append(resultStr, dag.VertexName(v))
			}

			sort.Strings(resultStr)
			sort.Strings(tc.Result)
			if !reflect.DeepEqual(resultStr, tc.Result) {
				t.Fatalf("bad: %#v", resultStr)
			}
		})
	}
}

type graphNodeRefParentTest struct {
	NameValue string
	PathValue addrs.ModuleInstance
	Names     []string
}

var _ GraphNodeReferenceable = (*graphNodeRefParentTest)(nil)

func (n *graphNodeRefParentTest) Name() string {
	return n.NameValue
}

func (n *graphNodeRefParentTest) ReferenceableAddrs() []addrs.Referenceable {
	ret := make([]addrs.Referenceable, len(n.Names))
	for i, name := range n.Names {
		ret[i] = addrs.LocalValue{Name: name}
	}
	return ret
}

func (n *graphNodeRefParentTest) Path() addrs.ModuleInstance {
	return n.PathValue
}

func (n *graphNodeRefParentTest) ModulePath() addrs.Module {
	return n.PathValue.Module()
}

type graphNodeRefChildTest struct {
	NameValue string
	PathValue addrs.ModuleInstance
	Refs      []string
}

var _ GraphNodeReferencer = (*graphNodeRefChildTest)(nil)

func (n *graphNodeRefChildTest) Name() string {
	return n.NameValue
}

func (n *graphNodeRefChildTest) References() []*addrs.Reference {
	ret := make([]*addrs.Reference, len(n.Refs))
	for i, name := range n.Refs {
		ret[i] = &addrs.Reference{
			Subject: addrs.LocalValue{Name: name},
		}
	}
	return ret
}

func (n *graphNodeRefChildTest) Path() addrs.ModuleInstance {
	return n.PathValue
}

func (n *graphNodeRefChildTest) ModulePath() addrs.Module {
	return n.PathValue.Module()
}

type graphNodeFakeResourceInstance struct {
	Addr addrs.AbsResourceInstance
	Refs []*addrs.Reference
}

var _ GraphNodeResourceInstance = (*graphNodeFakeResourceInstance)(nil)
var _ GraphNodeReferenceable = (*graphNodeFakeResourceInstance)(nil)
var _ GraphNodeReferencer = (*graphNodeFakeResourceInstance)(nil)

func (n *graphNodeFakeResourceInstance) ResourceInstanceAddr() addrs.AbsResourceInstance {
	return n.Addr
}

func (n *graphNodeFakeResourceInstance) ModulePath() addrs.Module {
	return n.Addr.Module.Module()
}

func (n *graphNodeFakeResourceInstance) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{n.Addr.Resource}
}

func (n *graphNodeFakeResourceInstance) References() []*addrs.Reference {
	return n.Refs
}

func (n *graphNodeFakeResourceInstance) StateDependencies() []addrs.ConfigResource {
	return nil
}

func (n *graphNodeFakeResourceInstance) String() string {
	return n.Addr.String()
}

const testTransformRefBasicStr = `
A
B
  A
`

const testTransformRefPathStr = `
A
B
  A
child.A
child.B
  child.A
`
