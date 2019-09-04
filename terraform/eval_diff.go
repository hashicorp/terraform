package terraform

import (
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/objchange"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalCheckPlannedChange is an EvalNode implementation that produces errors
// if the _actual_ expected value is not compatible with what was recorded
// in the plan.
//
// Errors here are most often indicative of a bug in the provider, so our
// error messages will report with that in mind. It's also possible that
// there's a bug in Terraform's Core's own "proposed new value" code in
// EvalDiff.
type EvalCheckPlannedChange struct {
	Addr           addrs.ResourceInstance
	ProviderAddr   addrs.AbsProviderConfig
	ProviderSchema **ProviderSchema

	// We take ResourceInstanceChange objects here just because that's what's
	// convenient to pass in from the evaltree implementation, but we really
	// only look at the "After" value of each change.
	Planned, Actual **plans.ResourceInstanceChange
}

func (n *EvalCheckPlannedChange) Eval(ctx EvalContext) (interface{}, error) {
	providerSchema := *n.ProviderSchema
	plannedChange := *n.Planned
	actualChange := *n.Actual

	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider does not support %q", n.Addr.Resource.Type)
	}

	var diags tfdiags.Diagnostics
	absAddr := n.Addr.Absolute(ctx.Path())

	log.Printf("[TRACE] EvalCheckPlannedChange: Verifying that actual change (action %s) matches planned change (action %s)", actualChange.Action, plannedChange.Action)

	if plannedChange.Action != actualChange.Action {
		switch {
		case plannedChange.Action == plans.Update && actualChange.Action == plans.NoOp:
			// It's okay for an update to become a NoOp once we've filled in
			// all of the unknown values, since the final values might actually
			// match what was there before after all.
			log.Printf("[DEBUG] After incorporating new values learned so far during apply, %s change has become NoOp", absAddr)
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced inconsistent final plan",
				fmt.Sprintf(
					"When expanding the plan for %s to include new values learned so far during apply, provider %q changed the planned action from %s to %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					absAddr, n.ProviderAddr.ProviderConfig.Type,
					plannedChange.Action, actualChange.Action,
				),
			))
		}
	}

	errs := objchange.AssertObjectCompatible(schema, plannedChange.After, actualChange.After)
	for _, err := range errs {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced inconsistent final plan",
			fmt.Sprintf(
				"When expanding the plan for %s to include new values learned so far during apply, provider %q produced an invalid new value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				absAddr, n.ProviderAddr.ProviderConfig.Type, tfdiags.FormatError(err),
			),
		))
	}
	return nil, diags.Err()
}

// EvalDiff is an EvalNode implementation that detects changes for a given
// resource instance.
type EvalDiff struct {
	Addr           addrs.ResourceInstance
	Config         *configs.Resource
	Provider       *providers.Interface
	ProviderAddr   addrs.AbsProviderConfig
	ProviderSchema **ProviderSchema
	State          **states.ResourceInstanceObject
	PreviousDiff   **plans.ResourceInstanceChange

	// CreateBeforeDestroy is set if either the resource's own config sets
	// create_before_destroy explicitly or if dependencies have forced the
	// resource to be handled as create_before_destroy in order to avoid
	// a dependency cycle.
	CreateBeforeDestroy bool

	OutputChange **plans.ResourceInstanceChange
	OutputValue  *cty.Value
	OutputState  **states.ResourceInstanceObject

	Stub bool
}

// TODO: test
func (n *EvalDiff) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State
	config := *n.Config
	provider := *n.Provider
	providerSchema := *n.ProviderSchema

	if providerSchema == nil {
		return nil, fmt.Errorf("provider schema is unavailable for %s", n.Addr)
	}
	if n.ProviderAddr.ProviderConfig.Type == "" {
		panic(fmt.Sprintf("EvalDiff for %s does not have ProviderAddr set", n.Addr.Absolute(ctx.Path())))
	}

	var diags tfdiags.Diagnostics

	// Evaluate the configuration
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Type)
	}
	forEach, _ := evaluateResourceForEachExpression(n.Config.ForEach, ctx)
	keyData := EvalDataForInstanceKey(n.Addr.Key, forEach)
	configVal, _, configDiags := ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags.Err()
	}

	absAddr := n.Addr.Absolute(ctx.Path())
	var priorVal cty.Value
	var priorValTainted cty.Value
	var priorPrivate []byte
	if state != nil {
		if state.Status != states.ObjectTainted {
			priorVal = state.Value
			priorPrivate = state.Private
		} else {
			// If the prior state is tainted then we'll proceed below like
			// we're creating an entirely new object, but then turn it into
			// a synthetic "Replace" change at the end, creating the same
			// result as if the provider had marked at least one argument
			// change as "requires replacement".
			priorValTainted = state.Value
			priorVal = cty.NullVal(schema.ImpliedType())
		}
	} else {
		priorVal = cty.NullVal(schema.ImpliedType())
	}

	proposedNewVal := objchange.ProposedNewObject(schema, priorVal, configVal)

	// Call pre-diff hook
	if !n.Stub {
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreDiff(absAddr, states.CurrentGen, priorVal, proposedNewVal)
		})
		if err != nil {
			return nil, err
		}
	}

	log.Printf("[TRACE] Re-validating config for %q", n.Addr.Absolute(ctx.Path()))
	// Allow the provider to validate the final set of values.
	// The config was statically validated early on, but there may have been
	// unknown values which the provider could not validate at the time.
	validateResp := provider.ValidateResourceTypeConfig(
		providers.ValidateResourceTypeConfigRequest{
			TypeName: n.Addr.Resource.Type,
			Config:   configVal,
		},
	)
	if validateResp.Diagnostics.HasErrors() {
		return nil, validateResp.Diagnostics.InConfigBody(config.Config).Err()
	}

	// The provider gets an opportunity to customize the proposed new value,
	// which in turn produces the _planned_ new value. But before
	// we send back this information, we need to process ignore_changes
	// so that CustomizeDiff will not act on them
	var ignoreChangeDiags tfdiags.Diagnostics
	proposedNewVal, ignoreChangeDiags = n.processIgnoreChanges(priorVal, proposedNewVal)
	diags = diags.Append(ignoreChangeDiags)
	if ignoreChangeDiags.HasErrors() {
		return nil, diags.Err()
	}

	resp := provider.PlanResourceChange(providers.PlanResourceChangeRequest{
		TypeName:         n.Addr.Resource.Type,
		Config:           configVal,
		PriorState:       priorVal,
		ProposedNewState: proposedNewVal,
		PriorPrivate:     priorPrivate,
	})
	diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config))
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	plannedNewVal := resp.PlannedState
	plannedPrivate := resp.PlannedPrivate

	if plannedNewVal == cty.NilVal {
		// Should never happen. Since real-world providers return via RPC a nil
		// is always a bug in the client-side stub. This is more likely caused
		// by an incompletely-configured mock provider in tests, though.
		panic(fmt.Sprintf("PlanResourceChange of %s produced nil value", absAddr.String()))
	}

	// We allow the planned new value to disagree with configuration _values_
	// here, since that allows the provider to do special logic like a
	// DiffSuppressFunc, but we still require that the provider produces
	// a value whose type conforms to the schema.
	for _, err := range plannedNewVal.Type().TestConformance(schema.ImpliedType()) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Provider produced invalid plan",
			fmt.Sprintf(
				"Provider %q planned an invalid value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
				n.ProviderAddr.ProviderConfig.Type, tfdiags.FormatErrorPrefixed(err, absAddr.String()),
			),
		))
	}
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	if errs := objchange.AssertPlanValid(schema, priorVal, configVal, plannedNewVal); len(errs) > 0 {
		if resp.LegacyTypeSystem {
			// The shimming of the old type system in the legacy SDK is not precise
			// enough to pass this consistency check, so we'll give it a pass here,
			// but we will generate a warning about it so that we are more likely
			// to notice in the logs if an inconsistency beyond the type system
			// leads to a downstream provider failure.
			var buf strings.Builder
			fmt.Fprintf(&buf, "[WARN] Provider %q produced an invalid plan for %s, but we are tolerating it because it is using the legacy plugin SDK.\n    The following problems may be the cause of any confusing errors from downstream operations:", n.ProviderAddr.ProviderConfig.Type, absAddr)
			for _, err := range errs {
				fmt.Fprintf(&buf, "\n      - %s", tfdiags.FormatError(err))
			}
			log.Print(buf.String())
		} else {
			for _, err := range errs {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Provider produced invalid plan",
					fmt.Sprintf(
						"Provider %q planned an invalid value for %s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
						n.ProviderAddr.ProviderConfig.Type, tfdiags.FormatErrorPrefixed(err, absAddr.String()),
					),
				))
			}
			return nil, diags.Err()
		}
	}

	// The provider produces a list of paths to attributes whose changes mean
	// that we must replace rather than update an existing remote object.
	// However, we only need to do that if the identified attributes _have_
	// actually changed -- particularly after we may have undone some of the
	// changes in processIgnoreChanges -- so now we'll filter that list to
	// include only where changes are detected.
	reqRep := cty.NewPathSet()
	if len(resp.RequiresReplace) > 0 {
		for _, path := range resp.RequiresReplace {
			if priorVal.IsNull() {
				// If prior is null then we don't expect any RequiresReplace at all,
				// because this is a Create action.
				continue
			}

			priorChangedVal, priorPathDiags := hcl.ApplyPath(priorVal, path, nil)
			plannedChangedVal, plannedPathDiags := hcl.ApplyPath(plannedNewVal, path, nil)
			if plannedPathDiags.HasErrors() && priorPathDiags.HasErrors() {
				// This means the path was invalid in both the prior and new
				// values, which is an error with the provider itself.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Provider produced invalid plan",
					fmt.Sprintf(
						"Provider %q has indicated \"requires replacement\" on %s for a non-existent attribute path %#v.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
						n.ProviderAddr.ProviderConfig.Type, absAddr, path,
					),
				))
				continue
			}

			// Make sure we have valid Values for both values.
			// Note: if the opposing value was of the type
			// cty.DynamicPseudoType, the type assigned here may not exactly
			// match the schema. This is fine here, since we're only going to
			// check for equality, but if the NullVal is to be used, we need to
			// check the schema for th true type.
			switch {
			case priorChangedVal == cty.NilVal && plannedChangedVal == cty.NilVal:
				// this should never happen without ApplyPath errors above
				panic("requires replace path returned 2 nil values")
			case priorChangedVal == cty.NilVal:
				priorChangedVal = cty.NullVal(plannedChangedVal.Type())
			case plannedChangedVal == cty.NilVal:
				plannedChangedVal = cty.NullVal(priorChangedVal.Type())
			}

			eqV := plannedChangedVal.Equals(priorChangedVal)
			if !eqV.IsKnown() || eqV.False() {
				reqRep.Add(path)
			}
		}
		if diags.HasErrors() {
			return nil, diags.Err()
		}
	}

	eqV := plannedNewVal.Equals(priorVal)
	eq := eqV.IsKnown() && eqV.True()

	var action plans.Action
	switch {
	case priorVal.IsNull():
		action = plans.Create
	case eq:
		action = plans.NoOp
	case !reqRep.Empty():
		// If there are any "requires replace" paths left _after our filtering
		// above_ then this is a replace action.
		if n.CreateBeforeDestroy {
			action = plans.CreateThenDelete
		} else {
			action = plans.DeleteThenCreate
		}
	default:
		action = plans.Update
		// "Delete" is never chosen here, because deletion plans are always
		// created more directly elsewhere, such as in "orphan" handling.
	}

	if action.IsReplace() {
		// In this strange situation we want to produce a change object that
		// shows our real prior object but has a _new_ object that is built
		// from a null prior object, since we're going to delete the one
		// that has all the computed values on it.
		//
		// Therefore we'll ask the provider to plan again here, giving it
		// a null object for the prior, and then we'll meld that with the
		// _actual_ prior state to produce a correctly-shaped replace change.
		// The resulting change should show any computed attributes changing
		// from known prior values to unknown values, unless the provider is
		// able to predict new values for any of these computed attributes.
		nullPriorVal := cty.NullVal(schema.ImpliedType())

		// create a new proposed value from the null state and the config
		proposedNewVal = objchange.ProposedNewObject(schema, nullPriorVal, configVal)

		resp = provider.PlanResourceChange(providers.PlanResourceChangeRequest{
			TypeName:         n.Addr.Resource.Type,
			Config:           configVal,
			PriorState:       nullPriorVal,
			ProposedNewState: proposedNewVal,
			PriorPrivate:     plannedPrivate,
		})
		// We need to tread carefully here, since if there are any warnings
		// in here they probably also came out of our previous call to
		// PlanResourceChange above, and so we don't want to repeat them.
		// Consequently, we break from the usual pattern here and only
		// append these new diagnostics if there's at least one error inside.
		if resp.Diagnostics.HasErrors() {
			diags = diags.Append(resp.Diagnostics.InConfigBody(config.Config))
			return nil, diags.Err()
		}
		plannedNewVal = resp.PlannedState
		plannedPrivate = resp.PlannedPrivate
		for _, err := range plannedNewVal.Type().TestConformance(schema.ImpliedType()) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider produced invalid plan",
				fmt.Sprintf(
					"Provider %q planned an invalid value for %s%s.\n\nThis is a bug in the provider, which should be reported in the provider's own issue tracker.",
					n.ProviderAddr.ProviderConfig.Type, absAddr, tfdiags.FormatError(err),
				),
			))
		}
		if diags.HasErrors() {
			return nil, diags.Err()
		}
	}

	// If our prior value was tainted then we actually want this to appear
	// as a replace change, even though so far we've been treating it as a
	// create.
	if action == plans.Create && priorValTainted != cty.NilVal {
		if n.CreateBeforeDestroy {
			action = plans.CreateThenDelete
		} else {
			action = plans.DeleteThenCreate
		}
		priorVal = priorValTainted
	}

	// As a special case, if we have a previous diff (presumably from the plan
	// phases, whereas we're now in the apply phase) and it was for a replace,
	// we've already deleted the original object from state by the time we
	// get here and so we would've ended up with a _create_ action this time,
	// which we now need to paper over to get a result consistent with what
	// we originally intended.
	if n.PreviousDiff != nil {
		prevChange := *n.PreviousDiff
		if prevChange.Action.IsReplace() && action == plans.Create {
			log.Printf("[TRACE] EvalDiff: %s treating Create change as %s change to match with earlier plan", absAddr, prevChange.Action)
			action = prevChange.Action
			priorVal = prevChange.Before
		}
	}

	// Call post-refresh hook
	if !n.Stub {
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostDiff(absAddr, states.CurrentGen, action, priorVal, plannedNewVal)
		})
		if err != nil {
			return nil, err
		}
	}

	// Update our output if we care
	if n.OutputChange != nil {
		*n.OutputChange = &plans.ResourceInstanceChange{
			Addr:         absAddr,
			Private:      plannedPrivate,
			ProviderAddr: n.ProviderAddr,
			Change: plans.Change{
				Action: action,
				Before: priorVal,
				After:  plannedNewVal,
			},
			RequiredReplace: reqRep,
		}
	}

	if n.OutputValue != nil {
		*n.OutputValue = configVal
	}

	// Update the state if we care
	if n.OutputState != nil {
		*n.OutputState = &states.ResourceInstanceObject{
			// We use the special "planned" status here to note that this
			// object's value is not yet complete. Objects with this status
			// cannot be used during expression evaluation, so the caller
			// must _also_ record the returned change in the active plan,
			// which the expression evaluator will use in preference to this
			// incomplete value recorded in the state.
			Status:  states.ObjectPlanned,
			Value:   plannedNewVal,
			Private: plannedPrivate,
		}
	}

	return nil, nil
}

func (n *EvalDiff) processIgnoreChanges(prior, proposed cty.Value) (cty.Value, tfdiags.Diagnostics) {
	// ignore_changes only applies when an object already exists, since we
	// can't ignore changes to a thing we've not created yet.
	if prior.IsNull() {
		return proposed, nil
	}

	ignoreChanges := n.Config.Managed.IgnoreChanges
	ignoreAll := n.Config.Managed.IgnoreAllChanges

	if len(ignoreChanges) == 0 && !ignoreAll {
		return proposed, nil
	}
	if ignoreAll {
		return prior, nil
	}
	if prior.IsNull() || proposed.IsNull() {
		// Ignore changes doesn't apply when we're creating for the first time.
		// Proposed should never be null here, but if it is then we'll just let it be.
		return proposed, nil
	}

	return processIgnoreChangesIndividual(prior, proposed, ignoreChanges)
}

func processIgnoreChangesIndividual(prior, proposed cty.Value, ignoreChanges []hcl.Traversal) (cty.Value, tfdiags.Diagnostics) {
	// When we walk below we will be using cty.Path values for comparison, so
	// we'll convert our traversals here so we can compare more easily.
	ignoreChangesPath := make([]cty.Path, len(ignoreChanges))
	for i, traversal := range ignoreChanges {
		path := make(cty.Path, len(traversal))
		for si, step := range traversal {
			switch ts := step.(type) {
			case hcl.TraverseRoot:
				path[si] = cty.GetAttrStep{
					Name: ts.Name,
				}
			case hcl.TraverseAttr:
				path[si] = cty.GetAttrStep{
					Name: ts.Name,
				}
			case hcl.TraverseIndex:
				path[si] = cty.IndexStep{
					Key: ts.Key,
				}
			default:
				panic(fmt.Sprintf("unsupported traversal step %#v", step))
			}
		}
		ignoreChangesPath[i] = path
	}

	var diags tfdiags.Diagnostics
	ret, _ := cty.Transform(proposed, func(path cty.Path, v cty.Value) (cty.Value, error) {
		// First we must see if this is a path that's being ignored at all.
		// We're looking for an exact match here because this walk will visit
		// leaf values first and then their containers, and we want to do
		// the "ignore" transform once we reach the point indicated, throwing
		// away any deeper values we already produced at that point.
		var ignoreTraversal hcl.Traversal
		for i, candidate := range ignoreChangesPath {
			if path.Equals(candidate) {
				ignoreTraversal = ignoreChanges[i]
			}
		}
		if ignoreTraversal == nil {
			return v, nil
		}

		// If we're able to follow the same path through the prior value,
		// we'll take the value there instead, effectively undoing the
		// change that was planned.
		priorV, diags := hcl.ApplyPath(prior, path, nil)
		if diags.HasErrors() {
			// We just ignore the errors and move on here, since we assume it's
			// just because the prior value was a slightly-different shape.
			// It could potentially also be that the traversal doesn't match
			// the schema, but we should've caught that during the validate
			// walk if so.
			return v, nil
		}
		return priorV, nil
	})
	return ret, diags
}

func (n *EvalDiff) processIgnoreChangesOld(diff *InstanceDiff) error {
	if diff == nil || n.Config == nil || n.Config.Managed == nil {
		return nil
	}
	ignoreChanges := n.Config.Managed.IgnoreChanges
	ignoreAll := n.Config.Managed.IgnoreAllChanges

	if len(ignoreChanges) == 0 && !ignoreAll {
		return nil
	}

	// If we're just creating the resource, we shouldn't alter the
	// Diff at all
	if diff.ChangeType() == DiffCreate {
		return nil
	}

	// If the resource has been tainted then we don't process ignore changes
	// since we MUST recreate the entire resource.
	if diff.GetDestroyTainted() {
		return nil
	}

	attrs := diff.CopyAttributes()

	// get the complete set of keys we want to ignore
	ignorableAttrKeys := make(map[string]bool)
	for k := range attrs {
		if ignoreAll {
			ignorableAttrKeys[k] = true
			continue
		}
		for _, ignoredTraversal := range ignoreChanges {
			ignoredKey := legacyFlatmapKeyForTraversal(ignoredTraversal)
			if k == ignoredKey || strings.HasPrefix(k, ignoredKey+".") {
				ignorableAttrKeys[k] = true
			}
		}
	}

	// If the resource was being destroyed, check to see if we can ignore the
	// reason for it being destroyed.
	if diff.GetDestroy() {
		for k, v := range attrs {
			if k == "id" {
				// id will always be changed if we intended to replace this instance
				continue
			}
			if v.Empty() || v.NewComputed {
				continue
			}

			// If any RequiresNew attribute isn't ignored, we need to keep the diff
			// as-is to be able to replace the resource.
			if v.RequiresNew && !ignorableAttrKeys[k] {
				return nil
			}
		}

		// Now that we know that we aren't replacing the instance, we can filter
		// out all the empty and computed attributes. There may be a bunch of
		// extraneous attribute diffs for the other non-requires-new attributes
		// going from "" -> "configval" or "" -> "<computed>".
		// We must make sure any flatmapped containers are filterred (or not) as a
		// whole.
		containers := groupContainers(diff)
		keep := map[string]bool{}
		for _, v := range containers {
			if v.keepDiff(ignorableAttrKeys) {
				// At least one key has changes, so list all the sibling keys
				// to keep in the diff
				for k := range v {
					keep[k] = true
					// this key may have been added by the user to ignore, but
					// if it's a subkey in a container, we need to un-ignore it
					// to keep the complete containter.
					delete(ignorableAttrKeys, k)
				}
			}
		}

		for k, v := range attrs {
			if (v.Empty() || v.NewComputed) && !keep[k] {
				ignorableAttrKeys[k] = true
			}
		}
	}

	// Here we undo the two reactions to RequireNew in EvalDiff - the "id"
	// attribute diff and the Destroy boolean field
	log.Printf("[DEBUG] Removing 'id' diff and setting Destroy to false " +
		"because after ignore_changes, this diff no longer requires replacement")
	diff.DelAttribute("id")
	diff.SetDestroy(false)

	// If we didn't hit any of our early exit conditions, we can filter the diff.
	for k := range ignorableAttrKeys {
		log.Printf("[DEBUG] [EvalIgnoreChanges] %s: Ignoring diff attribute: %s", n.Addr.String(), k)
		diff.DelAttribute(k)
	}

	return nil
}

// legacyFlagmapKeyForTraversal constructs a key string compatible with what
// the flatmap package would generate for an attribute addressable by the given
// traversal.
//
// This is used only to shim references to attributes within the diff and
// state structures, which have not (at the time of writing) yet been updated
// to use the newer HCL-based representations.
func legacyFlatmapKeyForTraversal(traversal hcl.Traversal) string {
	var buf bytes.Buffer
	first := true
	for _, step := range traversal {
		if !first {
			buf.WriteByte('.')
		}
		switch ts := step.(type) {
		case hcl.TraverseRoot:
			buf.WriteString(ts.Name)
		case hcl.TraverseAttr:
			buf.WriteString(ts.Name)
		case hcl.TraverseIndex:
			val := ts.Key
			switch val.Type() {
			case cty.Number:
				bf := val.AsBigFloat()
				buf.WriteString(bf.String())
			case cty.String:
				s := val.AsString()
				buf.WriteString(s)
			default:
				// should never happen, since no other types appear in
				// traversals in practice.
				buf.WriteByte('?')
			}
		default:
			// should never happen, since we've covered all of the types
			// that show up in parsed traversals in practice.
			buf.WriteByte('?')
		}
		first = false
	}
	return buf.String()
}

// a group of key-*ResourceAttrDiff pairs from the same flatmapped container
type flatAttrDiff map[string]*ResourceAttrDiff

// we need to keep all keys if any of them have a diff that's not ignored
func (f flatAttrDiff) keepDiff(ignoreChanges map[string]bool) bool {
	for k, v := range f {
		ignore := false
		for attr := range ignoreChanges {
			if strings.HasPrefix(k, attr) {
				ignore = true
			}
		}

		if !v.Empty() && !v.NewComputed && !ignore {
			return true
		}
	}
	return false
}

// sets, lists and maps need to be compared for diff inclusion as a whole, so
// group the flatmapped keys together for easier comparison.
func groupContainers(d *InstanceDiff) map[string]flatAttrDiff {
	isIndex := multiVal.MatchString
	containers := map[string]flatAttrDiff{}
	attrs := d.CopyAttributes()
	// we need to loop once to find the index key
	for k := range attrs {
		if isIndex(k) {
			// add the key, always including the final dot to fully qualify it
			containers[k[:len(k)-1]] = flatAttrDiff{}
		}
	}

	// loop again to find all the sub keys
	for prefix, values := range containers {
		for k, attrDiff := range attrs {
			// we include the index value as well, since it could be part of the diff
			if strings.HasPrefix(k, prefix) {
				values[k] = attrDiff
			}
		}
	}

	return containers
}

// EvalDiffDestroy is an EvalNode implementation that returns a plain
// destroy diff.
type EvalDiffDestroy struct {
	Addr         addrs.ResourceInstance
	DeposedKey   states.DeposedKey
	State        **states.ResourceInstanceObject
	ProviderAddr addrs.AbsProviderConfig

	Output      **plans.ResourceInstanceChange
	OutputState **states.ResourceInstanceObject
}

// TODO: test
func (n *EvalDiffDestroy) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())
	state := *n.State

	if n.ProviderAddr.ProviderConfig.Type == "" {
		if n.DeposedKey == "" {
			panic(fmt.Sprintf("EvalDiffDestroy for %s does not have ProviderAddr set", absAddr))
		} else {
			panic(fmt.Sprintf("EvalDiffDestroy for %s (deposed %s) does not have ProviderAddr set", absAddr, n.DeposedKey))
		}
	}

	// If there is no state or our attributes object is null then we're already
	// destroyed.
	if state == nil || state.Value.IsNull() {
		return nil, nil
	}

	// Call pre-diff hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreDiff(
			absAddr, n.DeposedKey.Generation(),
			state.Value,
			cty.NullVal(cty.DynamicPseudoType),
		)
	})
	if err != nil {
		return nil, err
	}

	// Change is always the same for a destroy. We don't need the provider's
	// help for this one.
	// TODO: Should we give the provider an opportunity to veto this?
	change := &plans.ResourceInstanceChange{
		Addr:       absAddr,
		DeposedKey: n.DeposedKey,
		Change: plans.Change{
			Action: plans.Delete,
			Before: state.Value,
			After:  cty.NullVal(cty.DynamicPseudoType),
		},
		Private:      state.Private,
		ProviderAddr: n.ProviderAddr,
	}

	// Call post-diff hook
	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostDiff(
			absAddr,
			n.DeposedKey.Generation(),
			change.Action,
			change.Before,
			change.After,
		)
	})
	if err != nil {
		return nil, err
	}

	// Update our output
	*n.Output = change

	if n.OutputState != nil {
		// Record our proposed new state, which is nil because we're destroying.
		*n.OutputState = nil
	}

	return nil, nil
}

// EvalReduceDiff is an EvalNode implementation that takes a planned resource
// instance change as might be produced by EvalDiff or EvalDiffDestroy and
// "simplifies" it to a single atomic action to be performed by a specific
// graph node.
//
// Callers must specify whether they are a destroy node or a regular apply
// node.  If the result is NoOp then the given change requires no action for
// the specific graph node calling this and so evaluation of the that graph
// node should exit early and take no action.
//
// The object written to OutChange may either be identical to InChange or
// a new change object derived from InChange. Because of the former case, the
// caller must not mutate the object returned in OutChange.
type EvalReduceDiff struct {
	Addr      addrs.ResourceInstance
	InChange  **plans.ResourceInstanceChange
	Destroy   bool
	OutChange **plans.ResourceInstanceChange
}

// TODO: test
func (n *EvalReduceDiff) Eval(ctx EvalContext) (interface{}, error) {
	in := *n.InChange
	out := in.Simplify(n.Destroy)
	if n.OutChange != nil {
		*n.OutChange = out
	}
	if out.Action != in.Action {
		if n.Destroy {
			log.Printf("[TRACE] EvalReduceDiff: %s change simplified from %s to %s for destroy node", n.Addr, in.Action, out.Action)
		} else {
			log.Printf("[TRACE] EvalReduceDiff: %s change simplified from %s to %s for apply node", n.Addr, in.Action, out.Action)
		}
	}
	return nil, nil
}

// EvalReadDiff is an EvalNode implementation that retrieves the planned
// change for a particular resource instance object.
type EvalReadDiff struct {
	Addr           addrs.ResourceInstance
	DeposedKey     states.DeposedKey
	ProviderSchema **ProviderSchema
	Change         **plans.ResourceInstanceChange
}

func (n *EvalReadDiff) Eval(ctx EvalContext) (interface{}, error) {
	providerSchema := *n.ProviderSchema
	changes := ctx.Changes()
	addr := n.Addr.Absolute(ctx.Path())

	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Type)
	}

	gen := states.CurrentGen
	if n.DeposedKey != states.NotDeposed {
		gen = n.DeposedKey
	}
	csrc := changes.GetResourceInstanceChange(addr, gen)
	if csrc == nil {
		log.Printf("[TRACE] EvalReadDiff: No planned change recorded for %s", addr)
		return nil, nil
	}

	change, err := csrc.Decode(schema.ImpliedType())
	if err != nil {
		return nil, fmt.Errorf("failed to decode planned changes for %s: %s", addr, err)
	}
	if n.Change != nil {
		*n.Change = change
	}

	log.Printf("[TRACE] EvalReadDiff: Read %s change from plan for %s", change.Action, addr)

	return nil, nil
}

// EvalWriteDiff is an EvalNode implementation that saves a planned change
// for an instance object into the set of global planned changes.
type EvalWriteDiff struct {
	Addr           addrs.ResourceInstance
	DeposedKey     states.DeposedKey
	ProviderSchema **ProviderSchema
	Change         **plans.ResourceInstanceChange
}

// TODO: test
func (n *EvalWriteDiff) Eval(ctx EvalContext) (interface{}, error) {
	changes := ctx.Changes()
	addr := n.Addr.Absolute(ctx.Path())
	if n.Change == nil || *n.Change == nil {
		// Caller sets nil to indicate that we need to remove a change from
		// the set of changes.
		gen := states.CurrentGen
		if n.DeposedKey != states.NotDeposed {
			gen = n.DeposedKey
		}
		changes.RemoveResourceInstanceChange(addr, gen)
		return nil, nil
	}

	providerSchema := *n.ProviderSchema
	change := *n.Change

	if change.Addr.String() != addr.String() || change.DeposedKey != n.DeposedKey {
		// Should never happen, and indicates a bug in the caller.
		panic("inconsistent address and/or deposed key in EvalWriteDiff")
	}

	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider does not support resource type %q", n.Addr.Resource.Type)
	}

	csrc, err := change.Encode(schema.ImpliedType())
	if err != nil {
		return nil, fmt.Errorf("failed to encode planned changes for %s: %s", addr, err)
	}

	changes.AppendResourceInstanceChange(csrc)
	if n.DeposedKey == states.NotDeposed {
		log.Printf("[TRACE] EvalWriteDiff: recorded %s change for %s", change.Action, addr)
	} else {
		log.Printf("[TRACE] EvalWriteDiff: recorded %s change for %s deposed object %s", change.Action, addr, n.DeposedKey)
	}

	return nil, nil
}
