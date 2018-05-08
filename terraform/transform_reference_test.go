package terraform

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
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
		PathValue: []string{"root", "child"},
		Names:     []string{"A"},
	})
	g.Add(&graphNodeRefChildTest{
		NameValue: "child.B",
		PathValue: []string{"root", "child"},
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
			result, _ := rm.References(tc.Check)

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

func TestReferenceMapReferencedBy(t *testing.T) {
	cases := map[string]struct {
		Nodes  []dag.Vertex
		Check  dag.Vertex
		Result []string
	}{
		"simple": {
			Nodes: []dag.Vertex{
				&graphNodeRefChildTest{
					NameValue: "A",
					Refs:      []string{"A"},
				},
				&graphNodeRefChildTest{
					NameValue: "B",
					Refs:      []string{"A"},
				},
				&graphNodeRefChildTest{
					NameValue: "C",
					Refs:      []string{"B"},
				},
			},
			Check: &graphNodeRefParentTest{
				NameValue: "foo",
				Names:     []string{"A"},
			},
			Result: []string{"A", "B"},
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			rm := NewReferenceMap(tc.Nodes)
			result := rm.Referrers(tc.Check)

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
	PathValue []string
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
	return normalizeModulePath(n.PathValue)
}

type graphNodeRefChildTest struct {
	NameValue string
	PathValue []string
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
	return normalizeModulePath(n.PathValue)
}

const testTransformRefBasicStr = `
A
B
  A
`

const testTransformRefBackupStr = `
A
B
  A
`

const testTransformRefBackupPrimaryStr = `
A
B
  C
C
`

const testTransformRefModulePathStr = `
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
