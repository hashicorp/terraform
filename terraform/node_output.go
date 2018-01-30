package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
)

// NodeApplyableOutput represents an output that is "applyable":
// it is ready to be applied.
type NodeApplyableOutput struct {
	PathValue []string
	Config    *config.Output // Config is the output in the config
}

func (n *NodeApplyableOutput) Name() string {
	result := fmt.Sprintf("output.%s", n.Config.Name)
	if len(n.PathValue) > 1 {
		result = fmt.Sprintf("%s.%s", modulePrefixStr(n.PathValue), result)
	}

	return result
}

// GraphNodeSubPath
func (n *NodeApplyableOutput) Path() []string {
	return n.PathValue
}

// RemovableIfNotTargeted
func (n *NodeApplyableOutput) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeTargetDownstream
func (n *NodeApplyableOutput) TargetDownstream(targetedDeps, untargetedDeps *dag.Set) bool {
	// If any of the direct dependencies of an output are targeted then
	// the output must always be targeted as well, so its value will always
	// be up-to-date at the completion of an apply walk.
	return true
}

// GraphNodeReferenceable
func (n *NodeApplyableOutput) ReferenceableName() []string {
	name := fmt.Sprintf("output.%s", n.Config.Name)
	return []string{name}
}

// GraphNodeReferencer
func (n *NodeApplyableOutput) References() []string {
	var result []string
	result = append(result, n.Config.DependsOn...)
	result = append(result, ReferencesFromConfig(n.Config.RawConfig)...)
	for _, v := range result {
		split := strings.Split(v, "/")
		for i, s := range split {
			split[i] = s + ".destroy"
		}

		result = append(result, strings.Join(split, "/"))
	}

	return result
}

// GraphNodeEvalable
func (n *NodeApplyableOutput) EvalTree() EvalNode {
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalOpFilter{
				// Don't let interpolation errors stop Input, since it happens
				// before Refresh.
				Ops: []walkOperation{walkInput},
				Node: &EvalWriteOutput{
					Name:          n.Config.Name,
					Sensitive:     n.Config.Sensitive,
					Value:         n.Config.RawConfig,
					ContinueOnErr: true,
				},
			},
			&EvalOpFilter{
				Ops: []walkOperation{walkRefresh, walkPlan, walkApply, walkValidate, walkDestroy, walkPlanDestroy},
				Node: &EvalWriteOutput{
					Name:      n.Config.Name,
					Sensitive: n.Config.Sensitive,
					Value:     n.Config.RawConfig,
				},
			},
		},
	}
}

// NodeDestroyableOutput represents an output that is "destroybale":
// its application will remove the output from the state.
type NodeDestroyableOutput struct {
	PathValue []string
	Config    *config.Output // Config is the output in the config
}

func (n *NodeDestroyableOutput) Name() string {
	result := fmt.Sprintf("output.%s (destroy)", n.Config.Name)
	if len(n.PathValue) > 1 {
		result = fmt.Sprintf("%s.%s", modulePrefixStr(n.PathValue), result)
	}

	return result
}

// GraphNodeSubPath
func (n *NodeDestroyableOutput) Path() []string {
	return n.PathValue
}

// RemovableIfNotTargeted
func (n *NodeDestroyableOutput) RemoveIfNotTargeted() bool {
	// We need to add this so that this node will be removed if
	// it isn't targeted or a dependency of a target.
	return true
}

// GraphNodeReferencer
func (n *NodeDestroyableOutput) References() []string {
	var result []string
	result = append(result, n.Config.DependsOn...)
	result = append(result, ReferencesFromConfig(n.Config.RawConfig)...)
	for _, v := range result {
		split := strings.Split(v, "/")
		for i, s := range split {
			split[i] = s + ".destroy"
		}

		result = append(result, strings.Join(split, "/"))
	}

	return result
}

// GraphNodeEvalable
func (n *NodeDestroyableOutput) EvalTree() EvalNode {
	return &EvalDeleteOutput{
		Name: n.Config.Name,
	}
}
