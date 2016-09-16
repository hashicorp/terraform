package terraform

import (
	"strings"
	"testing"
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

const testTransformRefPathStr = `
A
B
  A
child.A
child.B
  child.A
`
