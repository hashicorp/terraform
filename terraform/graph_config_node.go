package terraform

import (
	"fmt"
	"strings"

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
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalWriteOutput{
					Name:  n.Output.Name,
					Value: n.Output.RawConfig,
				},
			},
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

// GraphNodeDotter impl.
func (n *GraphNodeConfigProvider) Dot(name string) string {
	return fmt.Sprintf(
		"\"%s\" [\n"+
			"\tlabel=\"%s\"\n"+
			"\tshape=diamond\n"+
			"];",
		name,
		n.Name())
}

// GraphNodeConfigResource represents a resource within the config graph.
type GraphNodeConfigResource struct {
	Resource *config.Resource

	// If this is set to anything other than destroyModeNone, then this
	// resource represents a resource that will be destroyed in some way.
	DestroyMode GraphNodeDestroyMode
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
	switch n.DestroyMode {
	case DestroyNone:
	case DestroyPrimary:
		result += " (destroy)"
	case DestroyTainted:
		result += " (destroy tainted)"
	default:
		result += " (unknown destroy type)"
	}

	return result
}

// GraphNodeDotter impl.
func (n *GraphNodeConfigResource) Dot(name string) string {
	if n.DestroyMode != DestroyNone {
		return ""
	}

	return fmt.Sprintf(
		"\"%s\" [\n"+
			"\tlabel=\"%s\"\n"+
			"\tshape=box\n"+
			"];",
		name,
		n.Name())
}

// GraphNodeDynamicExpandable impl.
func (n *GraphNodeConfigResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	state, lock := ctx.State()
	lock.RLock()
	defer lock.RUnlock()

	// Start creating the steps
	steps := make([]GraphTransformer, 0, 5)

	// Primary and non-destroy modes are responsible for creating/destroying
	// all the nodes, expanding counts.
	switch n.DestroyMode {
	case DestroyNone:
		fallthrough
	case DestroyPrimary:
		steps = append(steps, &ResourceCountTransformer{
			Resource: n.Resource,
			Destroy:  n.DestroyMode != DestroyNone,
		})
	}

	// Additional destroy modifications.
	switch n.DestroyMode {
	case DestroyPrimary:
		// If we're destroying the primary instance, then we want to
		// expand orphans, which have all the same semantics in a destroy
		// as a primary.
		steps = append(steps, &OrphanTransformer{
			State: state,
			View:  n.Resource.Id(),
		})

		steps = append(steps, &DeposedTransformer{
			State: state,
			View:  n.Resource.Id(),
		})
	case DestroyTainted:
		// If we're only destroying tainted resources, then we only
		// want to find tainted resources and destroy them here.
		steps = append(steps, &TaintedTransformer{
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
func (n *GraphNodeConfigResource) DestroyNode(mode GraphNodeDestroyMode) GraphNodeDestroy {
	// If we're already a destroy node, then don't do anything
	if n.DestroyMode != DestroyNone {
		return nil
	}

	result := &graphNodeResourceDestroy{
		GraphNodeConfigResource: *n,
		Original:                n,
	}
	result.DestroyMode = mode
	return result
}

// graphNodeResourceDestroy represents the logical destruction of a
// resource. This node doesn't mean it will be destroyed for sure, but
// instead that if a destroy were to happen, it must happen at this point.
type graphNodeResourceDestroy struct {
	GraphNodeConfigResource
	Original *GraphNodeConfigResource
}

func (n *graphNodeResourceDestroy) CreateBeforeDestroy() bool {
	// CBD is enabled if the resource enables it in addition to us
	// being responsible for destroying the primary state. The primary
	// state destroy node is the only destroy node that needs to be
	// "shuffled" according to the CBD rules, since tainted resources
	// don't have the same inverse dependencies.
	return n.Original.Resource.Lifecycle.CreateBeforeDestroy &&
		n.DestroyMode == DestroyPrimary
}

func (n *graphNodeResourceDestroy) CreateNode() dag.Vertex {
	return n.Original
}

func (n *graphNodeResourceDestroy) DestroyInclude(d *ModuleDiff, s *ModuleState) bool {
	// Always include anything other than the primary destroy
	if n.DestroyMode != DestroyPrimary {
		return true
	}

	// Get the count, and specifically the raw value of the count
	// (with interpolations and all). If the count is NOT a static "1",
	// then we keep the destroy node no matter what.
	//
	// The reasoning for this is complicated and not intuitively obvious,
	// but I attempt to explain it below.
	//
	// The destroy transform works by generating the worst case graph,
	// with worst case being the case that every resource already exists
	// and needs to be destroy/created (force-new). There is a single important
	// edge case where this actually results in a real-life cycle: if a
	// create-before-destroy (CBD) resource depends on a non-CBD resource.
	// Imagine a EC2 instance "foo" with CBD depending on a security
	// group "bar" without CBD, and conceptualize the worst case destroy
	// order:
	//
	//   1.) SG must be destroyed (non-CBD)
	//   2.) SG must be created/updated
	//   3.) EC2 instance must be created (CBD, requires the SG be made)
	//   4.) EC2 instance must be destroyed (requires SG be destroyed)
	//
	// Except, #1 depends on #4, since the SG can't be destroyed while
	// an EC2 instance is using it (AWS API requirements). As you can see,
	// this is a real life cycle that can't be automatically reconciled
	// except under two conditions:
	//
	//   1.) SG is also CBD. This doesn't work 100% of the time though
	//       since the non-CBD resource might not support CBD. To make matters
	//       worse, the entire transitive closure of dependencies must be
	//       CBD (if the SG depends on a VPC, you have the same problem).
	//   2.) EC2 must not CBD. This can't happen automatically because CBD
	//       is used as a way to ensure zero (or minimal) downtime Terraform
	//       applies, and it isn't acceptable for TF to ignore this request,
	//       since it can result in unexpected downtime.
	//
	// Therefore, we compromise with this edge case here: if there is
	// a static count of "1", we prune the diff to remove cycles during a
	// graph optimization path if we don't see the resource in the diff.
	// If the count is set to ANYTHING other than a static "1" (variable,
	// computed attribute, static number greater than 1), then we keep the
	// destroy, since it is required for dynamic graph expansion to find
	// orphan/tainted count objects.
	//
	// This isn't ideal logic, but its strictly better without introducing
	// new impossibilities. It breaks the cycle in practical cases, and the
	// cycle comes back in no cases we've found to be practical, but just
	// as the cycle would already exist without this anyways.
	count := n.Original.Resource.RawCount
	if raw := count.Raw[count.Key]; raw != "1" {
		return true
	}

	// Okay, we're dealing with a static count. There are a few ways
	// to include this resource.
	prefix := n.Original.Resource.Id()

	// If we're present in the diff proper, then keep it.
	if d != nil {
		for k, _ := range d.Resources {
			if strings.HasPrefix(k, prefix) {
				return true
			}
		}
	}

	// If we're in the state as a primary in any form, then keep it.
	// This does a prefix check so it will also catch orphans on count
	// decreases to "1".
	if s != nil {
		for k, v := range s.Resources {
			if !strings.HasPrefix(k, prefix) {
				continue
			}

			// Ignore exact matches and the 0'th index. We only care
			// about if there is a decrease in count.
			if k == prefix {
				continue
			}
			if k == prefix+".0" {
				continue
			}

			if v.Primary != nil {
				return true
			}
		}

		// If we're in the state as _both_ "foo" and "foo.0", then
		// keep it, since we treat the latter as an orphan.
		_, okOne := s.Resources[prefix]
		_, okTwo := s.Resources[prefix+".0"]
		if okOne && okTwo {
			return true
		}
	}

	return false
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

// GraphNodeDotter impl.
func (n *graphNodeModuleExpanded) Dot(name string) string {
	return fmt.Sprintf(
		"\"%s\" [\n"+
			"\tlabel=\"%s\"\n"+
			"\tshape=component\n"+
			"];",
		name,
		dag.VertexName(n.Original))
}

// GraphNodeEvalable impl.
func (n *graphNodeModuleExpanded) EvalTree() EvalNode {
	var resourceConfig *ResourceConfig
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalInterpolate{
				Config: n.InputConfig,
				Output: &resourceConfig,
			},

			&EvalVariableBlock{
				Config:    &resourceConfig,
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
