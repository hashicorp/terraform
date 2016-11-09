package terraform

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/terraform/dag"
	"github.com/mitchellh/copystructure"
)

// The NodeDebug method outputs debug information to annotate the graphs
// stored in the DebugInfo
type GraphNodeDebugger interface {
	NodeDebug() string
}

type GraphNodeDebugOrigin interface {
	DotOrigin() bool
}
type DebugGraph struct {
	sync.Mutex
	Name string

	ord int
	buf bytes.Buffer

	Graph *Graph

	dotOpts *dag.DotOpts
}

// DebugGraph holds a dot representation of the Terraform graph, and can be
// written out to the DebugInfo log with DebugInfo.WriteGraph. A DebugGraph can
// log data to it's internal buffer via the Printf and Write methods, which
// will be also be written out to the DebugInfo archive.
func NewDebugGraph(name string, g *Graph, opts *dag.DotOpts) (*DebugGraph, error) {
	dg := &DebugGraph{
		Name:    name,
		Graph:   g,
		dotOpts: opts,
	}

	dbug.WriteFile(dg.Name, g.Dot(opts))
	return dg, nil
}

// Printf to the internal buffer
func (dg *DebugGraph) Printf(f string, args ...interface{}) (int, error) {
	if dg == nil {
		return 0, nil
	}
	dg.Lock()
	defer dg.Unlock()
	return fmt.Fprintf(&dg.buf, f, args...)
}

// Write to the internal buffer
func (dg *DebugGraph) Write(b []byte) (int, error) {
	if dg == nil {
		return 0, nil
	}
	dg.Lock()
	defer dg.Unlock()
	return dg.buf.Write(b)
}

func (dg *DebugGraph) LogBytes() []byte {
	if dg == nil {
		return nil
	}
	dg.Lock()
	defer dg.Unlock()
	return dg.buf.Bytes()
}

func (dg *DebugGraph) DotBytes() []byte {
	if dg == nil {
		return nil
	}
	dg.Lock()
	defer dg.Unlock()
	return dg.Graph.Dot(dg.dotOpts)
}

func (dg *DebugGraph) DebugNode(v interface{}) {
	if dg == nil {
		return
	}
	dg.Lock()
	defer dg.Unlock()

	// record the ordinal value for each node
	ord := dg.ord
	dg.ord++

	name := dag.VertexName(v)
	vCopy, _ := copystructure.Config{Lock: true}.Copy(v)

	// record as much of the node data structure as we can
	spew.Fdump(&dg.buf, vCopy)

	dg.buf.WriteString(fmt.Sprintf("%d visited %s\n", ord, name))

	// if the node provides debug output, insert it into the graph, and log it
	if nd, ok := v.(GraphNodeDebugger); ok {
		out := nd.NodeDebug()
		dg.buf.WriteString(fmt.Sprintf("NodeDebug (%s):'%s'\n", name, out))
	}
}
