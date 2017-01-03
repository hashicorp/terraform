package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// DisableProviderTransformer "disables" any providers that are only
// depended on by modules.
//
// NOTE: "old" = used by old graph builders, will be removed one day
type DisableProviderTransformerOld struct{}

func (t *DisableProviderTransformerOld) Transform(g *Graph) error {
	// Since we're comparing against edges, we need to make sure we connect
	g.ConnectDependents()

	for _, v := range g.Vertices() {
		// We only care about providers
		pn, ok := v.(GraphNodeProvider)
		if !ok || pn.ProviderName() == "" {
			continue
		}

		// Go through all the up-edges (things that depend on this
		// provider) and if any is not a module, then ignore this node.
		nonModule := false
		for _, sourceRaw := range g.UpEdges(v).List() {
			source := sourceRaw.(dag.Vertex)
			cn, ok := source.(graphNodeConfig)
			if !ok {
				nonModule = true
				break
			}

			if cn.ConfigType() != GraphNodeConfigTypeModule {
				nonModule = true
				break
			}
		}
		if nonModule {
			// We found something that depends on this provider that
			// isn't a module, so skip it.
			continue
		}

		// Disable the provider by replacing it with a "disabled" provider
		disabled := &graphNodeDisabledProvider{GraphNodeProvider: pn}
		if !g.Replace(v, disabled) {
			panic(fmt.Sprintf(
				"vertex disappeared from under us: %s",
				dag.VertexName(v)))
		}
	}

	return nil
}

type graphNodeDisabledProvider struct {
	GraphNodeProvider
}

// GraphNodeEvalable impl.
func (n *graphNodeDisabledProvider) EvalTree() EvalNode {
	var resourceConfig *ResourceConfig

	return &EvalOpFilter{
		Ops: []walkOperation{walkInput, walkValidate, walkRefresh, walkPlan, walkApply, walkDestroy},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalInterpolate{
					Config: n.ProviderConfig(),
					Output: &resourceConfig,
				},
				&EvalBuildProviderConfig{
					Provider: n.ProviderName(),
					Config:   &resourceConfig,
					Output:   &resourceConfig,
				},
				&EvalSetProviderConfig{
					Provider: n.ProviderName(),
					Config:   &resourceConfig,
				},
			},
		},
	}
}

// GraphNodeFlattenable impl.
func (n *graphNodeDisabledProvider) Flatten(p []string) (dag.Vertex, error) {
	return &graphNodeDisabledProviderFlat{
		graphNodeDisabledProvider: n,
		PathValue:                 p,
	}, nil
}

func (n *graphNodeDisabledProvider) Name() string {
	return fmt.Sprintf("%s (disabled)", dag.VertexName(n.GraphNodeProvider))
}

// GraphNodeDotter impl.
func (n *graphNodeDisabledProvider) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label": n.Name(),
			"shape": "diamond",
		},
	}
}

// GraphNodeDotterOrigin impl.
func (n *graphNodeDisabledProvider) DotOrigin() bool {
	return true
}

// GraphNodeDependable impl.
func (n *graphNodeDisabledProvider) DependableName() []string {
	return []string{"provider." + n.ProviderName()}
}

// GraphNodeProvider impl.
func (n *graphNodeDisabledProvider) ProviderName() string {
	return n.GraphNodeProvider.ProviderName()
}

// GraphNodeProvider impl.
func (n *graphNodeDisabledProvider) ProviderConfig() *config.RawConfig {
	return n.GraphNodeProvider.ProviderConfig()
}

// Same as graphNodeDisabledProvider, but for flattening
type graphNodeDisabledProviderFlat struct {
	*graphNodeDisabledProvider

	PathValue []string
}

func (n *graphNodeDisabledProviderFlat) Name() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue), n.graphNodeDisabledProvider.Name())
}

func (n *graphNodeDisabledProviderFlat) Path() []string {
	return n.PathValue
}

func (n *graphNodeDisabledProviderFlat) ProviderName() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue),
		n.graphNodeDisabledProvider.ProviderName())
}

// GraphNodeDependable impl.
func (n *graphNodeDisabledProviderFlat) DependableName() []string {
	return modulePrefixList(
		n.graphNodeDisabledProvider.DependableName(),
		modulePrefixStr(n.PathValue))
}

func (n *graphNodeDisabledProviderFlat) DependentOn() []string {
	var result []string

	// If we're in a module, then depend on our parent's provider
	if len(n.PathValue) > 1 {
		prefix := modulePrefixStr(n.PathValue[:len(n.PathValue)-1])
		result = modulePrefixList(
			n.graphNodeDisabledProvider.DependableName(), prefix)
	}

	return result
}
