package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/dot"
)

// GraphNodeConfigModule represents a module within the configuration graph.
type GraphNodeConfigModule struct {
	Path   []string
	Module *config.Module
	Tree   *module.Tree
}

func (n *GraphNodeConfigModule) ConfigType() GraphNodeConfigType {
	return GraphNodeConfigTypeModule
}

func (n *GraphNodeConfigModule) DependableName() []string {
	config := n.Tree.Config()

	result := make([]string, 1, len(config.Outputs)+1)
	result[0] = n.Name()
	for _, o := range config.Outputs {
		result = append(result, fmt.Sprintf("%s.output.%s", n.Name(), o.Name))
	}

	return result
}

func (n *GraphNodeConfigModule) DependentOn() []string {
	vars := n.Module.RawConfig.Variables
	result := make([]string, 0, len(vars))
	for _, v := range vars {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}

	return result
}

func (n *GraphNodeConfigModule) Name() string {
	return fmt.Sprintf("module.%s", n.Module.Name)
}

// GraphNodeExpandable
func (n *GraphNodeConfigModule) Expand(b GraphBuilder) (GraphNodeSubgraph, error) {
	// Build the graph first
	graph, err := b.Build(n.Path)
	if err != nil {
		return nil, err
	}

	{
		// Add the destroy marker to the graph
		t := &ModuleDestroyTransformer{}
		if err := t.Transform(graph); err != nil {
			return nil, err
		}
	}

	// Build the actual subgraph node
	return &graphNodeModuleExpanded{
		Original:  n,
		Graph:     graph,
		Variables: make(map[string]string),
	}, nil
}

// GraphNodeExpandable
func (n *GraphNodeConfigModule) ProvidedBy() []string {
	// Build up the list of providers by simply going over our configuration
	// to find the providers that are configured there as well as the
	// providers that the resources use.
	config := n.Tree.Config()
	providers := make(map[string]struct{})
	for _, p := range config.ProviderConfigs {
		providers[p.Name] = struct{}{}
	}
	for _, r := range config.Resources {
		providers[resourceProvider(r.Type, r.Provider)] = struct{}{}
	}

	// Turn the map into a string. This makes sure that the list is
	// de-dupped since we could be going over potentially many resources.
	result := make([]string, 0, len(providers))
	for p, _ := range providers {
		result = append(result, p)
	}

	return result
}

// graphNodeModuleExpanded represents a module where the graph has
// been expanded. It stores the graph of the module as well as a reference
// to the map of variables.
type graphNodeModuleExpanded struct {
	Original *GraphNodeConfigModule
	Graph    *Graph

	// Variables is a map of the input variables. This reference should
	// be shared with ModuleInputTransformer in order to create a connection
	// where the variables are set properly.
	Variables map[string]string
}

func (n *graphNodeModuleExpanded) Name() string {
	return fmt.Sprintf("%s (expanded)", dag.VertexName(n.Original))
}

func (n *graphNodeModuleExpanded) ConfigType() GraphNodeConfigType {
	return GraphNodeConfigTypeModule
}

// GraphNodeDependable
func (n *graphNodeModuleExpanded) DependableName() []string {
	return n.Original.DependableName()
}

// GraphNodeDependent
func (n *graphNodeModuleExpanded) DependentOn() []string {
	return n.Original.DependentOn()
}

// GraphNodeDotter impl.
func (n *graphNodeModuleExpanded) DotNode(name string, opts *GraphDotOpts) *dot.Node {
	return dot.NewNode(name, map[string]string{
		"label": dag.VertexName(n.Original),
		"shape": "component",
	})
}

// GraphNodeEvalable impl.
func (n *graphNodeModuleExpanded) EvalTree() EvalNode {
	var resourceConfig *ResourceConfig
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalInterpolate{
				Config: n.Original.Module.RawConfig,
				Output: &resourceConfig,
			},

			&EvalVariableBlock{
				Config:    &resourceConfig,
				Variables: n.Variables,
			},
		},
	}
}

// GraphNodeFlattenable impl.
func (n *graphNodeModuleExpanded) FlattenGraph() *Graph {
	graph := n.Subgraph()
	input := n.Original.Module.RawConfig

	// Go over each vertex and do some modifications to the graph for
	// flattening. We have to skip some nodes (graphNodeModuleSkippable)
	// as well as setup the variable values.
	for _, v := range graph.Vertices() {
		// If this is a variable, then look it up in the raw configuration.
		// If it exists in the raw configuration, set the value of it.
		if vn, ok := v.(*GraphNodeConfigVariable); ok && input != nil {
			key := vn.VariableName()
			if v, ok := input.Raw[key]; ok {
				config, err := config.NewRawConfig(map[string]interface{}{
					key: v,
				})
				if err != nil {
					// This shouldn't happen because it is already in
					// a RawConfig above meaning it worked once before.
					panic(err)
				}

				// Set the variable value so it is interpolated properly.
				// Also set the module so we set the value on it properly.
				vn.Module = graph.Path[len(graph.Path)-1]
				vn.Value = config
			}
		}
	}

	return graph
}

// GraphNodeSubgraph impl.
func (n *graphNodeModuleExpanded) Subgraph() *Graph {
	return n.Graph
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
