package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalApply is an EvalNode implementation that writes the diff to
// the full diff.
type EvalApply struct {
	Addr           addrs.ResourceInstance
	Config         *configs.Resource
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

	configVal := cty.NullVal(cty.DynamicPseudoType) // TODO: Populate this when n.Config is non-nil; will need config and provider schema in here

	log.Printf("[DEBUG] %s: applying the planned %s change", n.Addr.Absolute(ctx.Path()), change.Action)
	resp := provider.ApplyResourceChange(providers.ApplyResourceChangeRequest{
		TypeName:       n.Addr.Resource.Type,
		PriorState:     change.Before,
		Config:         configVal, // TODO
		PlannedState:   change.After,
		PlannedPrivate: change.Private, // TODO
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

	for _, err := range schema.ImpliedType().TestConformance(newVal.Type()) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q planned an invalid value for %s%s after apply, and so the result could not be saved.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ProviderAddr.ProviderConfig.Type, absAddr, tfdiags.FormatError(err),
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

	var newState *states.ResourceInstanceObject
	if !newVal.IsNull() { // null value indicates that the object is deleted, so we won't set a new state in that case
		newState = &states.ResourceInstanceObject{
			Status:       states.ObjectReady, // TODO: Consider marking as tainted if the provider returned errors?
			Value:        newVal,
			Private:      resp.Private,
			Dependencies: nil, // not populated here; this will be mutated by a later eval step
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
			n.Error = &err
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
	Addr  addrs.ResourceInstance
	Gen   states.Generation
	State **states.ResourceInstanceObject
	Error *error
}

// TODO: test
func (n *EvalApplyPost) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State

	if resourceHasUserVisibleApply(n.Addr) {
		absAddr := n.Addr.Absolute(ctx.Path())
		newState := state.Value
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
		return nil, nil
	}

	provs := n.filterProvisioners()
	if len(provs) == 0 {
		// We have no provisioners, so don't do anything
		return nil, nil
	}

	// taint tells us whether to enable tainting.
	taint := n.When == configs.ProvisionerWhenCreate

	if n.Error != nil && *n.Error != nil {
		if taint {
			state.Status = states.ObjectTainted
		}

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
		if taint {
			state.Status = states.ObjectTainted
		}

		*n.Error = multierror.Append(*n.Error, err)
		return nil, err
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
	return fmt.Errorf("EvalApplyProvisioners.apply not yet updated for new types")
	/*
		instanceAddr := n.Addr
		absAddr := instanceAddr.Absolute(ctx.Path())
		state := *n.State

		// The hook API still uses the legacy InstanceInfo type, so we need to shim it.
		legacyInfo := NewInstanceInfo(n.Addr.Absolute(ctx.Path()))

		// Store the original connection info, restore later
		origConnInfo := state.Ephemeral.ConnInfo
		defer func() {
			state.Ephemeral.ConnInfo = origConnInfo
		}()

		var diags tfdiags.Diagnostics

		for _, prov := range provs {
			// Get the provisioner
			provisioner := ctx.Provisioner(prov.Type)
			schema := ctx.ProvisionerSchema(prov.Type)

			keyData := EvalDataForInstanceKey(instanceAddr.Key)

			// Evaluate the main provisioner configuration.
			config, _, configDiags := ctx.EvaluateBlock(prov.Config, schema, instanceAddr, keyData)
			diags = diags.Append(configDiags)

			// A provisioner may not have a connection block
			if prov.Connection != nil {
				connInfo, _, connInfoDiags := ctx.EvaluateBlock(prov.Connection.Config, connectionBlockSupersetSchema, instanceAddr, keyData)
				diags = diags.Append(connInfoDiags)

				if configDiags.HasErrors() || connInfoDiags.HasErrors() {
					continue
				}

				// Merge the connection information, and also lower everything to strings
				// for compatibility with the communicator API.
				overlay := make(map[string]string)
				if origConnInfo != nil {
					for k, v := range origConnInfo {
						overlay[k] = v
					}
				}
				for it := connInfo.ElementIterator(); it.Next(); {
					kv, vv := it.Element()
					var k, v string

					// there are no unset or null values in a connection block, and
					// everything needs to map to a string.
					if vv.IsNull() {
						continue
					}

					err := gocty.FromCtyValue(kv, &k)
					if err != nil {
						// Should never happen, because connectionBlockSupersetSchema requires all primitives
						panic(err)
					}
					err = gocty.FromCtyValue(vv, &v)
					if err != nil {
						// Should never happen, because connectionBlockSupersetSchema requires all primitives
						panic(err)
					}

					overlay[k] = v
				}

				state.Ephemeral.ConnInfo = overlay
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

			// The provisioner API still uses our legacy ResourceConfig type, so
			// we need to shim it.
			legacyRC := NewResourceConfigShimmed(config, schema)

			// Invoke the Provisioner
			output := CallbackUIOutput{OutputFn: outputFn}
			applyErr := provisioner.Apply(&output, state, legacyRC)

			// Call post hook
			hookErr := ctx.Hook(func(h Hook) (HookAction, error) {
				return h.PostProvisionInstanceStep(absAddr, prov.Type, applyErr)
			})

			// Handle the error before we deal with the hook
			if applyErr != nil {
				// Determine failure behavior
				switch prov.OnFailure {
				case configs.ProvisionerOnFailureContinue:
					log.Printf("[INFO] apply %s [%s]: error during provision, but continuing as requested in configuration", n.Addr, prov.Type)
				case configs.ProvisionerOnFailureFail:
					return applyErr
				}
			}

			// Deal with the hook
			if hookErr != nil {
				return hookErr
			}
		}

		return diags.ErrWithWarnings()
	*/
}
