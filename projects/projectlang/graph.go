package projectlang

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/projects/projectconfigs"
)

type graph struct {
	g     dag.AcyclicGraph
	nodes map[addrs.ProjectReferenceable]graphNode
}

func newGraph() *graph {
	return &graph{
		g:     dag.AcyclicGraph{},
		nodes: make(map[addrs.ProjectReferenceable]graphNode),
	}
}

func (g *graph) Add(node graphNode) {
	addr := node.ReferenceableAddr()
	if _, exists := g.nodes[addr]; exists {
		return
	}
	g.nodes[addr] = node
	g.Add(node)
}

type graphNode interface {
	ReferenceableAddr() addrs.ProjectReferenceable
	References() []*addrs.ProjectConfigReference
}

var _ graphNode = (*graphNodeContextValue)(nil)
var _ graphNode = (*graphNodeLocalValue)(nil)
var _ graphNode = (*graphNodeWorkspace)(nil)
var _ graphNode = (*graphNodeUpstreamWorkspace)(nil)

type graphNodeContextValue struct {
	Addr   addrs.ProjectContextValue
	Config *projectconfigs.ContextValue
}

func (n *graphNodeContextValue) ReferenceableAddr() addrs.ProjectReferenceable {
	return n.Addr
}

func (n *graphNodeContextValue) References() []*addrs.ProjectConfigReference {
	return nil // context values can never refer to anything
}

type graphNodeLocalValue struct {
	Addr   addrs.LocalValue
	Config *projectconfigs.LocalValue
}

func (n *graphNodeLocalValue) ReferenceableAddr() addrs.ProjectReferenceable {
	return n.Addr
}

func (n *graphNodeLocalValue) References() []*addrs.ProjectConfigReference {
	return findReferencesInExpr(n.Config.Value)
}

type graphNodeWorkspace struct {
	Addr   addrs.ProjectWorkspace
	Config *projectconfigs.Workspace
}

func (n *graphNodeWorkspace) ReferenceableAddr() addrs.ProjectReferenceable {
	return n.Addr
}

func (n *graphNodeWorkspace) References() []*addrs.ProjectConfigReference {
	var ret []*addrs.ProjectConfigReference
	// NOTE: for_each is not included here because we assume it gets resolved
	// in an earlier step that instantiates all of the workspace objects.
	ret = append(ret, findReferencesInExpr(n.Config.Variables)...)
	ret = append(ret, findReferencesInExpr(n.Config.ConfigSource)...)
	ret = append(ret, findReferencesInExpr(n.Config.Remote)...)
	// TODO: What about the state storage? Seems like we need something at
	// a higher level of abstraction than the raw configuration in here.
	return ret
}

type graphNodeUpstreamWorkspace struct {
	Addr   addrs.ProjectUpstreamWorkspace
	Config *projectconfigs.Upstream
}

func (n *graphNodeUpstreamWorkspace) ReferenceableAddr() addrs.ProjectReferenceable {
	return n.Addr
}

func (n *graphNodeUpstreamWorkspace) References() []*addrs.ProjectConfigReference {
	var ret []*addrs.ProjectConfigReference
	ret = append(ret, findReferencesInExpr(n.Config.Remote)...)
	return ret
}

func buildEvalGraph(needRefs []*addrs.ProjectConfigReference, config *projectconfigs.Config) *dag.AcyclicGraph {
	graph := &dag.AcyclicGraph{}

	return graph
}

func addGraphNodeForAddr(graph *graph, addr addrs.ProjectReferenceable, config *projectconfigs.Config) {
	switch addr := addr.(type) {
	case addrs.ProjectContextValue:
		c, ok := config.Context[addr.Name]
		if !ok {
			panic(fmt.Sprintf("reference to undeclared context value %q", addr.Name))
		}
		node := &graphNodeContextValue{
			Addr:   addr,
			Config: c,
		}
		graph.Add(node)
		return
	case addrs.LocalValue:
		c, ok := config.Locals[addr.Name]
		if !ok {
			panic(fmt.Sprintf("reference to undeclared local value %q", addr.Name))
		}
		node := &graphNodeLocalValue{
			Addr:   addr,
			Config: c,
		}
		graph.Add(node)
		return
	case addrs.ProjectWorkspace:
		c, ok := config.Workspaces[addr.Name]
		if !ok {
			panic(fmt.Sprintf("reference to undeclared workspace configuration %q", addr.Name))
		}
		node := &graphNodeWorkspace{
			Addr:   addr,
			Config: c,
		}
		graph.Add(node)
		return
	case addrs.ProjectUpstreamWorkspace:
		c, ok := config.Upstreams[addr.Name]
		if !ok {
			panic(fmt.Sprintf("reference to undeclared upstream workspace configuration %q", addr.Name))
		}
		node := &graphNodeUpstreamWorkspace{
			Addr:   addr,
			Config: c,
		}
		graph.Add(node)
		return
	case addrs.ForEachAttr:
		// These don't need a graph node because they refer to data
		// from the object they are referenced within.
		return
	}

	// Should not get here; the above cases should be comprehensive for
	// all available implementations of addrs.ProjectReferenceable.
	panic(fmt.Sprintf("don't know how to construct a graph node for %s", addr))
}

func addGraphNodesForRefs(graph *graph, needRefs []*addrs.ProjectConfigReference, config *projectconfigs.Config) {
	// We'll panic on anything that refers to an object not declared
	// in the configuration because we assume the caller already checked
	// that before calling us.
	for _, ref := range needRefs {
		addGraphNodeForAddr(graph, ref.Subject, config)
	}
}
