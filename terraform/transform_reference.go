package terraform

import (
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/states"
)

// GraphNodeReferenceable must be implemented by any node that represents
// a Terraform thing that can be referenced (resource, module, etc.).
//
// Even if the thing has no name, this should return an empty list. By
// implementing this and returning a non-nil result, you say that this CAN
// be referenced and other methods of referencing may still be possible (such
// as by path!)
type GraphNodeReferenceable interface {
	GraphNodeModulePath

	// ReferenceableAddrs returns a list of addresses through which this can be
	// referenced.
	ReferenceableAddrs() []addrs.Referenceable
}

// GraphNodeReferencer must be implemented by nodes that reference other
// Terraform items and therefore depend on them.
type GraphNodeReferencer interface {
	GraphNodeModulePath

	// References returns a list of references made by this node, which
	// include both a referenced address and source location information for
	// the reference.
	References() []*addrs.Reference
}

type GraphNodeAttachDependencies interface {
	GraphNodeConfigResource
	AttachDependencies([]addrs.ConfigResource)
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
	ReferenceOutside() (selfPath, referencePath addrs.Module)
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
		if _, ok := v.(GraphNodeDestroyer); ok {
			// destroy nodes references are not connected, since they can only
			// use their own state.
			continue
		}
		parents := m.References(v)
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

		if len(parents) > 0 {
			continue
		}
	}

	return nil
}

// AttachDependenciesTransformer records all resource dependencies for each
// instance, and attaches the addresses to the node itself. Managed resource
// will record these in the state for proper ordering of destroy operations.
type AttachDependenciesTransformer struct {
	Config  *configs.Config
	State   *states.State
	Schemas *Schemas
}

func (t AttachDependenciesTransformer) Transform(g *Graph) error {
	// FIXME: this is only working with ResourceConfigAddr for now

	for _, v := range g.Vertices() {
		attacher, ok := v.(GraphNodeAttachDependencies)
		if !ok {
			continue
		}
		selfAddr := attacher.ResourceAddr()

		// Data sources don't need to track destroy dependencies
		if selfAddr.Resource.Mode == addrs.DataResourceMode {
			continue
		}

		ans, err := g.Ancestors(v)
		if err != nil {
			return err
		}

		// dedupe addrs when there's multiple instances involved, or
		// multiple paths in the un-reduced graph
		depMap := map[string]addrs.ConfigResource{}
		for _, d := range ans {
			var addr addrs.ConfigResource

			switch d := d.(type) {
			case GraphNodeResourceInstance:
				instAddr := d.ResourceInstanceAddr()
				addr = instAddr.ContainingResource().Config()
			case GraphNodeConfigResource:
				addr = d.ResourceAddr()
			default:
				continue
			}

			// Data sources don't need to track destroy dependencies
			if addr.Resource.Mode == addrs.DataResourceMode {
				continue
			}

			if addr.Equal(selfAddr) {
				continue
			}
			depMap[addr.String()] = addr
		}

		deps := make([]addrs.ConfigResource, 0, len(depMap))
		for _, d := range depMap {
			deps = append(deps, d)
		}
		sort.Slice(deps, func(i, j int) bool {
			return deps[i].String() < deps[j].String()
		})

		log.Printf("[TRACE] AttachDependenciesTransformer: %s depends on %s", attacher.ResourceAddr(), deps)
		attacher.AttachDependencies(deps)
	}

	return nil
}

// PruneUnusedValuesTransformer is a GraphTransformer that removes local,
// variable, and output values which are not referenced in the graph. If these
// values reference a resource that is no longer in the state the interpolation
// could fail.
type PruneUnusedValuesTransformer struct {
	Destroy bool
}

func (t *PruneUnusedValuesTransformer) Transform(g *Graph) error {
	// Pruning a value can effect previously checked edges, so loop until there
	// are no more changes.
	for removed := 0; ; removed = 0 {
		for _, v := range g.Vertices() {
			// we're only concerned with values that don't need to be saved in state
			switch v := v.(type) {
			case graphNodeTemporaryValue:
				if !v.temporaryValue() {
					continue
				}
			default:
				continue
			}

			dependants := g.UpEdges(v)

			// any referencers in the dependents means we need to keep this
			// value for evaluation
			removable := true
			for _, d := range dependants.List() {
				if _, ok := d.(GraphNodeReferencer); ok {
					removable = false
					break
				}
			}

			if removable {
				log.Printf("[TRACE] PruneUnusedValuesTransformer: removing unused value %s", dag.VertexName(v))
				g.Remove(v)
				removed++
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
}

// References returns the set of vertices that the given vertex refers to,
// and any referenced addresses that do not have corresponding vertices.
func (m *ReferenceMap) References(v dag.Vertex) []dag.Vertex {
	rn, ok := v.(GraphNodeReferencer)
	if !ok {
		return nil
	}

	var matches []dag.Vertex

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
			case addrs.AbsModuleCallOutput:
				subject = ri.ModuleCallOutput()
			default:
				log.Printf("[WARN] ReferenceTransformer: reference not found: %q", subject)
				continue
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
	}

	return matches
}

func (m *ReferenceMap) mapKey(path addrs.Module, addr addrs.Referenceable) string {
	return fmt.Sprintf("%s|%s", path.String(), addr.String())
}

// vertexReferenceablePath returns the path in which the given vertex can be
// referenced. This is the path that its results from ReferenceableAddrs
// are considered to be relative to.
//
// Only GraphNodeModulePath implementations can be referenced, so this method will
// panic if the given vertex does not implement that interface.
func vertexReferenceablePath(v dag.Vertex) addrs.Module {
	sp, ok := v.(GraphNodeModulePath)
	if !ok {
		// Only nodes with paths can participate in a reference map.
		panic(fmt.Errorf("vertexMapKey on vertex type %T which doesn't implement GraphNodeModulePath", sp))
	}

	if outside, ok := v.(GraphNodeReferenceOutside); ok {
		// Vertex is referenced from a different module than where it was
		// declared.
		path, _ := outside.ReferenceOutside()
		return path
	}

	// Vertex is referenced from the same module as where it was declared.
	return sp.ModulePath()
}

// vertexReferencePath returns the path in which references _from_ the given
// vertex must be interpreted.
//
// Only GraphNodeModulePath implementations can have references, so this method
// will panic if the given vertex does not implement that interface.
func vertexReferencePath(v dag.Vertex) addrs.Module {
	sp, ok := v.(GraphNodeModulePath)
	if !ok {
		// Only nodes with paths can participate in a reference map.
		panic(fmt.Errorf("vertexReferencePath on vertex type %T which doesn't implement GraphNodeModulePath", v))
	}

	if outside, ok := v.(GraphNodeReferenceOutside); ok {
		// Vertex makes references to objects in a different module than where
		// it was declared.
		_, path := outside.ReferenceOutside()
		return path
	}

	// Vertex makes references to objects in the same module as where it
	// was declared.
	return sp.ModulePath()
}

// referenceMapKey produces keys for the "edges" map. "referrer" is the vertex
// that the reference is from, and "addr" is the address of the object being
// referenced.
//
// The result is an opaque string that includes both the address of the given
// object and the address of the module instance that object belongs to.
//
// Only GraphNodeModulePath implementations can be referrers, so this method will
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
		// We're only looking for referenceable nodes
		rn, ok := v.(GraphNodeReferenceable)
		if !ok {
			continue
		}

		path := vertexReferenceablePath(v)

		// Go through and cache them
		for _, addr := range rn.ReferenceableAddrs() {
			key := m.mapKey(path, addr)
			vertices[key] = append(vertices[key], v)
		}
	}

	m.vertices = vertices
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
		// FIXME: Using this method in module expansion references,
		// May want to refactor this method beyond resources
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
