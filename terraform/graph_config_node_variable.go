package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// GraphNodeConfigVariable represents a Variable in the config.
type GraphNodeConfigVariable struct {
	Variable *config.Variable

	// Value, if non-nil, will be used to set the value of the variable
	// during evaluation. If this is nil, evaluation will do nothing.
	//
	// Module is the name of the module to set the variables on.
	Module string
	Value  *config.RawConfig

	ModuleTree *module.Tree
	ModulePath []string
}

func (n *GraphNodeConfigVariable) Name() string {
	return fmt.Sprintf("var.%s", n.Variable.Name)
}

func (n *GraphNodeConfigVariable) ConfigType() GraphNodeConfigType {
	return GraphNodeConfigTypeVariable
}

func (n *GraphNodeConfigVariable) DependableName() []string {
	return []string{n.Name()}
}

// RemoveIfNotTargeted implements RemovableIfNotTargeted.
// When targeting is active, variables that are not targeted should be removed
// from the graph, because otherwise module variables trying to interpolate
// their references can fail when they're missing the referent resource node.
func (n *GraphNodeConfigVariable) RemoveIfNotTargeted() bool {
	return true
}

func (n *GraphNodeConfigVariable) DependentOn() []string {
	// If we don't have any value set, we don't depend on anything
	if n.Value == nil {
		return nil
	}

	// Get what we depend on based on our value
	vars := n.Value.Variables
	result := make([]string, 0, len(vars))
	for _, v := range vars {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}

	return result
}

func (n *GraphNodeConfigVariable) VariableName() string {
	return n.Variable.Name
}

// GraphNodeDestroyEdgeInclude impl.
func (n *GraphNodeConfigVariable) DestroyEdgeInclude(v dag.Vertex) bool {
	// Only include this variable in a destroy edge if the source vertex
	// "v" has a count dependency on this variable.
	log.Printf("[DEBUG] DestroyEdgeInclude: Checking: %s", dag.VertexName(v))
	cv, ok := v.(GraphNodeCountDependent)
	if !ok {
		log.Printf("[DEBUG] DestroyEdgeInclude: Not GraphNodeCountDependent: %s", dag.VertexName(v))
		return false
	}

	for _, d := range cv.CountDependentOn() {
		for _, d2 := range n.DependableName() {
			log.Printf("[DEBUG] DestroyEdgeInclude: d = %s : d2 = %s", d, d2)
			if d == d2 {
				return true
			}
		}
	}

	return false
}

// GraphNodeNoopPrunable
func (n *GraphNodeConfigVariable) Noop(opts *NoopOpts) bool {
	log.Printf("[DEBUG] Checking variable noop: %s", n.Name())
	// If we have no diff, always keep this in the graph. We have to do
	// this primarily for validation: we want to validate that variable
	// interpolations are valid even if there are no resources that
	// depend on them.
	if opts.Diff == nil || opts.Diff.Empty() {
		log.Printf("[DEBUG] No diff, not a noop")
		return false
	}

	// We have to find our our module diff since we do funky things with
	// the flat node's implementation of Path() below.
	modDiff := opts.Diff.ModuleByPath(n.ModulePath)

	// If we're destroying, we have no need of variables unless they are depended
	// on by the count of a resource.
	if modDiff != nil && modDiff.Destroy {
		if n.hasDestroyEdgeInPath(opts, nil) {
			log.Printf("[DEBUG] Variable has destroy edge from %s, not a noop",
				dag.VertexName(opts.Vertex))
			return false
		}
		log.Printf("[DEBUG] Variable has no included destroy edges: noop!")
		return true
	}

	for _, v := range opts.Graph.UpEdges(opts.Vertex).List() {
		// This is terrible, but I can't think of a better way to do this.
		if dag.VertexName(v) == rootNodeName {
			continue
		}

		log.Printf("[DEBUG] Found up edge to %s, var is not noop", dag.VertexName(v))
		return false
	}

	log.Printf("[DEBUG] No up edges, treating variable as a noop")
	return true
}

// hasDestroyEdgeInPath recursively walks for a destroy edge, ensuring that
// a variable both has no immediate destroy edges or any in its full module
// path, ensuring that links do not get severed in the middle.
func (n *GraphNodeConfigVariable) hasDestroyEdgeInPath(opts *NoopOpts, vertex dag.Vertex) bool {
	if vertex == nil {
		vertex = opts.Vertex
	}

	log.Printf("[DEBUG] hasDestroyEdgeInPath: Looking for destroy edge: %s - %T", dag.VertexName(vertex), vertex)
	for _, v := range opts.Graph.UpEdges(vertex).List() {
		if len(opts.Graph.UpEdges(v).List()) > 1 {
			if n.hasDestroyEdgeInPath(opts, v) == true {
				return true
			}
		}

		// Here we borrow the implementation of DestroyEdgeInclude, whose logic
		// and semantics are exactly what we want here. We add a check for the
		// the root node, since we have to always depend on its existance.
		if cv, ok := vertex.(*GraphNodeConfigVariableFlat); ok {
			if dag.VertexName(v) == rootNodeName || cv.DestroyEdgeInclude(v) {
				return true
			}
		}
	}
	return false
}

// GraphNodeProxy impl.
func (n *GraphNodeConfigVariable) Proxy() bool {
	return true
}

// GraphNodeEvalable impl.
func (n *GraphNodeConfigVariable) EvalTree() EvalNode {
	// If we have no value, do nothing
	if n.Value == nil {
		return &EvalNoop{}
	}

	// Otherwise, interpolate the value of this variable and set it
	// within the variables mapping.
	var config *ResourceConfig
	variables := make(map[string]interface{})
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalInterpolate{
				Config: n.Value,
				Output: &config,
			},

			&EvalVariableBlock{
				Config:         &config,
				VariableValues: variables,
			},

			&EvalCoerceMapVariable{
				Variables:  variables,
				ModulePath: n.ModulePath,
				ModuleTree: n.ModuleTree,
			},

			&EvalTypeCheckVariable{
				Variables:  variables,
				ModulePath: n.ModulePath,
				ModuleTree: n.ModuleTree,
			},

			&EvalSetVariables{
				Module:    &n.Module,
				Variables: variables,
			},
		},
	}
}

// GraphNodeFlattenable impl.
func (n *GraphNodeConfigVariable) Flatten(p []string) (dag.Vertex, error) {
	return &GraphNodeConfigVariableFlat{
		GraphNodeConfigVariable: n,
		PathValue:               p,
	}, nil
}

type GraphNodeConfigVariableFlat struct {
	*GraphNodeConfigVariable

	PathValue []string
}

func (n *GraphNodeConfigVariableFlat) Name() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue), n.GraphNodeConfigVariable.Name())
}

func (n *GraphNodeConfigVariableFlat) DependableName() []string {
	return []string{n.Name()}
}

func (n *GraphNodeConfigVariableFlat) DependentOn() []string {
	// We only wrap the dependencies and such if we have a path that is
	// longer than 2 elements (root, child, more). This is because when
	// flattened, variables can point outside the graph.
	prefix := ""
	if len(n.PathValue) > 2 {
		prefix = modulePrefixStr(n.PathValue[:len(n.PathValue)-1])
	}

	return modulePrefixList(
		n.GraphNodeConfigVariable.DependentOn(),
		prefix)
}

func (n *GraphNodeConfigVariableFlat) Path() []string {
	if len(n.PathValue) > 2 {
		return n.PathValue[:len(n.PathValue)-1]
	}

	return nil
}

func (n *GraphNodeConfigVariableFlat) Noop(opts *NoopOpts) bool {
	// First look for provider nodes that depend on this variable downstream
	modDiff := opts.Diff.ModuleByPath(n.ModulePath)
	if modDiff != nil && modDiff.Destroy {
		ds, err := opts.Graph.Descendents(n)
		if err != nil {
			log.Printf("[ERROR] Error looking up descendents of %s: %s", n.Name(), err)
		} else {
			for _, d := range ds.List() {
				if _, ok := d.(GraphNodeProvider); ok {
					log.Printf("[DEBUG] This variable is depended on by a provider, can't be a noop.")
					return false
				}
			}
		}
	}

	// Then fall back to existing impl
	return n.GraphNodeConfigVariable.Noop(opts)
}
