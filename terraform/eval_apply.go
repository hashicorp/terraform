package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalApply is an EvalNode implementation that writes the diff to
// the full diff.
type EvalApply struct {
	Addr           addrs.ResourceInstance
	Config         *configs.Resource
	Dependencies   []addrs.Referenceable
	State          **states.ResourceInstanceObject
	Change         **plans.ResourceInstanceChange
	ProviderAddr   addrs.AbsProviderConfig
	Provider       *providers.Interface
	ProviderSchema **ProviderSchema
	Output         **states.ResourceInstanceObject
	CreateNew      *bool
	Error          *error
}

// TODO: test
func (n *EvalApply) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics

	change := *n.Change
	provider := *n.Provider
	state := *n.State
	absAddr := n.Addr.Absolute(ctx.Path())

	if state == nil {
		state = &states.ResourceInstanceObject{}
	}

	schema := (*n.ProviderSchema).ResourceTypes[n.Addr.Resource.Type]
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Type)
	}

	if n.CreateNew != nil {
		*n.CreateNew = (change.Action == plans.Create || change.Action == plans.Replace)
	}

	configVal := cty.NullVal(cty.DynamicPseudoType)
	if n.Config != nil {
		var configDiags tfdiags.Diagnostics
		keyData := EvalDataForInstanceKey(n.Addr.Key)
		configVal, _, configDiags = ctx.EvaluateBlock(n.Config.Config, schema, nil, keyData)
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return nil, diags.Err()
		}
	}

	log.Printf("[DEBUG] %s: applying the planned %s change", n.Addr.Absolute(ctx.Path()), change.Action)
	resp := provider.ApplyResourceChange(providers.ApplyResourceChangeRequest{
		TypeName:       n.Addr.Resource.Type,
		PriorState:     change.Before,
		Config:         configVal,
		PlannedState:   change.After,
		PlannedPrivate: change.Private,
	})
	applyDiags := resp.Diagnostics
	if n.Config != nil {
		applyDiags = applyDiags.InConfigBody(n.Config.Config)
	}
	diags = diags.Append(applyDiags)

	// Even if there are errors in the returned diagnostics, the provider may
	// have returned a _partial_ state for an object that already exists but
	// failed to fully configure, and so the remaining code must always run
	// to completion but must be defensive against the new value being
	// incomplete.
	newVal := resp.NewState

	// newVal should never be cty.NilVal in a real case, but it can happen
	// sometimes in sloppy mocks in tests where error diagnostics are returned
	// and the mock implementation doesn't populate the value at all.
	if newVal == cty.NilVal {
		newVal = cty.NullVal(schema.ImpliedType())
	}

	for _, err := range newVal.Type().TestConformance(schema.ImpliedType()) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q planned an invalid value after apply for %s. The result could not be saved.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ProviderAddr.ProviderConfig.Type, tfdiags.FormatErrorPrefixed(err, absAddr.String()),
			),
		))
	}
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	// By this point there must not be any unknown values remaining in our
	// object, because we've applied the change and we can't save unknowns
	// in our persistent state. If any are present then we will indicate an
	// error (which is always a bug in the provider) but we will also replace
	// them with nulls so that we can successfully save the portions of the
	// returned value that are known.
	if !newVal.IsWhollyKnown() {
		// To generate better error messages, we'll go for a walk through the
		// value and make a separate diagnostic for each unknown value we
		// find.
		cty.Walk(newVal, func(path cty.Path, val cty.Value) (bool, error) {
			if !val.IsKnown() {
				pathStr := tfdiags.FormatCtyPath(path)
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Provider returned invalid result object after apply",
					fmt.Sprintf(
						"After the apply operation, the provider still indicated an unknown value for %s%s. All values must be known after apply, so this is always a bug in the provider and should be reported in the provider's own repository. Terraform will still save the other known object values in the state.",
						n.Addr.Absolute(ctx.Path()), pathStr,
					),
				))
			}
			return true, nil
		})

		// NOTE: This operation can potentially be lossy if there are multiple
		// elements in a set that differ only by unknown values: after
		// replacing with null these will be merged together into a single set
		// element. Since we can only get here in the presence of a provider
		// bug, we accept this because storing a result here is always a
		// best-effort sort of thing.
		newVal = cty.UnknownAsNull(newVal)
	}

	// If a provider returns a null or non-null object at the wrong time then
	// we still want to save that but it often causes some confusing behaviors
	// where it seems like Terraform is failing to take any action at all,
	// so we'll generate some errors to draw attention to it.
	if change.Action == plans.Delete && !newVal.IsNull() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider returned invalid result object after apply",
			fmt.Sprintf(
				"After applying a %s plan, the provider returned a non-null object for %s. Destroying should always produce a null value, so this is always a bug in the provider and should be reported in the provider's own repository. Terraform will still save this errant object in the state for debugging and recovery.",
				change.Action, n.Addr.Absolute(ctx.Path()),
			),
		))
	}
	if change.Action != plans.Delete && newVal.IsNull() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider returned invalid result object after apply",
			fmt.Sprintf(
				"After applying a %s plan, the provider returned a null object for %s. Only destroying should always produce a null value, so this is always a bug in the provider and should be reported in the provider's own repository.",
				change.Action, n.Addr.Absolute(ctx.Path()),
			),
		))
	}

	var newState *states.ResourceInstanceObject
	if !newVal.IsNull() { // null value indicates that the object is deleted, so we won't set a new state in that case
		newState = &states.ResourceInstanceObject{
			Status:       states.ObjectReady,
			Value:        newVal,
			Private:      resp.Private,
			Dependencies: n.Dependencies, // Should be populated by the caller from the StateDependencies method on the resource instance node
		}
	}

	// Write the final state
	if n.Output != nil {
		*n.Output = newState
	}

	if diags.HasErrors() {
		// If the caller provided an error pointer then they are expected to
		// handle the error some other way and we treat our own result as
		// success.
		if n.Error != nil {
			err := diags.Err()
			*n.Error = err
			log.Printf("[DEBUG] %s: apply errored, but we're indicating that via the Error pointer rather than returning it: %s", n.Addr.Absolute(ctx.Path()), err)
			return nil, nil
		}
	}

	return nil, diags.ErrWithWarnings()
}

// EvalApplyPre is an EvalNode implementation that does the pre-Apply work
type EvalApplyPre struct {
	Addr   addrs.ResourceInstance
	Gen    states.Generation
	State  **states.ResourceInstanceObject
	Change **plans.ResourceInstanceChange
}

// TODO: test
func (n *EvalApplyPre) Eval(ctx EvalContext) (interface{}, error) {
	change := *n.Change
	absAddr := n.Addr.Absolute(ctx.Path())

	if resourceHasUserVisibleApply(n.Addr) {
		priorState := change.Before
		plannedNewState := change.After

		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreApply(absAddr, n.Gen, change.Action, priorState, plannedNewState)
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// EvalApplyPost is an EvalNode implementation that does the post-Apply work
type EvalApplyPost struct {
	Addr   addrs.ResourceInstance
	Gen    states.Generation
	State  **states.ResourceInstanceObject
	Error  *error
}

// TODO: test
func (n *EvalApplyPost) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State

	if resourceHasUserVisibleApply(n.Addr) {
		absAddr := n.Addr.Absolute(ctx.Path())
		var newState cty.Value
		if state != nil {
			newState = state.Value
		} else {
			newState = cty.NullVal(cty.DynamicPseudoType)
		}
		var err error
		if n.Error != nil {
			err = *n.Error
		}

		hookErr := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostApply(absAddr, n.Gen, newState, err)
		})
		if hookErr != nil {
			return nil, hookErr
		}
	}

	return nil, *n.Error
}

// EvalMaybeTainted is an EvalNode that takes the planned change, new value,
// and possible error from an apply operation and produces a new instance
// object marked as tainted if it appears that a create operation has failed.
//
// This EvalNode never returns an error, to ensure that a subsequent EvalNode
// can still record the possibly-tainted object in the state.
type EvalMaybeTainted struct {
	Addr   addrs.ResourceInstance
	Gen    states.Generation
	Change **plans.ResourceInstanceChange
	State  **states.ResourceInstanceObject
	Error  *error

	// If StateOutput is not nil, its referent will be assigned either the same
	// pointer as State or a new object with its status set as Tainted,
	// depending on whether an error is given and if this was a create action.
	StateOutput **states.ResourceInstanceObject
}

// TODO: test
func (n *EvalMaybeTainted) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State
	change := *n.Change
	err := *n.Error

	if state != nil && state.Status == states.ObjectTainted {
		log.Printf("[TRACE] EvalMaybeTainted: %s was already tainted, so nothing to do", n.Addr.Absolute(ctx.Path()))
		return nil, nil
	}

	if n.StateOutput != nil {
		if err != nil && change.Action == plans.Create {
			// If there are errors during a _create_ then the object is
			// in an undefined state, and so we'll mark it as tainted so
			// we can try again on the next run.
			//
			// We don't do this for other change actions because errors
			// during updates will often not change the remote object at all.
			// If there _were_ changes prior to the error, it's the provider's
			// responsibility to record the effect of those changes in the
			// object value it returned.
			log.Printf("[TRACE] EvalMaybeTainted: %s encountered an error during creation, so it is now marked as tainted", n.Addr.Absolute(ctx.Path()))
			*n.StateOutput = state.AsTainted()
		} else {
			*n.StateOutput = state
		}
	}

	return nil, nil
}

// resourceHasUserVisibleApply returns true if the given resource is one where
// apply actions should be exposed to the user.
//
// Certain resources do apply actions only as an implementation detail, so
// these should not be advertised to code outside of this package.
func resourceHasUserVisibleApply(addr addrs.ResourceInstance) bool {
	// Only managed resources have user-visible apply actions.
	// In particular, this excludes data resources since we "apply" these
	// only as an implementation detail of removing them from state when
	// they are destroyed. (When reading, they don't get here at all because
	// we present them as "Refresh" actions.)
	return addr.ContainingResource().Mode == addrs.ManagedResourceMode
}

// EvalApplyProvisioners is an EvalNode implementation that executes
// the provisioners for a resource.
//
// TODO(mitchellh): This should probably be split up into a more fine-grained
// ApplyProvisioner (single) that is looped over.
type EvalApplyProvisioners struct {
	Addr           addrs.ResourceInstance
	State          **states.ResourceInstanceObject
	ResourceConfig *configs.Resource
	CreateNew      *bool
	Error          *error

	// When is the type of provisioner to run at this point
	When configs.ProvisionerWhen
}

// TODO: test
func (n *EvalApplyProvisioners) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())
	state := *n.State
	if state == nil {
		log.Printf("[TRACE] EvalApplyProvisioners: %s has no state, so skipping provisioners", n.Addr)
		return nil, nil
	}
	if n.CreateNew != nil && !*n.CreateNew {
		// If we're not creating a new resource, then don't run provisioners
		log.Printf("[TRACE] EvalApplyProvisioners: %s is not freshly-created, so no provisioning is required", n.Addr)
		return nil, nil
	}
	if state.Status == states.ObjectTainted {
		// No point in provisioning an object that is already tainted, since
		// it's going to get recreated on the next apply anyway.
		log.Printf("[TRACE] EvalApplyProvisioners: %s is tainted, so skipping provisioning", n.Addr)
		return nil, nil
	}

	provs := n.filterProvisioners()
	if len(provs) == 0 {
		// We have no provisioners, so don't do anything
		return nil, nil
	}

	if n.Error != nil && *n.Error != nil {
		// We're already tainted, so just return out
		return nil, nil
	}

	{
		// Call pre hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreProvisionInstance(absAddr, state.Value)
		})
		if err != nil {
			return nil, err
		}
	}

	// If there are no errors, then we append it to our output error
	// if we have one, otherwise we just output it.
	err := n.apply(ctx, provs)
	if err != nil {
		*n.Error = multierror.Append(*n.Error, err)
		if n.Error == nil {
			return nil, err
		} else {
			log.Printf("[TRACE] EvalApplyProvisioners: %s provisioning failed, but we will continue anyway at the caller's request", absAddr)
			return nil, nil
		}
	}

	{
		// Call post hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostProvisionInstance(absAddr, state.Value)
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// filterProvisioners filters the provisioners on the resource to only
// the provisioners specified by the "when" option.
func (n *EvalApplyProvisioners) filterProvisioners() []*configs.Provisioner {
	// Fast path the zero case
	if n.ResourceConfig == nil || n.ResourceConfig.Managed == nil {
		return nil
	}

	if len(n.ResourceConfig.Managed.Provisioners) == 0 {
		return nil
	}

	result := make([]*configs.Provisioner, 0, len(n.ResourceConfig.Managed.Provisioners))
	for _, p := range n.ResourceConfig.Managed.Provisioners {
		if p.When == n.When {
			result = append(result, p)
		}
	}

	return result
}

func (n *EvalApplyProvisioners) apply(ctx EvalContext, provs []*configs.Provisioner) error {
	var diags tfdiags.Diagnostics
	instanceAddr := n.Addr
	absAddr := instanceAddr.Absolute(ctx.Path())

	// If there's a connection block defined directly inside the resource block
	// then it'll serve as a base connection configuration for all of the
	// provisioners.
	var baseConn hcl.Body
	if n.ResourceConfig.Managed != nil && n.ResourceConfig.Managed.Connection != nil {
		baseConn = n.ResourceConfig.Managed.Connection.Config
	}

	for _, prov := range provs {
		log.Printf("[TRACE] EvalApplyProvisioners: provisioning %s with %q", absAddr, prov.Type)

		// Get the provisioner
		provisioner := ctx.Provisioner(prov.Type)
		schema := ctx.ProvisionerSchema(prov.Type)

		keyData := EvalDataForInstanceKey(instanceAddr.Key)

		// Evaluate the main provisioner configuration.
		config, _, configDiags := ctx.EvaluateBlock(prov.Config, schema, instanceAddr, keyData)
		diags = diags.Append(configDiags)

		// If the provisioner block contains a connection block of its own then
		// it can override the base connection configuration, if any.
		var localConn hcl.Body
		if prov.Connection != nil {
			localConn = prov.Connection.Config
		}

		var connBody hcl.Body
		switch {
		case baseConn != nil && localConn != nil:
			// Our standard merging logic applies here, similar to what we do
			// with _override.tf configuration files: arguments from the
			// base connection block will be masked by any arguments of the
			// same name in the local connection block.
			connBody = configs.MergeBodies(baseConn, localConn)
		case baseConn != nil:
			connBody = baseConn
		case localConn != nil:
			connBody = localConn
		default: // both are nil, by elimination
			connBody = hcl.EmptyBody()
		}

		connInfo, _, connInfoDiags := ctx.EvaluateBlock(connBody, connectionBlockSupersetSchema, instanceAddr, keyData)
		diags = diags.Append(connInfoDiags)
		if diags.HasErrors() {
			// "on failure continue" setting only applies to failures of the
			// provisioner itself, not to invalid configuration.
			return diags.Err()
		}

		{
			// Call pre hook
			err := ctx.Hook(func(h Hook) (HookAction, error) {
				return h.PreProvisionInstanceStep(absAddr, prov.Type)
			})
			if err != nil {
				return err
			}
		}

		// The output function
		outputFn := func(msg string) {
			ctx.Hook(func(h Hook) (HookAction, error) {
				h.ProvisionOutput(absAddr, prov.Type, msg)
				return HookActionContinue, nil
			})
		}

		output := CallbackUIOutput{OutputFn: outputFn}
		resp := provisioner.ProvisionResource(provisioners.ProvisionResourceRequest{
			Config:     config,
			Connection: connInfo,
			UIOutput:   &output,
		})
		applyDiags := resp.Diagnostics.InConfigBody(prov.Config)

		// Call post hook
		hookErr := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostProvisionInstanceStep(absAddr, prov.Type, applyDiags.Err())
		})

		switch prov.OnFailure {
		case configs.ProvisionerOnFailureContinue:
			if applyDiags.HasErrors() {
				log.Printf("[WARN] Errors while provisioning %s with %q, but continuing as requested in configuration", n.Addr, prov.Type)
			} else {
				// Maybe there are warnings that we still want to see
				diags = diags.Append(applyDiags)
			}
		default:
			diags = diags.Append(applyDiags)
			if applyDiags.HasErrors() {
				log.Printf("[WARN] Errors while provisioning %s with %q, so aborting", n.Addr, prov.Type)
				return diags.Err()
			}
		}

		// Deal with the hook
		if hookErr != nil {
			return hookErr
		}
	}

	return diags.ErrWithWarnings()
}
