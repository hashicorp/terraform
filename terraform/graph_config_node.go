package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
)

// graphNodeConfig is an interface that all graph nodes for the
// configuration graph need to implement in order to build the variable
// dependencies properly.
type graphNodeConfig interface {
	dag.NamedVertex

	// All graph nodes should be dependent on other things, and able to
	// be depended on.
	GraphNodeDependable
	GraphNodeDependent
}

// GraphNodeConfigModule represents a module within the configuration graph.
type GraphNodeConfigModule struct {
	Path   []string
	Module *config.Module
	Tree   *module.Tree
}

func (n *GraphNodeConfigModule) DependableName() []string {
	return []string{n.Name()}
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

	// Add the parameters node to the module
	t := &ModuleInputTransformer{Variables: make(map[string]string)}
	if err := t.Transform(graph); err != nil {
		return nil, err
	}

	// Build the actual subgraph node
	return &graphNodeModuleExpanded{
		Original:    n,
		Graph:       graph,
		InputConfig: n.Module.RawConfig,
		Variables:   t.Variables,
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
		providers[resourceProvider(r.Type)] = struct{}{}
	}

	// Turn the map into a string. This makes sure that the list is
	// de-dupped since we could be going over potentially many resources.
	result := make([]string, 0, len(providers))
	for p, _ := range providers {
		result = append(result, p)
	}

	return result
}

// GraphNodeConfigOutput represents an output configured within the
// configuration.
type GraphNodeConfigOutput struct {
	Output *config.Output
}

func (n *GraphNodeConfigOutput) Name() string {
	return fmt.Sprintf("output.%s", n.Output.Name)
}

func (n *GraphNodeConfigOutput) DependableName() []string {
	return []string{n.Name()}
}

func (n *GraphNodeConfigOutput) DependentOn() []string {
	vars := n.Output.RawConfig.Variables
	result := make([]string, 0, len(vars))
	for _, v := range vars {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}

	return result
}

// GraphNodeEvalable impl.
func (n *GraphNodeConfigOutput) EvalTree() EvalNode {
	return &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkPlan, walkApply},
		Node: &EvalWriteOutput{
			Name:  n.Output.Name,
			Value: &EvalInterpolate{Config: n.Output.RawConfig},
		},
	}
}

// GraphNodeConfigProvider represents a configured provider within the
// configuration graph. These are only immediately in the graph when an
// explicit `provider` configuration block is in the configuration.
type GraphNodeConfigProvider struct {
	Provider *config.ProviderConfig
}

func (n *GraphNodeConfigProvider) Name() string {
	return fmt.Sprintf("provider.%s", n.Provider.Name)
}

func (n *GraphNodeConfigProvider) DependableName() []string {
	return []string{n.Name()}
}

func (n *GraphNodeConfigProvider) DependentOn() []string {
	vars := n.Provider.RawConfig.Variables
	result := make([]string, 0, len(vars))
	for _, v := range vars {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}

	return result
}

// GraphNodeEvalable impl.
func (n *GraphNodeConfigProvider) EvalTree() EvalNode {
	return ProviderEvalTree(n.Provider.Name, n.Provider.RawConfig)
}

// GraphNodeProvider implementation
func (n *GraphNodeConfigProvider) ProviderName() string {
	return n.Provider.Name
}

// GraphNodeConfigResource represents a resource within the config graph.
type GraphNodeConfigResource struct {
	Resource *config.Resource

	// If set to true, this represents a resource that can only be
	// destroyed. It doesn't mean that the resource WILL be destroyed, only
	// that logically this node is where it would happen.
	Destroy bool
}

func (n *GraphNodeConfigResource) DependableName() []string {
	return []string{n.Resource.Id()}
}

// GraphNodeDependent impl.
func (n *GraphNodeConfigResource) DependentOn() []string {
	result := make([]string, len(n.Resource.DependsOn),
		(len(n.Resource.RawCount.Variables)+
			len(n.Resource.RawConfig.Variables)+
			len(n.Resource.DependsOn))*2)
	copy(result, n.Resource.DependsOn)

	for _, v := range n.Resource.RawCount.Variables {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}
	for _, v := range n.Resource.RawConfig.Variables {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}
	for _, p := range n.Resource.Provisioners {
		for _, v := range p.ConnInfo.Variables {
			if vn := varNameForVar(v); vn != "" && vn != n.Resource.Id() {
				result = append(result, vn)
			}
		}
		for _, v := range p.RawConfig.Variables {
			if vn := varNameForVar(v); vn != "" && vn != n.Resource.Id() {
				result = append(result, vn)
			}
		}
	}

	return result
}

func (n *GraphNodeConfigResource) Name() string {
	result := n.Resource.Id()
	if n.Destroy {
		result += " (destroy)"
	}

	return result
}

// GraphNodeDynamicExpandable impl.
func (n *GraphNodeConfigResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	// Start creating the steps
	steps := make([]GraphTransformer, 0, 5)
	steps = append(steps, &ResourceCountTransformer{
		Resource: n.Resource,
		Destroy:  n.Destroy,
	})

	// If we're destroying, then we care about adding orphans to
	// the graph. Orphans in this case are the leftover resources when
	// we decrease count.
	if n.Destroy {
		state, lock := ctx.State()
		lock.RLock()
		defer lock.RUnlock()

		steps = append(steps, &OrphanTransformer{
			State: state,
			View:  n.Resource.Id(),
		})
	}

	// Always end with the root being added
	steps = append(steps, &RootTransformer{})

	// Build the graph
	b := &BasicGraphBuilder{Steps: steps}
	return b.Build(ctx.Path())
}

// GraphNodeEvalable impl.
func (n *GraphNodeConfigResource) EvalTree() EvalNode {
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalInterpolate{Config: n.Resource.RawCount},
			&EvalOpFilter{
				Ops:  []walkOperation{walkValidate},
				Node: &EvalValidateCount{Resource: n.Resource},
			},
			&EvalCountFixZeroOneBoundary{Resource: n.Resource},
		},
	}
}

// GraphNodeProviderConsumer
func (n *GraphNodeConfigResource) ProvidedBy() []string {
	return []string{resourceProvider(n.Resource.Type)}
}

// GraphNodeProvisionerConsumer
func (n *GraphNodeConfigResource) ProvisionedBy() []string {
	result := make([]string, len(n.Resource.Provisioners))
	for i, p := range n.Resource.Provisioners {
		result[i] = p.Type
	}

	return result
}

// GraphNodeDestroyable
func (n *GraphNodeConfigResource) DestroyNode() dag.Vertex {
	// If we're already a destroy node, then don't do anything
	if n.Destroy {
		return nil
	}

	// Just make a copy that is set to destroy
	result := *n
	result.Destroy = true

	return &result
}

// graphNodeModuleExpanded represents a module where the graph has
// been expanded. It stores the graph of the module as well as a reference
// to the map of variables.
type graphNodeModuleExpanded struct {
	Original    dag.Vertex
	Graph       *Graph
	InputConfig *config.RawConfig

	// Variables is a map of the input variables. This reference should
	// be shared with ModuleInputTransformer in order to create a connection
	// where the variables are set properly.
	Variables map[string]string
}

func (n *graphNodeModuleExpanded) Name() string {
	return fmt.Sprintf("%s (expanded)", dag.VertexName(n.Original))
}

// GraphNodeEvalable impl.
func (n *graphNodeModuleExpanded) EvalTree() EvalNode {
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalVariableBlock{
				Config:    &EvalInterpolate{Config: n.InputConfig},
				Variables: n.Variables,
			},

			&EvalOpFilter{
				Ops: []walkOperation{walkPlanDestroy},
				Node: &EvalSequence{
					Nodes: []EvalNode{
						&EvalDiffDestroyModule{Path: n.Graph.Path},
					},
				},
			},
		},
	}
}

// GraphNodeSubgraph impl.
func (n *graphNodeModuleExpanded) Subgraph() *Graph {
	return n.Graph
}
