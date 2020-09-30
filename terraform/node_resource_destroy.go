package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/plans"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/states"
)

// NodeDestroyResourceInstance represents a resource instance that is to be
// destroyed.
type NodeDestroyResourceInstance struct {
	*NodeAbstractResourceInstance

	// If DeposedKey is set to anything other than states.NotDeposed then
	// this node destroys a deposed object of the associated instance
	// rather than its current object.
	DeposedKey states.DeposedKey
}

var (
	_ GraphNodeModuleInstance      = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeConfigResource      = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeResourceInstance    = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeDestroyer           = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeDestroyerCBD        = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeReferenceable       = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeReferencer          = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeExecutable          = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeProviderConsumer    = (*NodeDestroyResourceInstance)(nil)
	_ GraphNodeProvisionerConsumer = (*NodeDestroyResourceInstance)(nil)
)

func (n *NodeDestroyResourceInstance) Name() string {
	if n.DeposedKey != states.NotDeposed {
		return fmt.Sprintf("%s (destroy deposed %s)", n.ResourceInstanceAddr(), n.DeposedKey)
	}
	return n.ResourceInstanceAddr().String() + " (destroy)"
}

// GraphNodeDestroyer
func (n *NodeDestroyResourceInstance) DestroyAddr() *addrs.AbsResourceInstance {
	addr := n.ResourceInstanceAddr()
	return &addr
}

// GraphNodeDestroyerCBD
func (n *NodeDestroyResourceInstance) CreateBeforeDestroy() bool {
	// State takes precedence during destroy.
	// If the resource was removed, there is no config to check.
	// If CBD was forced from descendent, it should be saved in the state
	// already.
	if s := n.instanceState; s != nil {
		if s.Current != nil {
			return s.Current.CreateBeforeDestroy
		}
	}

	if n.Config != nil && n.Config.Managed != nil {
		return n.Config.Managed.CreateBeforeDestroy
	}

	return false
}

// GraphNodeDestroyerCBD
func (n *NodeDestroyResourceInstance) ModifyCreateBeforeDestroy(v bool) error {
	return nil
}

// GraphNodeReferenceable, overriding NodeAbstractResource
func (n *NodeDestroyResourceInstance) ReferenceableAddrs() []addrs.Referenceable {
	normalAddrs := n.NodeAbstractResourceInstance.ReferenceableAddrs()
	destroyAddrs := make([]addrs.Referenceable, len(normalAddrs))

	phaseType := addrs.ResourceInstancePhaseDestroy
	if n.CreateBeforeDestroy() {
		phaseType = addrs.ResourceInstancePhaseDestroyCBD
	}

	for i, normalAddr := range normalAddrs {
		switch ta := normalAddr.(type) {
		case addrs.Resource:
			destroyAddrs[i] = ta.Phase(phaseType)
		case addrs.ResourceInstance:
			destroyAddrs[i] = ta.Phase(phaseType)
		default:
			destroyAddrs[i] = normalAddr
		}
	}

	return destroyAddrs
}

// GraphNodeReferencer, overriding NodeAbstractResource
func (n *NodeDestroyResourceInstance) References() []*addrs.Reference {
	// If we have a config, then we need to include destroy-time dependencies
	if c := n.Config; c != nil && c.Managed != nil {
		var result []*addrs.Reference

		// We include conn info and config for destroy time provisioners
		// as dependencies that we have.
		for _, p := range c.Managed.Provisioners {
			schema := n.ProvisionerSchemas[p.Type]

			if p.When == configs.ProvisionerWhenDestroy {
				if p.Connection != nil {
					result = append(result, ReferencesFromConfig(p.Connection.Config, connectionBlockSupersetSchema)...)
				}
				result = append(result, ReferencesFromConfig(p.Config, schema)...)
			}
		}

		return result
	}

	return nil
}

// GraphNodeExecutable
func (n *NodeDestroyResourceInstance) Execute(ctx EvalContext, op walkOperation) error {
	addr := n.ResourceInstanceAddr()

	// Get our state
	is := n.instanceState
	if is == nil {
		log.Printf("[WARN] NodeDestroyResourceInstance for %s with no state", addr)
	}

	// These vars are updated through pointers at various stages below.
	var changeApply *plans.ResourceInstanceChange
	var state *states.ResourceInstanceObject
	var provisionerErr error

	switch op {
	case walkApply, walkDestroy:
		provider, providerSchema, err := GetProvider(ctx, n.ResolvedProvider)
		if err != nil {
			return err
		}

		changeApply, err = n.ReadDiff(ctx, providerSchema)
		if err != nil {
			return err
		}

		evalReduceDiff := &EvalReduceDiff{
			Addr:      addr.Resource,
			InChange:  &changeApply,
			Destroy:   true,
			OutChange: &changeApply,
		}
		_, err = evalReduceDiff.Eval(ctx)
		if err != nil {
			return err
		}

		// EvalReduceDiff may have simplified our planned change
		// into a NoOp if it does not require destroying.
		if changeApply == nil || changeApply.Action == plans.NoOp {
			return EvalEarlyExitError{}
		}

		state, err = n.ReadResourceInstanceState(ctx, addr)
		if err != nil {
			return err
		}

		// Exit early if the state object is null after reading the state
		if state == nil || state.Value.IsNull() {
			return EvalEarlyExitError{}
		}

		evalApplyPre := &EvalApplyPre{
			Addr:   addr.Resource,
			State:  &state,
			Change: &changeApply,
		}
		_, err = evalApplyPre.Eval(ctx)
		if err != nil {
			return err
		}

		// Run destroy provisioners if not tainted
		if state != nil && state.Status != states.ObjectTainted {
			evalApplyProvisioners := &EvalApplyProvisioners{
				Addr:           addr.Resource,
				State:          &state,
				ResourceConfig: n.Config,
				Error:          &provisionerErr,
				When:           configs.ProvisionerWhenDestroy,
			}
			_, err := evalApplyProvisioners.Eval(ctx)
			if err != nil {
				return err
			}
			if provisionerErr != nil {
				// If we have a provisioning error, then we just call
				// the post-apply hook now.
				evalApplyPost := &EvalApplyPost{
					Addr:  addr.Resource,
					State: &state,
					Error: &provisionerErr,
				}
				_, err = evalApplyPost.Eval(ctx)
				if err != nil {
					return err
				}
			}
		}

		// Managed resources need to be destroyed, while data sources
		// are only removed from state.
		if addr.Resource.Resource.Mode == addrs.ManagedResourceMode {
			evalApply := &EvalApply{
				Addr:           addr.Resource,
				Config:         nil, // No configuration because we are destroying
				State:          &state,
				Change:         &changeApply,
				Provider:       &provider,
				ProviderAddr:   n.ResolvedProvider,
				ProviderMetas:  n.ProviderMetas,
				ProviderSchema: &providerSchema,
				Output:         &state,
				Error:          &provisionerErr,
			}
			_, err = evalApply.Eval(ctx)
			if err != nil {
				return err
			}

			evalWriteState := &EvalWriteState{
				Addr:           addr.Resource,
				ProviderAddr:   n.ResolvedProvider,
				ProviderSchema: &providerSchema,
				State:          &state,
			}
			_, err = evalWriteState.Eval(ctx)
			if err != nil {
				return err
			}
		} else {
			log.Printf("[TRACE] NodeDestroyResourceInstance: removing state object for %s", n.Addr)
			state := ctx.State()
			state.SetResourceInstanceCurrent(n.Addr, nil, n.ResolvedProvider)
		}

		evalApplyPost := &EvalApplyPost{
			Addr:  addr.Resource,
			State: &state,
			Error: &provisionerErr,
		}
		_, err = evalApplyPost.Eval(ctx)
		if err != nil {
			return err
		}

		err = UpdateStateHook(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
