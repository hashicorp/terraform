package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/lang"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeReferenceable must be implemented by any node that represents
// a Terraform thing that can be referenced (resource, module, etc.).
//
// Even if the thing has no name, this should return an empty list. By
// implementing this and returning a non-nil result, you say that this CAN
// be referenced and other methods of referencing may still be possible (such
// as by path!)
type GraphNodeReferenceable interface {
	GraphNodeSubPath

	// ReferenceableAddrs returns a list of addresses through which this can be
	// referenced.
	ReferenceableAddrs() []addrs.Referenceable
}

// GraphNodeReferencer must be implemented by nodes that reference other
// Terraform items and therefore depend on them.
type GraphNodeReferencer interface {
	GraphNodeSubPath

	// References returns a list of references made by this node, which
	// include both a referenced address and source location information for
	// the reference.
	References() []*addrs.Reference
}

// GraphNodeReferenceOutside is an interface that can optionally be implemented.
// A node that implements it can specify that its own referenceable addresses
// and/or the addresses it references are in a different module than the
// node itself.
//
// Any referenceable addresses returned by ReferenceableAddrs are interpreted
// relative to the returned selfPath.
//
// Any references returned by References are interpreted relative to the
// returned referencePath.
//
// It is valid but not required for either of these paths to match what is
// returned by method Path, though if both match the main Path then there
// is no reason to implement this method.
//
// The primary use-case for this is the nodes representing module input
// variables, since their expressions are resolved in terms of their calling
// module, but they are still referenced from their own module.
type GraphNodeReferenceOutside interface {
	// ReferenceOutside returns a path in which any references from this node
	// are resolved.
	ReferenceOutside() (selfPath, referencePath addrs.ModuleInstance)
}

// ReferenceTransformer is a GraphTransformer that connects all the
// nodes that reference each other in order to form the proper ordering.
type ReferenceTransformer struct{}

func (t *ReferenceTransformer) Transform(g *Graph) error {
	// Build a reference map so we can efficiently look up the references
	vs := g.Vertices()
	m := NewReferenceMap(vs)

	// Find the things that reference things and connect them
	for _, v := range vs {
		parents, _ := m.References(v)
		parentsDbg := make([]string, len(parents))
		for i, v := range parents {
			parentsDbg[i] = dag.VertexName(v)
		}
		log.Printf(
			"[DEBUG] ReferenceTransformer: %q references: %v",
			dag.VertexName(v), parentsDbg)

		for _, parent := range parents {
			g.Connect(dag.BasicEdge(v, parent))
		}
	}

	return nil
}

// DestroyReferenceTransformer is a GraphTransformer that reverses the edges
// for locals and outputs that depend on other nodes which will be
// removed during destroy. If a destroy node is evaluated before the local or
// output value, it will be removed from the state, and the later interpolation
// will fail.
type DestroyValueReferenceTransformer struct{}

func (t *DestroyValueReferenceTransformer) Transform(g *Graph) error {
	vs := g.Vertices()
	for _, v := range vs {
		switch v.(type) {
		case *NodeApplyableOutput, *NodeLocal:
			// OK
		default:
			continue
		}

		// reverse any outgoing edges so that the value is evaluated first.
		for _, e := range g.EdgesFrom(v) {
			target := e.Target()

			// only destroy nodes will be evaluated in reverse
			if _, ok := target.(GraphNodeDestroyer); !ok {
				continue
			}

			log.Printf("[TRACE] output dep: %s", dag.VertexName(target))

			g.RemoveEdge(e)
			g.Connect(&DestroyEdge{S: target, T: v})
		}
	}

	return nil
}

// PruneUnusedValuesTransformer is s GraphTransformer that removes local and
// output values which are not referenced in the graph. Since outputs and
// locals always need to be evaluated, if they reference a resource that is not
// available in the state the interpolation could fail.
type PruneUnusedValuesTransformer struct{}

func (t *PruneUnusedValuesTransformer) Transform(g *Graph) error {
	// this might need multiple runs in order to ensure that pruning a value
	// doesn't effect a previously checked value.
	for removed := 0; ; removed = 0 {
		for _, v := range g.Vertices() {
			switch v.(type) {
			case *NodeApplyableOutput, *NodeLocal:
				// OK
			default:
				continue
			}

			dependants := g.UpEdges(v)

			switch dependants.Len() {
			case 0:
				// nothing at all depends on this
				g.Remove(v)
				removed++
			case 1:
				// because an output's destroy node always depends on the output,
				// we need to check for the case of a single destroy node.
				d := dependants.List()[0]
				if _, ok := d.(*NodeDestroyableOutput); ok {
					g.Remove(v)
					removed++
				}
			}
		}
		if removed == 0 {
			break
		}
	}

	return nil
}

// ReferenceMap is a structure that can be used to efficiently check
// for references on a graph.
type ReferenceMap struct {
	// vertices is a map from internal reference keys (as produced by the
	// mapKey method) to one or more vertices that are identified by each key.
	//
	// A particular reference key might actually identify multiple vertices,
	// e.g. in situations where one object is contained inside another.
	vertices map[string][]dag.Vertex

	// edges is a map whose keys are a subset of the internal reference keys
	// from "vertices", and whose values are the nodes that refer to each
	// key. The values in this map are the referrers, while values in
	// "verticies" are the referents. The keys in both cases are referents.
	edges map[string][]dag.Vertex
}

// References returns the set of vertices that the given vertex refers to,
// and any referenced addresses that do not have corresponding vertices.
func (m *ReferenceMap) References(v dag.Vertex) ([]dag.Vertex, []addrs.Referenceable) {
	rn, ok := v.(GraphNodeReferencer)
	if !ok {
		return nil, nil
	}
	if _, ok := v.(GraphNodeSubPath); !ok {
		return nil, nil
	}

	var matches []dag.Vertex
	var missing []addrs.Referenceable

	for _, ref := range rn.References() {
		subject := ref.Subject

		key := m.referenceMapKey(v, subject)
		if _, exists := m.vertices[key]; !exists {
			// If what we were looking for was a ResourceInstance then we
			// might be in a resource-oriented graph rather than an
			// instance-oriented graph, and so we'll see if we have the
			// resource itself instead.
			switch ri := subject.(type) {
			case addrs.ResourceInstance:
				subject = ri.ContainingResource()
			case addrs.ResourceInstancePhase:
				subject = ri.ContainingResource()
			}
			key = m.referenceMapKey(v, subject)
		}

		vertices := m.vertices[key]
		for _, rv := range vertices {
			// don't include self-references
			if rv == v {
				continue
			}
			matches = append(matches, rv)
		}
		if len(vertices) == 0 {
			missing = append(missing, ref.Subject)
		}
	}

	return matches, missing
}

// Referrers returns the set of vertices that refer to the given vertex.
func (m *ReferenceMap) Referrers(v dag.Vertex) []dag.Vertex {
	rn, ok := v.(GraphNodeReferenceable)
	if !ok {
		return nil
	}
	sp, ok := v.(GraphNodeSubPath)
	if !ok {
		return nil
	}

	var matches []dag.Vertex
	for _, addr := range rn.ReferenceableAddrs() {
		key := m.mapKey(sp.Path(), addr)
		referrers, ok := m.edges[key]
		if !ok {
			continue
		}

		// If the referrer set includes our own given vertex then we skip,
		// since we don't want to return self-references.
		selfRef := false
		for _, p := range referrers {
			if p == v {
				selfRef = true
				break
			}
		}
		if selfRef {
			continue
		}

		matches = append(matches, referrers...)
	}

	return matches
}

func (m *ReferenceMap) mapKey(path addrs.ModuleInstance, addr addrs.Referenceable) string {
	return fmt.Sprintf("%s|%s", path.String(), addr.String())
}

// vertexReferenceablePath returns the path in which the given vertex can be
// referenced. This is the path that its results from ReferenceableAddrs
// are considered to be relative to.
//
// Only GraphNodeSubPath implementations can be referenced, so this method will
// panic if the given vertex does not implement that interface.
func (m *ReferenceMap) vertexReferenceablePath(v dag.Vertex) addrs.ModuleInstance {
	sp, ok := v.(GraphNodeSubPath)
	if !ok {
		// Only nodes with paths can participate in a reference map.
		panic(fmt.Errorf("vertexMapKey on vertex type %T which doesn't implement GraphNodeSubPath", sp))
	}

	if outside, ok := v.(GraphNodeReferenceOutside); ok {
		// Vertex is referenced from a different module than where it was
		// declared.
		path, _ := outside.ReferenceOutside()
		return path
	}

	// Vertex is referenced from the same module as where it was declared.
	return sp.Path()
}

// vertexReferencePath returns the path in which references _from_ the given
// vertex must be interpreted.
//
// Only GraphNodeSubPath implementations can have references, so this method
// will panic if the given vertex does not implement that interface.
func vertexReferencePath(referrer dag.Vertex) addrs.ModuleInstance {
	sp, ok := referrer.(GraphNodeSubPath)
	if !ok {
		// Only nodes with paths can participate in a reference map.
		panic(fmt.Errorf("vertexReferencePath on vertex type %T which doesn't implement GraphNodeSubPath", sp))
	}

	var path addrs.ModuleInstance
	if outside, ok := referrer.(GraphNodeReferenceOutside); ok {
		// Vertex makes references to objects in a different module than where
		// it was declared.
		_, path = outside.ReferenceOutside()
		return path
	}

	// Vertex makes references to objects in the same module as where it
	// was declared.
	return sp.Path()
}

// referenceMapKey produces keys for the "edges" map. "referrer" is the vertex
// that the reference is from, and "addr" is the address of the object being
// referenced.
//
// The result is an opaque string that includes both the address of the given
// object and the address of the module instance that object belongs to.
//
// Only GraphNodeSubPath implementations can be referrers, so this method will
// panic if the given vertex does not implement that interface.
func (m *ReferenceMap) referenceMapKey(referrer dag.Vertex, addr addrs.Referenceable) string {
	path := vertexReferencePath(referrer)
	return m.mapKey(path, addr)
}

// NewReferenceMap is used to create a new reference map for the
// given set of vertices.
func NewReferenceMap(vs []dag.Vertex) *ReferenceMap {
	var m ReferenceMap

	// Build the lookup table
	vertices := make(map[string][]dag.Vertex)
	for _, v := range vs {
		_, ok := v.(GraphNodeSubPath)
		if !ok {
			// Only nodes with paths can participate in a reference map.
			continue
		}

		// We're only looking for referenceable nodes
		rn, ok := v.(GraphNodeReferenceable)
		if !ok {
			continue
		}

		path := m.vertexReferenceablePath(v)

		// Go through and cache them
		for _, addr := range rn.ReferenceableAddrs() {
			key := m.mapKey(path, addr)
			vertices[key] = append(vertices[key], v)
		}

		// Any node can be referenced by the address of the module it belongs
		// to or any of that module's ancestors.
		for _, addr := range path.Ancestors()[1:] {
			// Can be referenced either as the specific call instance (with
			// an instance key) or as the bare module call itself (the "module"
			// block in the parent module that created the instance).
			callPath, call := addr.Call()
			callInstPath, callInst := addr.CallInstance()
			callKey := m.mapKey(callPath, call)
			callInstKey := m.mapKey(callInstPath, callInst)
			vertices[callKey] = append(vertices[callKey], v)
			vertices[callInstKey] = append(vertices[callInstKey], v)
		}
	}

	// Build the lookup table for referenced by
	edges := make(map[string][]dag.Vertex)
	for _, v := range vs {
		_, ok := v.(GraphNodeSubPath)
		if !ok {
			// Only nodes with paths can participate in a reference map.
			continue
		}

		rn, ok := v.(GraphNodeReferencer)
		if !ok {
			// We're only looking for referenceable nodes
			continue
		}

		// Go through and cache them
		for _, ref := range rn.References() {
			if ref.Subject == nil {
				// Should never happen
				panic(fmt.Sprintf("%T.References returned reference with nil subject", rn))
			}
			key := m.referenceMapKey(v, ref.Subject)
			edges[key] = append(edges[key], v)
		}
	}

	m.vertices = vertices
	m.edges = edges
	return &m
}

// ReferencesFromConfig returns the references that a configuration has
// based on the interpolated variables in a configuration.
func ReferencesFromConfig(body hcl.Body, schema *configschema.Block) []*addrs.Reference {
	if body == nil {
		return nil
	}
	refs, _ := lang.ReferencesInBlock(body, schema)
	return refs
}

// ReferenceFromInterpolatedVar returns the reference from this variable,
// or an empty string if there is no reference.
func ReferenceFromInterpolatedVar(v config.InterpolatedVariable) []string {
	switch v := v.(type) {
	case *config.ModuleVariable:
		return []string{fmt.Sprintf("module.%s.output.%s", v.Name, v.Field)}
	case *config.ResourceVariable:
		id := v.ResourceId()

		// If we have a multi-reference (splat), then we depend on ALL
		// resources with this type/name.
		if v.Multi && v.Index == -1 {
			return []string{fmt.Sprintf("%s.*", id)}
		}

		// Otherwise, we depend on a specific index.
		idx := v.Index
		if !v.Multi || v.Index == -1 {
			idx = 0
		}

		// Depend on the index, as well as "N" which represents the
		// un-expanded set of resources.
		return []string{fmt.Sprintf("%s.%d/%s.N", id, idx, id)}
	case *config.UserVariable:
		return []string{fmt.Sprintf("var.%s", v.Name)}
	case *config.LocalVariable:
		return []string{fmt.Sprintf("local.%s", v.Name)}
	default:
		return nil
	}
}

// appendResourceDestroyReferences identifies resource and resource instance
// references in the given slice and appends to it the "destroy-phase"
// equivalents of those references, returning the result.
//
// This can be used in the References implementation for a node which must also
// depend on the destruction of anything it references.
func appendResourceDestroyReferences(refs []*addrs.Reference) []*addrs.Reference {
	given := refs
	for _, ref := range given {
		switch tr := ref.Subject.(type) {
		case addrs.Resource:
			newRef := *ref // shallow copy
			newRef.Subject = tr.Phase(addrs.ResourceInstancePhaseDestroy)
			refs = append(refs, &newRef)
		case addrs.ResourceInstance:
			newRef := *ref // shallow copy
			newRef.Subject = tr.Phase(addrs.ResourceInstancePhaseDestroy)
			refs = append(refs, &newRef)
		}
	}
	return refs
}

func modulePrefixStr(p addrs.ModuleInstance) string {
	return p.String()
}

func modulePrefixList(result []string, prefix string) []string {
	if prefix != "" {
		for i, v := range result {
			result[i] = fmt.Sprintf("%s.%s", prefix, v)
		}
	}

	return result
}
