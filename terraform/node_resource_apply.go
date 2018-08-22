package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// NodeApplyableResource represents a resource that is "applyable":
// it is ready to be applied and is represented by a diff.
type NodeApplyableResource struct {
	*NodeAbstractResource
}

// GraphNodeCreator
func (n *NodeApplyableResource) CreateAddr() *ResourceAddress {
	return n.NodeAbstractResource.Addr
}

// GraphNodeReferencer, overriding NodeAbstractResource
func (n *NodeApplyableResource) References() []string {
	result := n.NodeAbstractResource.References()

	// The "apply" side of a resource generally also depends on the
	// destruction of its dependencies as well. For example, if a LB
	// references a set of VMs with ${vm.foo.*.id}, then we must wait for
	// the destruction so we get the newly updated list of VMs.
	//
	// The exception here is CBD. When CBD is set, we don't do this since
	// it would create a cycle. By not creating a cycle, we require two
	// applies since the first apply the creation step will use the OLD
	// values (pre-destroy) and the second step will update.
	//
	// This is how Terraform behaved with "legacy" graphs (TF <= 0.7.x).
	// We mimic that behavior here now and can improve upon it in the future.
	//
	// This behavior is tested in graph_build_apply_test.go to test ordering.
	cbd := n.Config != nil && n.Config.Lifecycle.CreateBeforeDestroy
	if !cbd {
		// The "apply" side of a resource always depends on the destruction
		// of all its dependencies in addition to the creation.
		for _, v := range result {
			result = append(result, v+".destroy")
		}
	}

	return result
}

// GraphNodeEvalable
func (n *NodeApplyableResource) EvalTree() EvalNode {
	addr := n.NodeAbstractResource.Addr

	// stateId is the ID to put into the state
	stateId := addr.stateId()

	// Build the instance info. More of this will be populated during eval
	info := &InstanceInfo{
		Id:   stateId,
		Type: addr.Type,
	}

	// Build the resource for eval
	resource := &Resource{
		Name:       addr.Name,
		Type:       addr.Type,
		CountIndex: addr.Index,
	}
	if resource.CountIndex < 0 {
		resource.CountIndex = 0
	}

	// Determine the dependencies for the state.
	stateDeps := n.StateReferences()

	// Eval info is different depending on what kind of resource this is
	switch n.Config.Mode {
	case config.ManagedResourceMode:
		return n.evalTreeManagedResource(
			stateId, info, resource, stateDeps,
		)
	case config.DataResourceMode:
		return n.evalTreeDataResource(
			stateId, info, resource, stateDeps)
	default:
		panic(fmt.Errorf("unsupported resource mode %s", n.Config.Mode))
	}
}

func (n *NodeApplyableResource) evalTreeDataResource(
	stateId string, info *InstanceInfo,
	resource *Resource, stateDeps []string) EvalNode {
	var provider ResourceProvider
	var config *ResourceConfig
	var diff *InstanceDiff
	var state *InstanceState

	return &EvalSequence{
		Nodes: []EvalNode{
			// Build the instance info
			&EvalInstanceInfo{
				Info: info,
			},

			// Get the saved diff for apply
			&EvalReadDiff{
				Name: stateId,
				Diff: &diff,
			},

			// Stop here if we don't actually have a diff
			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					if diff == nil {
						return true, EvalEarlyExitError{}
					}

					if diff.GetAttributesLen() == 0 {
						return true, EvalEarlyExitError{}
					}

					return true, nil
				},
				Then: EvalNoop{},
			},

			// Normally we interpolate count as a preparation step before
			// a DynamicExpand, but an apply graph has pre-expanded nodes
			// and so the count would otherwise never be interpolated.
			//
			// This is redundant when there are multiple instances created
			// from the same config (count > 1) but harmless since the
			// underlying structures have mutexes to make this concurrency-safe.
			//
			// In most cases this isn't actually needed because we dealt with
			// all of the counts during the plan walk, but we do it here
			// for completeness because other code assumes that the
			// final count is always available during interpolation.
			//
			// Here we are just populating the interpolated value in-place
			// inside this RawConfig object, like we would in
			// NodeAbstractCountResource.
			&EvalInterpolate{
				Config:        n.Config.RawCount,
				ContinueOnErr: true,
			},

			// We need to re-interpolate the config here, rather than
			// just using the diff's values directly, because we've
			// potentially learned more variable values during the
			// apply pass that weren't known when the diff was produced.
			&EvalInterpolate{
				Config:   n.Config.RawConfig.Copy(),
				Resource: resource,
				Output:   &config,
			},

			&EvalGetProvider{
				Name:   n.ResolvedProvider,
				Output: &provider,
			},

			// Make a new diff with our newly-interpolated config.
			&EvalReadDataDiff{
				Info:     info,
				Config:   &config,
				Previous: &diff,
				Provider: &provider,
				Output:   &diff,
			},

			&EvalReadDataApply{
				Info:     info,
				Diff:     &diff,
				Provider: &provider,
				Output:   &state,
			},

			&EvalWriteState{
				Name:         stateId,
				ResourceType: n.Config.Type,
				Provider:     n.ResolvedProvider,
				Dependencies: stateDeps,
				State:        &state,
			},

			// Clear the diff now that we've applied it, so
			// later nodes won't see a diff that's now a no-op.
			&EvalWriteDiff{
				Name: stateId,
				Diff: nil,
			},

			&EvalUpdateStateHook{},
		},
	}
}

func (n *NodeApplyableResource) evalTreeManagedResource(
	stateId string, info *InstanceInfo,
	resource *Resource, stateDeps []string) EvalNode {
	// Declare a bunch of variables that are used for state during
	// evaluation. Most of this are written to by-address below.
	var provider ResourceProvider
	var diff, diffApply *InstanceDiff
	var state *InstanceState
	var resourceConfig *ResourceConfig
	var err error
	var createNew bool
	var createBeforeDestroyEnabled bool

	return &EvalSequence{
		Nodes: []EvalNode{
			// Build the instance info
			&EvalInstanceInfo{
				Info: info,
			},

			// Get the saved diff for apply
			&EvalReadDiff{
				Name: stateId,
				Diff: &diffApply,
			},

			// We don't want to do any destroys
			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					if diffApply == nil {
						return true, EvalEarlyExitError{}
					}

					if diffApply.GetDestroy() && diffApply.GetAttributesLen() == 0 {
						return true, EvalEarlyExitError{}
					}

					diffApply.SetDestroy(false)
					return true, nil
				},
				Then: EvalNoop{},
			},

			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					destroy := false
					if diffApply != nil {
						destroy = diffApply.GetDestroy() || diffApply.RequiresNew()
					}

					createBeforeDestroyEnabled =
						n.Config.Lifecycle.CreateBeforeDestroy &&
							destroy

					return createBeforeDestroyEnabled, nil
				},
				Then: &EvalDeposeState{
					Name: stateId,
				},
			},

			// Normally we interpolate count as a preparation step before
			// a DynamicExpand, but an apply graph has pre-expanded nodes
			// and so the count would otherwise never be interpolated.
			//
			// This is redundant when there are multiple instances created
			// from the same config (count > 1) but harmless since the
			// underlying structures have mutexes to make this concurrency-safe.
			//
			// In most cases this isn't actually needed because we dealt with
			// all of the counts during the plan walk, but we need to do this
			// in order to support interpolation of resource counts from
			// apply-time-interpolated expressions, such as those in
			// "provisioner" blocks.
			//
			// Here we are just populating the interpolated value in-place
			// inside this RawConfig object, like we would in
			// NodeAbstractCountResource.
			&EvalInterpolate{
				Config:        n.Config.RawCount,
				ContinueOnErr: true,
			},

			&EvalInterpolate{
				Config:   n.Config.RawConfig.Copy(),
				Resource: resource,
				Output:   &resourceConfig,
			},
			&EvalGetProvider{
				Name:   n.ResolvedProvider,
				Output: &provider,
			},
			&EvalReadState{
				Name:   stateId,
				Output: &state,
			},
			// Re-run validation to catch any errors we missed, e.g. type
			// mismatches on computed values.
			&EvalValidateResource{
				Provider:       &provider,
				Config:         &resourceConfig,
				ResourceName:   n.Config.Name,
				ResourceType:   n.Config.Type,
				ResourceMode:   n.Config.Mode,
				IgnoreWarnings: true,
			},
			&EvalDiff{
				Info:       info,
				Config:     &resourceConfig,
				Resource:   n.Config,
				Provider:   &provider,
				Diff:       &diffApply,
				State:      &state,
				OutputDiff: &diffApply,
			},

			// Get the saved diff
			&EvalReadDiff{
				Name: stateId,
				Diff: &diff,
			},

			// Compare the diffs
			&EvalCompareDiff{
				Info: info,
				One:  &diff,
				Two:  &diffApply,
			},

			&EvalGetProvider{
				Name:   n.ResolvedProvider,
				Output: &provider,
			},
			&EvalReadState{
				Name:   stateId,
				Output: &state,
			},
			// Call pre-apply hook
			&EvalApplyPre{
				Info:  info,
				State: &state,
				Diff:  &diffApply,
			},
			&EvalApply{
				Info:      info,
				State:     &state,
				Diff:      &diffApply,
				Provider:  &provider,
				Output:    &state,
				Error:     &err,
				CreateNew: &createNew,
			},
			&EvalWriteState{
				Name:         stateId,
				ResourceType: n.Config.Type,
				Provider:     n.ResolvedProvider,
				Dependencies: stateDeps,
				State:        &state,
			},
			&EvalApplyProvisioners{
				Info:           info,
				State:          &state,
				Resource:       n.Config,
				InterpResource: resource,
				CreateNew:      &createNew,
				Error:          &err,
				When:           config.ProvisionerWhenCreate,
			},
			&EvalIf{
				If: func(ctx EvalContext) (bool, error) {
					return createBeforeDestroyEnabled && err != nil, nil
				},
				Then: &EvalUndeposeState{
					Name:  stateId,
					State: &state,
				},
				Else: &EvalWriteState{
					Name:         stateId,
					ResourceType: n.Config.Type,
					Provider:     n.ResolvedProvider,
					Dependencies: stateDeps,
					State:        &state,
				},
			},

			// We clear the diff out here so that future nodes
			// don't see a diff that is already complete. There
			// is no longer a diff!
			&EvalWriteDiff{
				Name: stateId,
				Diff: nil,
			},

			&EvalApplyPost{
				Info:  info,
				State: &state,
				Error: &err,
			},
			&EvalUpdateStateHook{},
		},
	}
}
