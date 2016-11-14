package terraform

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestReferenceTransformer_simple(t *testing.T) {
	g := Graph{Path: RootModulePath}
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
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestReferenceTransformer_self(t *testing.T) {
	g := Graph{Path: RootModulePath}
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
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestReferenceTransformer_path(t *testing.T) {
	g := Graph{Path: RootModulePath}
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
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestReferenceTransformer_backup(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeRefParentTest{
		NameValue: "A",
		Names:     []string{"A"},
	})
	g.Add(&graphNodeRefChildTest{
		NameValue: "B",
		Refs:      []string{"C/A"},
	})

	tf := &ReferenceTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformRefBackupStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestReferenceTransformer_backupPrimary(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeRefParentTest{
		NameValue: "A",
		Names:     []string{"A"},
	})
	g.Add(&graphNodeRefChildTest{
		NameValue: "B",
		Refs:      []string{"C/A"},
	})
	g.Add(&graphNodeRefParentTest{
		NameValue: "C",
		Names:     []string{"C"},
	})

	tf := &ReferenceTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformRefBackupPrimaryStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestReferenceTransformer_modulePath(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeRefParentTest{
		NameValue: "A",
		Names:     []string{"A"},
		PathValue: []string{"foo"},
	})
	g.Add(&graphNodeRefChildTest{
		NameValue: "B",
		Refs:      []string{"module.foo"},
	})

	tf := &ReferenceTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformRefModulePathStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
	}
}

func TestReferenceTransformer_modulePathNormalized(t *testing.T) {
	g := Graph{Path: RootModulePath}
	g.Add(&graphNodeRefParentTest{
		NameValue: "A",
		Names:     []string{"A"},
		PathValue: []string{"root", "foo"},
	})
	g.Add(&graphNodeRefChildTest{
		NameValue: "B",
		Refs:      []string{"module.foo"},
	})

	tf := &ReferenceTransformer{}
	if err := tf.Transform(&g); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testTransformRefModulePathStr)
	if actual != expected {
		t.Fatalf("bad:\n\n%s", actual)
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
			result := rm.ReferencedBy(tc.Check)

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

func (n *graphNodeRefParentTest) Name() string                { return n.NameValue }
func (n *graphNodeRefParentTest) ReferenceableName() []string { return n.Names }
func (n *graphNodeRefParentTest) Path() []string              { return n.PathValue }

type graphNodeRefChildTest struct {
	NameValue string
	PathValue []string
	Refs      []string
}

func (n *graphNodeRefChildTest) Name() string         { return n.NameValue }
func (n *graphNodeRefChildTest) References() []string { return n.Refs }
func (n *graphNodeRefChildTest) Path() []string       { return n.PathValue }

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
