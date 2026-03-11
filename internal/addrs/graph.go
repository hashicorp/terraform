// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/internal/dag"
)

// DirectedGraph represents a directed graph whose nodes are addresses of
// type T.
//
// This graph type supports directed edges between pairs of addresses, and
// because Terraform most commonly uses graphs to represent dependency
// relationships it uses "dependency" and "dependent" as the names of the
// endpoints of an edge, even though technically this data structure could
// be used to represent other kinds of directed relationships if needed.
// When used as an operation dependency graph, the "dependency" must be visited
// before the "dependent".
//
// This data structure is not concurrency-safe for writes and so callers must
// supply suitable synchronization primitives if modifying a graph concurrently
// with readers or other writers. Concurrent reads of an already-constructed
// graph are safe.
type DirectedGraph[T UniqueKeyer] struct {
	// Our dag.AcyclicGraph implementation is a little quirky but also
	// well-tested and stable, so we'll use that for the underlying
	// graph operations and just wrap a slightly nicer address-oriented
	// API around it.
	// Reusing this does mean that some of our operations end up allocating
	// more than they would need to otherwise, so perhaps we'll revisit this
	// in future if it seems to be causing performance problems.
	g *dag.AcyclicGraph

	// dag.AcyclicGraph can only support node types that are either
	// comparable using == or that implement a special "hashable"
	// interface, so we'll use our UniqueKeyer technique to produce
	// suitable node values but we need a sidecar structure to remember
	// which real address value belongs to each node value.
	nodes map[UniqueKey]T
}

func NewDirectedGraph[T UniqueKeyer]() DirectedGraph[T] {
	return DirectedGraph[T]{
		g:     &dag.AcyclicGraph{},
		nodes: map[UniqueKey]T{},
	}
}

func (g DirectedGraph[T]) Add(addr T) {
	k := addr.UniqueKey()
	g.nodes[k] = addr
	g.g.Add(k)
}

func (g DirectedGraph[T]) Has(addr T) bool {
	k := addr.UniqueKey()
	_, ok := g.nodes[k]
	return ok
}

func (g DirectedGraph[T]) Remove(addr T) {
	k := addr.UniqueKey()
	g.g.Remove(k)
	delete(g.nodes, k)
}

func (g DirectedGraph[T]) AllNodes() Set[T] {
	ret := make(Set[T], len(g.nodes))
	for _, addr := range g.nodes {
		ret.Add(addr)
	}
	return ret
}

// AddDependency records that the first address depends on the second.
//
// If either address is not already in the graph then it will be implicitly
// added as part of this operation.
func (g DirectedGraph[T]) AddDependency(dependent, dependency T) {
	g.Add(dependent)
	g.Add(dependency)
	g.g.Connect(dag.BasicEdge(dependent.UniqueKey(), dependency.UniqueKey()))
}

// DirectDependenciesOf returns only the direct dependencies of the given
// dependent address.
func (g DirectedGraph[T]) DirectDependenciesOf(addr T) Set[T] {
	k := addr.UniqueKey()
	ret := MakeSet[T]()
	raw := g.g.DownEdges(k)
	for otherKI := range raw {
		ret.Add(g.nodes[otherKI.(UniqueKey)])
	}
	return ret
}

// TransitiveDependenciesOf returns both direct and indirect dependencies of the
// given dependent address.
//
// This operation is valid only for an acyclic graph, and will panic if
// the graph contains cycles.
func (g DirectedGraph[T]) TransitiveDependenciesOf(addr T) Set[T] {
	k := addr.UniqueKey()
	ret := MakeSet[T]()
	for otherKI := range g.g.Ancestors(k) {
		ret.Add(g.nodes[otherKI.(UniqueKey)])
	}
	return ret
}

// DirectDependentsOf returns only the direct dependents of the given
// dependency address.
func (g DirectedGraph[T]) DirectDependentsOf(addr T) Set[T] {
	k := addr.UniqueKey()
	ret := MakeSet[T]()
	raw := g.g.UpEdges(k)
	for otherKI := range raw {
		ret.Add(g.nodes[otherKI.(UniqueKey)])
	}
	return ret
}

// TransitiveDependentsOf returns both direct and indirect dependents of the
// given dependency address.
//
// This operation is valid only for an acyclic graph, and will panic if
// the graph contains cycles.
func (g DirectedGraph[T]) TransitiveDependentsOf(addr T) Set[T] {
	k := addr.UniqueKey()
	ret := MakeSet[T]()
	for otherKI := range g.g.Descendants(k) {
		ret.Add(g.nodes[otherKI.(UniqueKey)])
	}
	return ret
}

// TopologicalOrder returns one possible topological sort of the addresses
// in the graph.
//
// There are often multiple possible sort orders that preserve the required
// dependency ordering. The exact order returned by this function is undefined
// and may vary between calls against the same graph.
func (g DirectedGraph[T]) TopologicalOrder() []T {
	raw := g.g.TopologicalOrder()
	if len(raw) == 0 {
		return nil
	}
	ret := make([]T, len(raw))
	for i, k := range raw {
		ret[i] = g.nodes[k.(UniqueKey)]
	}
	return ret
}

// StringForComparison outputs a string representing the topology of the
// graph in a form intended for convenient equality testing by string comparison.
//
// For best results all possible dynamic types of T should implement
// fmt.Stringer.
func (g DirectedGraph[T]) StringForComparison() string {
	var buf bytes.Buffer

	stringRepr := func(v any) string {
		switch v := v.(type) {
		case fmt.Stringer:
			return v.String()
		default:
			return fmt.Sprintf("%#v", v)
		}
	}

	// We want the addresses in a consistent order but it doesn't really
	// matter what that order is, so we'll just do it lexically by each
	// type's standard string representation.
	nodeKeys := make([]UniqueKey, 0, len(g.nodes))
	for k := range g.nodes {
		nodeKeys = append(nodeKeys, k)
	}
	sort.Slice(nodeKeys, func(i, j int) bool {
		iStr := stringRepr(g.nodes[nodeKeys[i]])
		jStr := stringRepr(g.nodes[nodeKeys[j]])
		return iStr < jStr
	})

	for _, k := range nodeKeys {
		addr := g.nodes[k]
		fmt.Fprintf(&buf, "%s\n", stringRepr(addr))

		deps := g.DirectDependenciesOf(addr)
		if len(deps) == 0 {
			continue
		}

		depKeys := make([]UniqueKey, 0, len(deps))
		for k := range deps {
			depKeys = append(depKeys, k)
		}
		sort.Slice(depKeys, func(i, j int) bool {
			iStr := stringRepr(g.nodes[depKeys[i]])
			jStr := stringRepr(g.nodes[depKeys[j]])
			return iStr < jStr
		})
		for _, k := range depKeys {
			fmt.Fprintf(&buf, "  %s\n", stringRepr(g.nodes[k]))
		}
	}

	return buf.String()
}
