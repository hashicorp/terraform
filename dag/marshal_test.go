package dag

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestGraphDot_empty(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Add(3)

	actual := strings.TrimSpace(string(g.Dot(nil)))
	expected := strings.TrimSpace(testGraphDotEmptyStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphDot_basic(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(1, 3))

	actual := strings.TrimSpace(string(g.Dot(nil)))
	expected := strings.TrimSpace(testGraphDotBasicStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphDot_attrs(t *testing.T) {
	var g Graph
	g.Add(&testGraphNodeDotter{
		Result: &DotNode{
			Name:  "foo",
			Attrs: map[string]string{"foo": "bar"},
		},
	})

	actual := strings.TrimSpace(string(g.Dot(nil)))
	expected := strings.TrimSpace(testGraphDotAttrsStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

type testGraphNodeDotter struct{ Result *DotNode }

func (n *testGraphNodeDotter) Name() string                      { return n.Result.Name }
func (n *testGraphNodeDotter) DotNode(string, *DotOpts) *DotNode { return n.Result }

const testGraphDotBasicStr = `digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] 1" -> "[root] 3"
	}
}
`

const testGraphDotEmptyStr = `digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
	}
}`

const testGraphDotAttrsStr = `digraph {
	compound = "true"
	newrank = "true"
	subgraph "root" {
		"[root] foo" [foo = "bar"]
	}
}`

func TestGraphJSON_empty(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Add(3)

	js, err := g.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	actual := strings.TrimSpace(string(js))
	expected := strings.TrimSpace(testGraphJSONEmptyStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

func TestGraphJSON_basic(t *testing.T) {
	var g Graph
	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(1, 3))

	js, err := g.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	actual := strings.TrimSpace(string(js))
	expected := strings.TrimSpace(testGraphJSONBasicStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

// record some graph transformations, and make sure we get the same graph when
// they're replayed
func TestGraphJSON_basicRecord(t *testing.T) {
	var g Graph
	var buf bytes.Buffer
	g.SetDebugWriter(&buf)

	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(1, 2))
	g.Connect(BasicEdge(1, 3))
	g.Connect(BasicEdge(2, 3))
	(&AcyclicGraph{g}).TransitiveReduction()

	recorded := buf.Bytes()
	// the Walk doesn't happen in a determined order, so just count operations
	// for now to make sure we wrote stuff out.
	if len(bytes.Split(recorded, []byte{'\n'})) != 17 {
		t.Fatalf("bad: %s", recorded)
	}

	original, err := g.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	// replay the logs, and marshal the graph back out again
	m, err := decodeGraph(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}

	replayed, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(original, replayed) {
		t.Fatalf("\noriginal: %s\nreplayed: %s", original, replayed)
	}
}

// Verify that Vertex and Edge annotations appear in the debug output
func TestGraphJSON_debugInfo(t *testing.T) {
	var g Graph
	var buf bytes.Buffer
	g.SetDebugWriter(&buf)

	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(BasicEdge(1, 2))

	g.DebugVertexInfo(2, "2")
	g.DebugVertexInfo(3, "3")
	g.DebugEdgeInfo(BasicEdge(1, 2), "1|2")

	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))

	var found2, found3, foundEdge bool
	for dec.More() {
		var d streamDecode

		err := dec.Decode(&d)
		if err != nil {
			t.Fatal(err)
		}

		switch d.Type {
		case typeVertexInfo:
			va := &marshalVertexInfo{}
			err := json.Unmarshal(d.JSON, va)
			if err != nil {
				t.Fatal(err)
			}

			switch va.Info {
			case "2":
				if va.Vertex.Name != "2" {
					t.Fatalf("wrong vertex annotated 2: %#v", va)
				}
				found2 = true
			case "3":
				if va.Vertex.Name != "3" {
					t.Fatalf("wrong vertex annotated 3: %#v", va)
				}
				found3 = true
			default:
				t.Fatalf("unexpected annotation: %#v", va)
			}
		case typeEdgeInfo:
			ea := &marshalEdgeInfo{}
			err := json.Unmarshal(d.JSON, ea)
			if err != nil {
				t.Fatal(err)
			}

			switch ea.Info {
			case "1|2":
				if ea.Edge.Name != "1|2" {
					t.Fatalf("incorrect edge annotation: %#v\n", ea)
				}
				foundEdge = true
			default:
				t.Fatalf("unexpected edge Info: %#v", ea)
			}
		}
	}

	if !found2 {
		t.Fatal("annotation 2 not found")
	}
	if !found3 {
		t.Fatal("annotation 3 not found")
	}
	if !foundEdge {
		t.Fatal("edge annotation not found")
	}
}

// Verify that debug operations appear in the debug output
func TestGraphJSON_debugOperations(t *testing.T) {
	var g Graph
	var buf bytes.Buffer
	g.SetDebugWriter(&buf)

	debugOp := g.DebugOperation("AddOne", "adding node 1")
	g.Add(1)
	debugOp.End("done adding node 1")

	// use an immediate closure to test defers
	func() {
		defer g.DebugOperation("AddTwo", "adding nodes 2 and 3").End("done adding 2 and 3")
		g.Add(2)
		defer g.DebugOperation("NestedAddThree", "second defer").End("done adding node 3")
		g.Add(3)
	}()

	g.Connect(BasicEdge(1, 2))

	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))

	var ops []string
	for dec.More() {
		var d streamDecode

		err := dec.Decode(&d)
		if err != nil {
			t.Fatal(err)
		}

		if d.Type != typeOperation {
			continue
		}

		o := &marshalOperation{}
		err = json.Unmarshal(d.JSON, o)
		if err != nil {
			t.Fatal(err)
		}

		switch {
		case o.Begin == "AddOne":
			ops = append(ops, "BeginAddOne")
		case o.End == "AddOne":
			ops = append(ops, "EndAddOne")
		case o.Begin == "AddTwo":
			ops = append(ops, "BeginAddTwo")
		case o.End == "AddTwo":
			ops = append(ops, "EndAddTwo")
		case o.Begin == "NestedAddThree":
			ops = append(ops, "BeginAddThree")
		case o.End == "NestedAddThree":
			ops = append(ops, "EndAddThree")
		}
	}

	expectedOps := []string{
		"BeginAddOne",
		"EndAddOne",
		"BeginAddTwo",
		"BeginAddThree",
		"EndAddThree",
		"EndAddTwo",
	}

	if strings.Join(ops, ",") != strings.Join(expectedOps, ",") {
		t.Fatalf("incorrect order of operations: %v", ops)
	}
}

// Verify that we can replay visiting each vertex in order
func TestGraphJSON_debugVisits(t *testing.T) {
	var g Graph
	var buf bytes.Buffer
	g.SetDebugWriter(&buf)

	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Add(4)

	g.Connect(BasicEdge(2, 1))
	g.Connect(BasicEdge(4, 2))
	g.Connect(BasicEdge(3, 4))

	err := (&AcyclicGraph{g}).Walk(func(v Vertex) error {
		g.DebugVisitInfo(v, "basic walk")
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}

	var visited []string

	dec := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	for dec.More() {
		var d streamDecode

		err := dec.Decode(&d)
		if err != nil {
			t.Fatal(err)
		}

		if d.Type != typeVisitInfo {
			continue
		}

		o := &marshalVertexInfo{}
		err = json.Unmarshal(d.JSON, o)
		if err != nil {
			t.Fatal(err)
		}

		visited = append(visited, o.Vertex.ID)
	}

	expected := []string{"1", "2", "4", "3"}

	if strings.Join(visited, "-") != strings.Join(expected, "-") {
		t.Fatalf("incorrect order of operations: %v", visited)
	}
}

const testGraphJSONEmptyStr = `{
  "Type": "Graph",
  "Name": "root",
  "Vertices": [
    {
      "ID": "1",
      "Name": "1"
    },
    {
      "ID": "2",
      "Name": "2"
    },
    {
      "ID": "3",
      "Name": "3"
    }
  ]
}`

const testGraphJSONBasicStr = `{
  "Type": "Graph",
  "Name": "root",
  "Vertices": [
    {
      "ID": "1",
      "Name": "1"
    },
    {
      "ID": "2",
      "Name": "2"
    },
    {
      "ID": "3",
      "Name": "3"
    }
  ],
  "Edges": [
    {
      "Name": "1|3",
      "Source": "1",
      "Target": "3"
    }
  ]
}`
