package terraform

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/objchange"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalApply is an EvalNode implementation that writes the diff to
// the full diff.
type EvalApply struct {
	Addr                addrs.ResourceInstance
	Config              *configs.Resource
	State               **states.ResourceInstanceObject
	Change              **plans.ResourceInstanceChange
	ProviderAddr        addrs.AbsProviderConfig
	Provider            *providers.Interface
	ProviderMetas       map[addrs.Provider]*configs.ProviderMeta
	ProviderSchema      **ProviderSchema
	Output              **states.ResourceInstanceObject
	CreateNew           *bool
	Error               *error
	CreateBeforeDestroy bool
}

// TODO: test
func (n *EvalApply) Eval(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	change := *n.Change
	provider := *n.Provider
	state := *n.State
	absAddr := n.Addr.Absolute(ctx.Path())

	if state == nil {
		state = &states.ResourceInstanceObject{}
	}

	schema, _ := (*n.ProviderSchema).SchemaForResourceType(n.Addr.Resource.Mode, n.Addr.Resource.Type)
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Type))
		return diags
	}

	if n.CreateNew != nil {
		*n.CreateNew = (change.Action == plans.Create || change.Action.IsReplace())
	}

	configVal := cty.NullVal(cty.DynamicPseudoType)
	if n.Config != nil {
		var configDiags tfdiags.Diagnostics
		forEach, _ := evaluateForEachExpression(n.Config.ForEach, ctx)
		keyData := EvalDataForInstanceKey(n.Addr.Key, forEach)
		configVal, _, configDiags = ctx.EvaluateBlock(n.Config.Config, schema, nil, keyData)
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			return diags
		}
	}

	if !configVal.IsWhollyKnown() {
		diags = diags.Append(fmt.Errorf(
			"configuration for %s still contains unknown values during apply (this is a bug in Terraform; please report it!)",
			absAddr,
		))
		return diags
	}

	metaConfigVal := cty.NullVal(cty.DynamicPseudoType)
	if n.ProviderMetas != nil {
		log.Printf("[DEBUG] EvalApply: ProviderMeta config value set")
		if m, ok := n.ProviderMetas[n.ProviderAddr.Provider]; ok && m != nil {
			// if the provider doesn't support this feature, throw an error
			if (*n.ProviderSchema).ProviderMeta == nil {
				log.Printf("[DEBUG] EvalApply: no ProviderMeta schema")
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Provider %s doesn't support provider_meta", n.ProviderAddr.Provider.String()),
					Detail:   fmt.Sprintf("The resource %s belongs to a provider that doesn't support provider_meta blocks", n.Addr),
					Subject:  &m.ProviderRange,
				})
			} else {
				log.Printf("[DEBUG] EvalApply: ProviderMeta schema found")
				var configDiags tfdiags.Diagnostics
				metaConfigVal, _, configDiags = ctx.EvaluateBlock(m.Config, (*n.ProviderSchema).ProviderMeta, nil, EvalDataForNoInstanceKey)
				diags = diags.Append(configDiags)
				if configDiags.HasErrors() {
					return diags
				}
			}
		}
	}

	log.Printf("[DEBUG] %s: applying the planned %s change", n.Addr.Absolute(ctx.Path()), change.Action)

	// If our config, Before or After value contain any marked values,
	// ensure those are stripped out before sending
	// this to the provider
	unmarkedConfigVal, _ := configVal.UnmarkDeep()
	unmarkedBefore, beforePaths := change.Before.UnmarkDeepWithPaths()
	unmarkedAfter, afterPaths := change.After.UnmarkDeepWithPaths()

	// If we have an Update action, our before and after values are equal,
	// and only differ on their sensitivity, the newVal is the after val
	// and we should not communicate with the provider. We do need to update
	// the state with this new value, to ensure the sensitivity change is
	// persisted.
	eqV := unmarkedBefore.Equals(unmarkedAfter)
	eq := eqV.IsKnown() && eqV.True()
	if change.Action == plans.Update && eq && !reflect.DeepEqual(beforePaths, afterPaths) {
		// Copy the previous state, changing only the value
		newState := &states.ResourceInstanceObject{
			CreateBeforeDestroy: state.CreateBeforeDestroy,
			Dependencies:        state.Dependencies,
			Private:             state.Private,
			Status:              state.Status,
			Value:               change.After,
		}

		// Write the final state
		if n.Output != nil {
			*n.Output = newState
		}

		return diags
	}

	resp := provider.ApplyResourceChange(providers.ApplyResourceChangeRequest{
		TypeName:       n.Addr.Resource.Type,
		PriorState:     unmarkedBefore,
		Config:         unmarkedConfigVal,
		PlannedState:   unmarkedAfter,
		PlannedPrivate: change.Private,
		ProviderMeta:   metaConfigVal,
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

	// If we have paths to mark, mark those on this new value
	if len(afterPaths) > 0 {
		newVal = newVal.MarkWithPaths(afterPaths)
	}

	if newVal == cty.NilVal {
		// Providers are supposed to return a partial new value even when errors
		// occur, but sometimes they don't and so in that case we'll patch that up
		// by just using the prior state, so we'll at least keep track of the
		// object for the user to retry.
		newVal = change.Before

		// As a special case, we'll set the new value to null if it looks like
		// we were trying to execute a delete, because the provider in this case
		// probably left the newVal unset intending it to be interpreted as "null".
		if change.After.IsNull() {
			newVal = cty.NullVal(schema.ImpliedType())
		}

		// Ideally we'd produce an error or warning here if newVal is nil and
		// there are no errors in diags, because that indicates a buggy
		// provider not properly reporting its result, but unfortunately many
		// of our historical test mocks behave in this way and so producing
		// a diagnostic here fails hundreds of tests. Instead, we must just
		// silently retain the old value for now. Returning a nil value with
		// no errors is still always considered a bug in the provider though,
		// and should be fixed for any "real" providers that do it.
	}

	var conformDiags tfdiags.Diagnostics
	for _, err := range newVal.Type().TestConformance(schema.ImpliedType()) {
		conformDiags = conformDiags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid object",
			fmt.Sprintf(
				"Provider %q produced an invalid value after apply for %s. The result cannot not be saved in the Terraform state.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ProviderAddr.Provider.String(), tfdiags.FormatErrorPrefixed(err, absAddr.String()),
			),
		))
	}
	diags = diags.Append(conformDiags)
	if conformDiags.HasErrors() {
		// Bail early in this particular case, because an object that doesn't
		// conform to the schema can't be saved in the state anyway -- the
		// serializer will reject it.
		return diags
	}

	// After this point we have a type-conforming result object and so we
	// must always run to completion to ensure it can be saved. If n.Error
	// is set then we must not return a non-nil error, in order to allow
	// evaluation to continue to a later point where our state object will
	// be saved.

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

	if change.Action != plans.Delete && !diags.HasErrors() {
		// Only values that were marked as unknown in the planned value are allowed
		// to change during the apply operation. (We do this after the unknown-ness
		// check above so that we also catch anything that became unknown after
		// being known during plan.)
		//
		// If we are returning other errors anyway then we'll give this
		// a pass since the other errors are usually the explanation for
		// this one and so it's more helpful to let the user focus on the
		// root cause rather than distract with this extra problem.
		if errs := objchange.AssertObjectCompatible(schema, change.After, newVal); len(errs) > 0 {
			if resp.LegacyTypeSystem {
				// The shimming of the old type system in the legacy SDK is not precise
				// enough to pass this consistency check, so we'll give it a pass here,
				// but we will generate a warning about it so that we are more likely
				// to notice in the logs if an inconsistency beyond the type system
				// leads to a downstream provider failure.
				var buf strings.Builder
				fmt.Fprintf(&buf, "[WARN] Provider %q produced an unexpected new value for %s, but we are tolerating it because it is using the legacy plugin SDK.\n    The following problems may be the cause of any confusing errors from downstream operations:", n.ProviderAddr.Provider.String(), absAddr)
				for _, err := range errs {
					fmt.Fprintf(&buf, "\n      - %s", tfdiags.FormatError(err))
				}
				log.Print(buf.String())

				// The sort of inconsistency we won't catch here is if a known value
				// in the plan is changed during apply. That can cause downstream
				// problems because a dependent resource would make its own plan based
				// on the planned value, and thus get a different result during the
				// apply phase. This will usually lead to a "Provider produced invalid plan"
				// error that incorrectly blames the downstream resource for the change.

			} else {
				for _, err := range errs {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Provider produced inconsistent result after apply",
						fmt.Sprintf(
							"When applying changes to %s, provider %q produced an unexpected new value: %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
							absAddr, n.ProviderAddr.Provider.String(), tfdiags.FormatError(err),
						),
					))
				}
			}
		}
	}

	// If a provider returns a null or non-null object at the wrong time then
	// we still want to save that but it often causes some confusing behaviors
	// where it seems like Terraform is failing to take any action at all,
	// so we'll generate some errors to draw attention to it.
	if !diags.HasErrors() {
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
	}

	newStatus := states.ObjectReady

	// Sometimes providers return a null value when an operation fails for some
	// reason, but we'd rather keep the prior state so that the error can be
	// corrected on a subsequent run. We must only do this for null new value
	// though, or else we may discard partial updates the provider was able to
	// complete.
	if diags.HasErrors() && newVal.IsNull() {
		// Otherwise, we'll continue but using the prior state as the new value,
		// making this effectively a no-op. If the item really _has_ been
		// deleted then our next refresh will detect that and fix it up.
		// If change.Action is Create then change.Before will also be null,
		// which is fine.
		newVal = change.Before

		// If we're recovering the previous state, we also want to restore the
		// the tainted status of the object.
		if state.Status == states.ObjectTainted {
			newStatus = states.ObjectTainted
		}
	}

	var newState *states.ResourceInstanceObject
	if !newVal.IsNull() { // null value indicates that the object is deleted, so we won't set a new state in that case
		newState = &states.ResourceInstanceObject{
			Status:              newStatus,
			Value:               newVal,
			Private:             resp.Private,
			CreateBeforeDestroy: n.CreateBeforeDestroy,
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
			return nil
		}
	}

	return diags
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
func (n *EvalApplyProvisioners) Eval(ctx EvalContext) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	absAddr := n.Addr.Absolute(ctx.Path())
	state := *n.State
	if state == nil {
		log.Printf("[TRACE] EvalApplyProvisioners: %s has no state, so skipping provisioners", n.Addr)
		return nil
	}
	if n.When == configs.ProvisionerWhenCreate && n.CreateNew != nil && !*n.CreateNew {
		// If we're not creating a new resource, then don't run provisioners
		log.Printf("[TRACE] EvalApplyProvisioners: %s is not freshly-created, so no provisioning is required", n.Addr)
		return nil
	}
	if state.Status == states.ObjectTainted {
		// No point in provisioning an object that is already tainted, since
		// it's going to get recreated on the next apply anyway.
		log.Printf("[TRACE] EvalApplyProvisioners: %s is tainted, so skipping provisioning", n.Addr)
		return nil
	}

	provs := n.filterProvisioners()
	if len(provs) == 0 {
		// We have no provisioners, so don't do anything
		return nil
	}

	if n.Error != nil && *n.Error != nil {
		// We're already tainted, so just return out
		return nil
	}

	// Call pre hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreProvisionInstance(absAddr, state.Value)
	}))
	if diags.HasErrors() {
		return diags
	}

	// If there are no errors, then we append it to our output error
	// if we have one, otherwise we just output it.
	err := n.apply(ctx, provs)
	if err != nil {
		*n.Error = multierror.Append(*n.Error, err)
		log.Printf("[TRACE] EvalApplyProvisioners: %s provisioning failed, but we will continue anyway at the caller's request", absAddr)
		return nil
	}

	// Call post hook
	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostProvisionInstance(absAddr, state.Value)
	}))
	if diags.HasErrors() {
		return diags
	}

	return diags
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

	// this self is only used for destroy provisioner evaluation, and must
	// refer to the last known value of the resource.
	self := (*n.State).Value

	var evalScope func(EvalContext, hcl.Body, cty.Value, *configschema.Block) (cty.Value, tfdiags.Diagnostics)
	switch n.When {
	case configs.ProvisionerWhenDestroy:
		evalScope = n.evalDestroyProvisionerConfig
	default:
		evalScope = n.evalProvisionerConfig
	}

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

		config, configDiags := evalScope(ctx, prov.Config, self, schema)
		diags = diags.Append(configDiags)
		if diags.HasErrors() {
			return diags.Err()
		}

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
		}

		// start with an empty connInfo
		connInfo := cty.NullVal(connectionBlockSupersetSchema.ImpliedType())

		if connBody != nil {
			var connInfoDiags tfdiags.Diagnostics
			connInfo, connInfoDiags = evalScope(ctx, connBody, self, connectionBlockSupersetSchema)
			diags = diags.Append(connInfoDiags)
			if diags.HasErrors() {
				return diags.Err()
			}
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

		// If our config or connection info contains any marked values, ensure
		// those are stripped out before sending to the provisioner. Unlike
		// resources, we have no need to capture the marked paths and reapply
		// later.
		unmarkedConfig, configMarks := config.UnmarkDeep()
		unmarkedConnInfo, _ := connInfo.UnmarkDeep()

		// Marks on the config might result in leaking sensitive values through
		// provisioner logging, so we conservatively suppress all output in
		// this case. This should not apply to connection info values, which
		// provisioners ought not to be logging anyway.
		if len(configMarks) > 0 {
			outputFn = func(msg string) {
				ctx.Hook(func(h Hook) (HookAction, error) {
					h.ProvisionOutput(absAddr, prov.Type, "(output suppressed due to sensitive value in config)")
					return HookActionContinue, nil
				})
			}
		}

		output := CallbackUIOutput{OutputFn: outputFn}
		resp := provisioner.ProvisionResource(provisioners.ProvisionResourceRequest{
			Config:     unmarkedConfig,
			Connection: unmarkedConnInfo,
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

	// we have to drop warning-only diagnostics for now
	if diags.HasErrors() {
		return diags.ErrWithWarnings()
	}

	// log any warnings since we can't return them
	if e := diags.ErrWithWarnings(); e != nil {
		log.Printf("[WARN] EvalApplyProvisioners %s: %v", n.Addr, e)
	}

	return nil
}

func (n *EvalApplyProvisioners) evalProvisionerConfig(ctx EvalContext, body hcl.Body, self cty.Value, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	forEach, forEachDiags := evaluateForEachExpression(n.ResourceConfig.ForEach, ctx)
	diags = diags.Append(forEachDiags)

	keyData := EvalDataForInstanceKey(n.Addr.Key, forEach)

	config, _, configDiags := ctx.EvaluateBlock(body, schema, n.Addr, keyData)
	diags = diags.Append(configDiags)

	return config, diags
}

// during destroy a provisioner can only evaluate within the scope of the parent resource
func (n *EvalApplyProvisioners) evalDestroyProvisionerConfig(ctx EvalContext, body hcl.Body, self cty.Value, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// For a destroy-time provisioner forEach is intentionally nil here,
	// which EvalDataForInstanceKey responds to by not populating EachValue
	// in its result. That's okay because each.value is prohibited for
	// destroy-time provisioners.
	keyData := EvalDataForInstanceKey(n.Addr.Key, nil)

	evalScope := ctx.EvaluationScope(n.Addr, keyData)
	config, evalDiags := evalScope.EvalSelfBlock(body, self, schema, keyData)
	diags = diags.Append(evalDiags)

	return config, diags
}
