package terraform

import (
	"fmt"
	"log"
	"strings"

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
	// ReferenceableName is the name by which this can be referenced.
	// This can be either just the type, or include the field. Example:
	// "aws_instance.bar" or "aws_instance.bar.id".
	ReferenceableName() []string
}

// GraphNodeReferencer must be implemented by nodes that reference other
// Terraform items and therefore depend on them.
type GraphNodeReferencer interface {
	// References are the list of things that this node references. This
	// can include fields or just the type, just like GraphNodeReferenceable
	// above.
	References() []string
}

// GraphNodeReferenceGlobal is an interface that can optionally be
// implemented. If ReferenceGlobal returns true, then the References()
// and ReferenceableName() must be _fully qualified_ with "module.foo.bar"
// etc.
//
// This allows a node to reference and be referenced by a specific name
// that may cross module boundaries. This can be very dangerous so use
// this wisely.
//
// The primary use case for this is module boundaries (variables coming in).
type GraphNodeReferenceGlobal interface {
	// Set to true to signal that references and name are fully
	// qualified. See the above docs for more information.
	ReferenceGlobal() bool
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
// for nodes that depend on an Output or Local value. Output and local nodes are
// removed during destroy, so anything which depends on them must be evaluated
// first. These can't be interpolated during destroy, so the stored value must
// be used anyway hence they don't need to be re-evaluated.
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

		// reverse any incoming edges so that the value is removed last
		for _, e := range g.EdgesTo(v) {
			source := e.Source()
			log.Printf("[TRACE] output dep: %s", dag.VertexName(source))

			g.RemoveEdge(e)
			g.Connect(&DestroyEdge{S: v, T: source})
		}
	}

	return nil
}

// ReferenceMap is a structure that can be used to efficiently check
// for references on a graph.
type ReferenceMap struct {
	// m is the mapping of referenceable name to list of verticies that
	// implement that name. This is built on initialization.
	references   map[string][]dag.Vertex
	referencedBy map[string][]dag.Vertex
}

// References returns the list of vertices that this vertex
// references along with any missing references.
func (m *ReferenceMap) References(v dag.Vertex) ([]dag.Vertex, []string) {
	rn, ok := v.(GraphNodeReferencer)
	if !ok {
		return nil, nil
	}

	var matches []dag.Vertex
	var missing []string
	prefix := m.prefix(v)
	for _, ns := range rn.References() {
		found := false
		for _, n := range strings.Split(ns, "/") {
			n = prefix + n
			parents, ok := m.references[n]
			if !ok {
				continue
			}

			// Mark that we found a match
			found = true

			// Make sure this isn't a self reference, which isn't included
			selfRef := false
			for _, p := range parents {
				if p == v {
					selfRef = true
					break
				}
			}
			if selfRef {
				continue
			}

			matches = append(matches, parents...)
			break
		}

		if !found {
			missing = append(missing, ns)
		}
	}

	return matches, missing
}

// ReferencedBy returns the list of vertices that reference the
// vertex passed in.
func (m *ReferenceMap) ReferencedBy(v dag.Vertex) []dag.Vertex {
	rn, ok := v.(GraphNodeReferenceable)
	if !ok {
		return nil
	}

	var matches []dag.Vertex
	prefix := m.prefix(v)
	for _, n := range rn.ReferenceableName() {
		n = prefix + n
		children, ok := m.referencedBy[n]
		if !ok {
			continue
		}

		// Make sure this isn't a self reference, which isn't included
		selfRef := false
		for _, p := range children {
			if p == v {
				selfRef = true
				break
			}
		}
		if selfRef {
			continue
		}

		matches = append(matches, children...)
	}

	return matches
}

func (m *ReferenceMap) prefix(v dag.Vertex) string {
	// If the node is stating it is already fully qualified then
	// we don't have to create the prefix!
	if gn, ok := v.(GraphNodeReferenceGlobal); ok && gn.ReferenceGlobal() {
		return ""
	}

	// Create the prefix based on the path
	var prefix string
	if pn, ok := v.(GraphNodeSubPath); ok {
		if path := normalizeModulePath(pn.Path()); len(path) > 1 {
			prefix = modulePrefixStr(path) + "."
		}
	}

	return prefix
}

// NewReferenceMap is used to create a new reference map for the
// given set of vertices.
func NewReferenceMap(vs []dag.Vertex) *ReferenceMap {
	var m ReferenceMap

	// Build the lookup table
	refMap := make(map[string][]dag.Vertex)
	for _, v := range vs {
		// We're only looking for referenceable nodes
		rn, ok := v.(GraphNodeReferenceable)
		if !ok {
			continue
		}

		// Go through and cache them
		prefix := m.prefix(v)
		for _, n := range rn.ReferenceableName() {
			n = prefix + n
			refMap[n] = append(refMap[n], v)
		}

		// If there is a path, it is always referenceable by that. For
		// example, if this is a referenceable thing at path []string{"foo"},
		// then it can be referenced at "module.foo"
		if pn, ok := v.(GraphNodeSubPath); ok {
			for _, p := range ReferenceModulePath(pn.Path()) {
				refMap[p] = append(refMap[p], v)
			}
		}
	}

	// Build the lookup table for referenced by
	refByMap := make(map[string][]dag.Vertex)
	for _, v := range vs {
		// We're only looking for referenceable nodes
		rn, ok := v.(GraphNodeReferencer)
		if !ok {
			continue
		}

		// Go through and cache them
		prefix := m.prefix(v)
		for _, n := range rn.References() {
			n = prefix + n
			refByMap[n] = append(refByMap[n], v)
		}
	}

	m.references = refMap
	m.referencedBy = refByMap
	return &m
}

// Returns the reference name for a module path. The path "foo" would return
// "module.foo". If this is a deeply nested module, it will be every parent
// as well. For example: ["foo", "bar"] would return both "module.foo" and
// "module.foo.module.bar"
func ReferenceModulePath(p []string) []string {
	p = normalizeModulePath(p)
	if len(p) == 1 {
		// Root, no name
		return nil
	}

	result := make([]string, 0, len(p)-1)
	for i := len(p); i > 1; i-- {
		result = append(result, modulePrefixStr(p[:i]))
	}

	return result
}

// ReferencesFromConfig returns the references that a configuration has
// based on the interpolated variables in a configuration.
func ReferencesFromConfig(c *config.RawConfig) []string {
	var result []string
	for _, v := range c.Variables {
		if r := ReferenceFromInterpolatedVar(v); len(r) > 0 {
			result = append(result, r...)
		}
	}

	return result
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

func modulePrefixStr(p []string) string {
	parts := make([]string, 0, len(p)*2)
	for _, p := range p[1:] {
		parts = append(parts, "module", p)
	}

	return strings.Join(parts, ".")
}

func modulePrefixList(result []string, prefix string) []string {
	if prefix != "" {
		for i, v := range result {
			result[i] = fmt.Sprintf("%s.%s", prefix, v)
		}
	}

	return result
}
