// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/didyoumean"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/resources/ephemeral"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Evaluator provides the necessary contextual data for evaluating expressions
// for a particular walk operation.
type Evaluator struct {
	// Operation defines what type of operation this evaluator is being used
	// for.
	Operation walkOperation

	// Meta is contextual metadata about the current operation.
	Meta *ContextMeta

	// Config is the root node in the configuration tree.
	Config *configs.Config

	// Instances tracks the dynamic instances that are associated with each
	// module call or resource. The graph walk gradually registers the
	// set of instances for each object within the graph nodes for those
	// objects, and so as long as the graph has been built correctly the
	// set of instances for an object should always be available by the time
	// we're evaluating expressions that refer to it.
	Instances *instances.Expander

	// NamedValues is where we keep the values of already-evaluated input
	// variables, local values, and output values.
	NamedValues *namedvals.State

	// EphemeralResources tracks the currently-open instances of any ephemeral
	// resources.
	EphemeralResources *ephemeral.Resources

	// Deferrals tracks resources and modules that have had either their
	// expansion or their specific planned actions deferred to a future
	// plan/apply round.
	Deferrals *deferring.Deferred

	// Plugins is the library of available plugin components (providers and
	// provisioners) that we have available to help us evaluate expressions
	// that interact with plugin-provided objects.
	//
	// From this we only access the schemas of the plugins, and don't otherwise
	// interact with plugin instances.
	Plugins *contextPlugins

	// State is the current state, embedded in a wrapper that ensures that
	// it can be safely accessed and modified concurrently.
	State *states.SyncState

	// Changes is the set of proposed changes, embedded in a wrapper that
	// ensures they can be safely accessed and modified concurrently.
	Changes *plans.ChangesSync

	PlanTimestamp time.Time
}

// Scope creates an evaluation scope for the given module path and optional
// resource.
//
// If the "self" argument is nil then the "self" object is not available
// in evaluated expressions. Otherwise, it behaves as an alias for the given
// address.
func (e *Evaluator) Scope(data lang.Data, self addrs.Referenceable, source addrs.Referenceable, extFuncs lang.ExternalFuncs) *lang.Scope {
	return &lang.Scope{
		Data:          data,
		ParseRef:      addrs.ParseRef,
		SelfAddr:      self,
		SourceAddr:    source,
		PureOnly:      e.Operation != walkApply && e.Operation != walkDestroy && e.Operation != walkEval,
		BaseDir:       ".", // Always current working directory for now.
		PlanTimestamp: e.PlanTimestamp,
		ExternalFuncs: extFuncs,
	}
}

// evaluationStateData is an implementation of lang.Data that resolves
// references primarily (but not exclusively) using information from a State.
type evaluationStateData struct {
	*evaluationData

	// ModulePath is the path through the dynamic module tree to the module
	// that references will be resolved relative to.
	ModulePath addrs.ModuleInstance

	// InstanceKeyData describes the values, if any, that are accessible due
	// to repetition of a containing object using "count" or "for_each"
	// arguments. (It is _not_ used for the for_each inside "dynamic" blocks,
	// since the user specifies in that case which variable name to locally
	// shadow.)
	InstanceKeyData InstanceKeyEvalData

	// Operation records the type of walk the evaluationStateData is being used
	// for.
	Operation walkOperation
}

// InstanceKeyEvalData is the old name for instances.RepetitionData, aliased
// here for compatibility. In new code, use instances.RepetitionData instead.
type InstanceKeyEvalData = instances.RepetitionData

// EvalDataForInstanceKey constructs a suitable InstanceKeyEvalData for
// evaluating in a context that has the given instance key.
//
// The forEachMap argument can be nil when preparing for evaluation
// in a context where each.value is prohibited, such as a destroy-time
// provisioner. In that case, the returned EachValue will always be
// cty.NilVal.
func EvalDataForInstanceKey(key addrs.InstanceKey, forEachMap map[string]cty.Value) InstanceKeyEvalData {
	var evalData InstanceKeyEvalData
	if key == nil {
		return evalData
	}

	keyValue := key.Value()
	switch keyValue.Type() {
	case cty.String:
		evalData.EachKey = keyValue
		evalData.EachValue = forEachMap[keyValue.AsString()]
	case cty.Number:
		evalData.CountIndex = keyValue
	}
	return evalData
}

// EvalDataForNoInstanceKey is a value of InstanceKeyData that sets no instance
// key values at all, suitable for use in contexts where no keyed instance
// is relevant.
var EvalDataForNoInstanceKey = InstanceKeyEvalData{}

// evaluationStateData must implement lang.Data
var _ lang.Data = (*evaluationStateData)(nil)

// StaticValidateReferences calls [Evaluator.StaticValidateReferences] on
// the evaluator embedded in this data object, using this data object's
// static module path.
func (d *evaluationStateData) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable, source addrs.Referenceable) tfdiags.Diagnostics {
	return d.Evaluator.StaticValidateReferences(refs, d.ModulePath.Module(), self, source)
}

func (d *evaluationStateData) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	switch addr.Name {

	case "index":
		idxVal := d.InstanceKeyData.CountIndex
		if idxVal == cty.NilVal {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Reference to "count" in non-counted context`,
				Detail:   `The "count" object can only be used in "module", "resource", and "data" blocks, and only when the "count" argument is set.`,
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.UnknownVal(cty.Number), diags
		}
		return idxVal, diags

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "count" attribute`,
			Detail:   fmt.Sprintf(`The "count" object does not have an attribute named %q. The only supported attribute is count.index, which is the index of each instance of a resource block that has the "count" argument set.`, addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}
}

func (d *evaluationStateData) GetForEachAttr(addr addrs.ForEachAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var returnVal cty.Value
	switch addr.Name {

	case "key":
		returnVal = d.InstanceKeyData.EachKey
	case "value":
		returnVal = d.InstanceKeyData.EachValue

		if returnVal == cty.NilVal {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `each.value cannot be used in this context`,
				Detail:   `A reference to "each.value" has been used in a context in which it is unavailable, such as when the configuration no longer contains the value in its "for_each" expression. Remove this reference to each.value in your configuration to work around this error.`,
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.UnknownVal(cty.DynamicPseudoType), diags
		}
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "each" attribute`,
			Detail:   fmt.Sprintf(`The "each" object does not have an attribute named %q. The supported attributes are each.key and each.value, the current key and value pair of the "for_each" attribute set.`, addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	if returnVal == cty.NilVal {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to "each" in context without for_each`,
			Detail:   `The "each" object can be used only in "module" or "resource" blocks, and only when the "for_each" argument is set.`,
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.UnknownVal(cty.DynamicPseudoType), diags
	}
	return returnVal, diags
}

func (d *evaluationStateData) GetInputVariable(addr addrs.InputVariable, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// First we'll make sure the requested value is declared in configuration,
	// so we can produce a nice message if not.
	moduleConfig := d.Evaluator.Config.DescendantForInstance(d.ModulePath)
	if moduleConfig == nil {
		// should never happen, since we can't be evaluating in a module
		// that wasn't mentioned in configuration.
		panic(fmt.Sprintf("input variable read from %s, which has no configuration", d.ModulePath))
	}

	config := moduleConfig.Module.Variables[addr.Name]
	if config == nil {
		var suggestions []string
		for k := range moduleConfig.Module.Variables {
			suggestions = append(suggestions, k)
		}
		suggestion := didyoumean.NameSuggestion(addr.Name, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		} else {
			suggestion = fmt.Sprintf(" This variable can be declared with a variable %q {} block.", addr.Name)
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared input variable`,
			Detail:   fmt.Sprintf(`An input variable with the name %q has not been declared.%s`, addr.Name, suggestion),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	// During the validate walk, input variables are always unknown so
	// that we are validating the configuration for all possible input values
	// rather than for a specific set. Checking against a specific set of
	// input values then happens during the plan walk.
	//
	// This is important because otherwise the validation walk will tend to be
	// overly strict, requiring expressions throughout the configuration to
	// be complicated to accommodate all possible inputs, whereas returning
	// unknown here allows for simpler patterns like using input values as
	// guards to broadly enable/disable resources, avoid processing things
	// that are disabled, etc. Terraform's static validation leans towards
	// being liberal in what it accepts because the subsequent plan walk has
	// more information available and so can be more conservative.
	if d.Operation == walkValidate {
		// We should still capture the statically-configured marks during
		// the validate walk.
		ret := cty.UnknownVal(config.Type)
		if config.Sensitive {
			ret = ret.Mark(marks.Sensitive)
		}
		if config.Ephemeral {
			ret = ret.Mark(marks.Ephemeral)
		}
		return ret, diags
	}

	val := d.Evaluator.NamedValues.GetInputVariableValue(d.ModulePath.InputVariable(addr.Name))

	// Mark if sensitive and/or ephemeral
	if config.Sensitive {
		val = val.Mark(marks.Sensitive)
	}
	if config.Ephemeral {
		val = val.Mark(marks.Ephemeral)
	}

	return val, diags
}

func (d *evaluationStateData) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// First we'll make sure the requested value is declared in configuration,
	// so we can produce a nice message if not.
	moduleConfig := d.Evaluator.Config.DescendantForInstance(d.ModulePath)
	if moduleConfig == nil {
		// should never happen, since we can't be evaluating in a module
		// that wasn't mentioned in configuration.
		panic(fmt.Sprintf("local value read from %s, which has no configuration", d.ModulePath))
	}

	config := moduleConfig.Module.Locals[addr.Name]
	if config == nil {
		var suggestions []string
		for k := range moduleConfig.Module.Locals {
			suggestions = append(suggestions, k)
		}
		suggestion := didyoumean.NameSuggestion(addr.Name, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared local value`,
			Detail:   fmt.Sprintf(`A local value with the name %q has not been declared.%s`, addr.Name, suggestion),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	val := d.Evaluator.NamedValues.GetLocalValue(addr.Absolute(d.ModulePath))
	return val, diags
}

func (d *evaluationStateData) GetModule(addr addrs.ModuleCall, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Output results live in the module that declares them, which is one of
	// the child module instances of our current module path.
	moduleAddr := d.ModulePath.Module().Child(addr.Name)
	absAddr := addr.Absolute(d.ModulePath)

	parentCfg := d.Evaluator.Config.DescendantForInstance(d.ModulePath)
	callConfig, ok := parentCfg.Module.ModuleCalls[addr.Name]
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared module`,
			Detail:   fmt.Sprintf(`The configuration contains no %s.`, moduleAddr),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	// We'll consult the configuration to see what output names we are
	// expecting, so we can ensure the resulting object is of the expected
	// type even if our data is incomplete for some reason.
	moduleConfig := d.Evaluator.Config.Descendant(moduleAddr)
	if moduleConfig == nil {
		// should never happen, since we have a valid module call above, this
		// should be caught during static validation.
		panic(fmt.Sprintf("output value read from %s, which has no configuration", moduleAddr))
	}
	outputConfigs := moduleConfig.Module.Outputs

	// We don't do instance expansion during validation, and so we need to
	// return an unknown value. Technically we should always return
	// cty.DynamicVal here because the final value during plan will always
	// be an object or tuple type with unpredictable attributes/elements,
	// but because we never actually carry values forward from validation to
	// planning we lie a little here and return unknown list and map types,
	// just to give us more opportunities to catch author mistakes during
	// validation.
	//
	// This means that in practice any expression that refers to a module
	// call must be written to be valid for either a collection type or
	// structural type of similar kind, so that it can be considered as
	// valid during both the validate and plan walks.
	if d.Operation == walkValidate {
		atys := make(map[string]cty.Type, len(outputConfigs))
		for name := range outputConfigs {
			atys[name] = cty.DynamicPseudoType // output values are dynamically-typed
		}
		instTy := cty.Object(atys)

		switch {
		case callConfig.Count != nil:
			return cty.UnknownVal(cty.List(instTy)), diags
		case callConfig.ForEach != nil:
			return cty.UnknownVal(cty.Map(instTy)), diags
		default:
			return cty.UnknownVal(instTy), diags
		}
	}

	// For all other walk types, we proceed to dynamic evaluation of individual
	// instances, using the global instance expander. An earlier graph node
	// should always have registered the expansion of this module call before
	// we get here, unless there's a bug in the graph builders.
	allInstances := d.Evaluator.Instances
	instKeyType, instKeys, known := allInstances.ExpandAbsModuleCall(absAddr)
	if !known {
		// If we don't know which instances exist then we can't really predict
		// anything at all. We can't even predict the return type based on
		// instKeyType because output values are dynamically-typed and so
		// our final result will always be an object or tuple type whose
		// attribute/element count we cannot predict.
		return cty.DynamicVal, diags
	}

	instanceObjVal := func(instKey addrs.InstanceKey) (cty.Value, tfdiags.Diagnostics) {
		// This function must always return a valid value, even if it's
		// just a cty.DynamicVal placeholder accompanying error diagnostics.
		var diags tfdiags.Diagnostics

		namedVals := d.Evaluator.NamedValues
		moduleInstAddr := absAddr.Instance(instKey)
		attrs := make(map[string]cty.Value, len(outputConfigs))
		for name := range outputConfigs {
			outputAddr := moduleInstAddr.OutputValue(name)

			// Although we do typically expect the graph dependencies to
			// ensure that values get registered before they are needed,
			// we track depedencies with specific output values where
			// possible, instead of with entire module calls, and so
			// in this specific case it's valid for some of this call's
			// output values to not be known yet, with the graph builder
			// being responsible for making sure that no expression
			// in the configuration can actually observe that.
			if !namedVals.HasOutputValue(outputAddr) {
				attrs[name] = cty.DynamicVal
				continue
			}
			outputVal := namedVals.GetOutputValue(outputAddr)
			attrs[name] = outputVal
		}

		return cty.ObjectVal(attrs), diags
	}

	switch instKeyType {

	case addrs.NoKeyType:
		// In this case we should always have exactly one instance that
		// is addrs.NoKey. If not then there's a bug in the [instances.Expander]
		// implementation.
		if len(instKeys) != 1 {
			panic(fmt.Sprintf("module call has no instance key type but has %d instances (should be 1)", len(instKeys)))
		}
		ret, moreDiags := instanceObjVal(instKeys[0])
		diags = diags.Append(moreDiags)
		return ret, diags

	case addrs.IntKeyType:
		// We can assume that the instance keys are in ascending numerical order
		// and are consecutive, per the contract of allInstances.ExpandModuleCall.
		elems := make([]cty.Value, 0, len(instKeys))
		for _, instKey := range instKeys {
			instVal, moreDiags := instanceObjVal(instKey)
			elems = append(elems, instVal)
			diags = diags.Append(moreDiags)
		}
		return cty.TupleVal(elems), diags

	case addrs.StringKeyType:
		attrs := make(map[string]cty.Value, len(instKeys))
		for _, instKey := range instKeys {
			instVal, moreDiags := instanceObjVal(instKey)
			attrs[string(instKey.(addrs.StringKey))] = instVal
			diags = diags.Append(moreDiags)
		}
		return cty.ObjectVal(attrs), diags

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Unsupported instance key type`,
			Detail: fmt.Sprintf(
				`Module call %s has instance key type %#v, which is not supported by the expression evaluator. This is a bug in Terraform.`,
				absAddr, instKeyType,
			),
			Subject: rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}
}

func (d *evaluationStateData) GetResource(addr addrs.Resource, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	// First we'll consult the configuration to see if an resource of this
	// name is declared at all.
	moduleAddr := d.ModulePath
	moduleConfig := d.Evaluator.Config.DescendantForInstance(moduleAddr)
	if moduleConfig == nil {
		// should never happen, since we can't be evaluating in a module
		// that wasn't mentioned in configuration.
		panic(fmt.Sprintf("resource value read from %s, which has no configuration", moduleAddr))
	}

	config := moduleConfig.Module.ResourceByAddr(addr)
	if config == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared resource`,
			Detail:   fmt.Sprintf(`A resource %q %q has not been declared in %s`, addr.Type, addr.Name, moduleDisplayAddr(moduleAddr)),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	// Much of this function was written before we had factored out the handling
	// of instance keys into the separate instance expander model, and so it
	// does a bunch of instance-related work itself below.
	//
	// Currently, unknown instance keys are only possible when planning with
	// DeferralAllowed set to true in the PlanOpts, which should only be the
	// case in the stacks runtime (not the "normal terraform" modules runtime).
	// Thus, we have some amount of duplicated code remaining, to be more
	// certain that stacks-specific behaviors won't leak out into the standard
	// runtime.
	//
	// TODO: When deferred actions are more stable and robust in stacks, it
	// would be nice to rework this function to rely on the ResourceInstanceKeys
	// result for _all_ of its work, rather than continuing to duplicate a bunch
	// of the logic we've tried to encapsulate over ther already.
	if d.Operation == walkPlan || d.Operation == walkApply {
		if _, _, hasUnknownKeys := d.Evaluator.Instances.ResourceInstanceKeys(addr.Absolute(moduleAddr)); hasUnknownKeys {
			// There really isn't anything interesting we can do in this situation,
			// because it means we have an unknown for_each/count, in which case
			// we can't even predict what the result type will be because it
			// would be either an object or tuple type decided based on the instance
			// keys.
			// (We can't get in here for a single-instance resource because in that
			// case we would know that there's only one key and it's addrs.NoKey,
			// so we'll fall through to the other logic below.)
			unknownVal := cty.DynamicVal

			// If an ephemeral resource is deferred we need to mark the returned unknown value as ephemeral
			if addr.Mode == addrs.EphemeralResourceMode {
				unknownVal = unknownVal.Mark(marks.Ephemeral)
			}
			return unknownVal, diags
		}
	}

	// Build the provider address from configuration, since we may not have
	// state available in all cases.
	// We need to build an abs provider address, but we can use a default
	// instance since we're only interested in the schema.
	schema := d.getResourceSchema(addr, config.Provider)
	if schema == nil {
		// This shouldn't happen, since validation before we get here should've
		// taken care of it, but we'll show a reasonable error message anyway.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Missing resource type schema`,
			Detail:   fmt.Sprintf("No schema is available for %s in %s. This is a bug in Terraform and should be reported.", addr, config.Provider),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}
	ty := schema.ImpliedType()

	if addr.Mode == addrs.EphemeralResourceMode {
		// FIXME: This does not yet work with deferrals, and it would be nice to
		// find some way to refactor this so that the following code is not so
		// tethered to the current implementation details. Instead we should
		// have an abstract idea of first determining what instances the
		// resource has (using d.Evaluator.Instances.ResourceInstanceKeys) and
		// then retrieving the value for each instance to assemble into the
		// result, using some per-resource-mode logic maintained elsewhere.
		return d.getEphemeralResource(addr, rng)
	}

	rs := d.Evaluator.State.Resource(addr.Absolute(d.ModulePath))

	if rs == nil {
		switch d.Operation {
		case walkPlan, walkApply:
			// During plan and apply as we evaluate each removed instance they
			// are removed from the working state. Since we know there are no
			// instances, return an empty container of the expected type.
			switch {
			case config.Count != nil:
				return cty.EmptyTupleVal, diags
			case config.ForEach != nil:
				return cty.EmptyObjectVal, diags
			default:
				// While we can reference an expanded resource with 0
				// instances, we cannot reference instances that do not exist.
				// Due to the fact that we may have direct references to
				// instances that may end up in a root output during destroy
				// (since a planned destroy cannot yet remove root outputs), we
				// need to return a dynamic value here to allow evaluation to
				// continue.
				log.Printf("[ERROR] unknown instance %q referenced during %s", addr.Absolute(d.ModulePath), d.Operation)
				return cty.DynamicVal, diags
			}

		case walkImport:
			// Import does not yet plan resource changes, so new resources from
			// config are not going to be found here. Once walkImport fully
			// plans resources, this case should not longer be needed.
			// In the single instance case, we can return a typed unknown value
			// for the instance to better satisfy other expressions using the
			// value. This of course will not help if statically known
			// attributes are expected to be known elsewhere, but reduces the
			// number of problematic configs for now.
			// Unlike in plan and apply above we can't be sure the count or
			// for_each instances are empty, so we return a DynamicVal. We
			// don't really have a good value to return otherwise -- empty
			// values will fail for direct index expressions, and unknown
			// Lists and Maps could fail in some type unifications.
			switch {
			case config.Count != nil:
				return cty.DynamicVal, diags
			case config.ForEach != nil:
				return cty.DynamicVal, diags
			default:
				return cty.UnknownVal(ty), diags
			}

		default:
			// We should only end up here during the validate walk (or
			// console/eval), since later walks should have at least partial
			// states populated for all resources in the configuration.
			return cty.DynamicVal, diags
		}
	}

	// Now, we're going to build up a value that represents the resource
	// or resources that are in the state.
	instances := map[addrs.InstanceKey]cty.Value{}

	// First, we're going to load any instances that we have written into the
	// deferrals system. A deferred resource overrides anything that might be
	// in the state for the resource, so we do this first.
	for key, value := range d.Evaluator.Deferrals.GetDeferredResourceInstances(addr.Absolute(d.ModulePath)) {
		instances[key] = value
	}

	// Decode all instances in the current state
	pendingDestroy := d.Operation == walkDestroy
	for key, is := range rs.Instances {
		if _, ok := instances[key]; ok {
			// Then we've already loaded this instance from the deferrals so
			// we'll just ignore it being in state.
			continue
		}
		// Otherwise, we'll load the instance from state.

		if is == nil || is.Current == nil {
			// Assume we're dealing with an instance that hasn't been created yet.
			instances[key] = cty.UnknownVal(ty)
			continue
		}

		instAddr := addr.Instance(key).Absolute(d.ModulePath)
		change := d.Evaluator.Changes.GetResourceInstanceChange(instAddr, addrs.NotDeposed)
		if change != nil {
			// Don't take any resources that are yet to be deleted into account.
			// If the referenced resource is CreateBeforeDestroy, then orphaned
			// instances will be in the state, as they are not destroyed until
			// after their dependants are updated.
			if change.Action == plans.Delete {
				if !pendingDestroy {
					continue
				}
			}
		}

		// Planned resources are temporarily stored in state with empty values,
		// and need to be replaced by the planned value here.
		if is.Current.Status == states.ObjectPlanned {
			if change == nil {
				// FIXME: This is usually an unfortunate case where we need to
				// lookup an individual instance referenced via "self" for
				// postconditions which we know exists, but because evaluation
				// must always get the resource in aggregate some instance
				// changes may not yet be registered.
				instances[key] = cty.DynamicVal
				// log the problem for debugging, since it may be a legitimate error we can't catch
				log.Printf("[WARN] instance %s is marked as having a change pending but that change is not recorded in the plan", instAddr)
				continue
			}
			instances[key] = change.After
			continue
		}

		ios, err := is.Current.Decode(ty)
		if err != nil {
			// This shouldn't happen, since by the time we get here we
			// should have upgraded the state data already.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid resource instance data in state",
				Detail:   fmt.Sprintf("Instance %s data could not be decoded from the state: %s.", instAddr, err),
				Subject:  &config.DeclRange,
			})
			continue
		}

		val := ios.Value

		instances[key] = val
	}

	// ret should be populated with a valid value in all cases below
	var ret cty.Value

	switch {
	case config.Count != nil:
		// figure out what the last index we have is
		length := -1
		for key := range instances {
			intKey, ok := key.(addrs.IntKey)
			if !ok {
				continue
			}
			if int(intKey) >= length {
				length = int(intKey) + 1
			}
		}

		if length > 0 {
			vals := make([]cty.Value, length)
			for key, instance := range instances {
				intKey, ok := key.(addrs.IntKey)
				if !ok {
					// old key from state, which isn't valid for evaluation
					continue
				}

				vals[int(intKey)] = instance
			}

			// Insert unknown values where there are any missing instances
			for i, v := range vals {
				if v == cty.NilVal {
					vals[i] = cty.UnknownVal(ty)
				}
			}
			ret = cty.TupleVal(vals)
		} else {
			ret = cty.EmptyTupleVal
		}

	case config.ForEach != nil:
		vals := make(map[string]cty.Value)
		for key, instance := range instances {
			strKey, ok := key.(addrs.StringKey)
			if !ok {
				// old key that is being dropped and not used for evaluation
				continue
			}
			vals[string(strKey)] = instance
		}

		if len(vals) > 0 {
			// We use an object rather than a map here because resource schemas
			// may include dynamically-typed attributes, which will then cause
			// each instance to potentially have a different runtime type even
			// though they all conform to the static schema.
			ret = cty.ObjectVal(vals)
		} else {
			ret = cty.EmptyObjectVal
		}

	default:
		val, ok := instances[addrs.NoKey]
		if !ok {
			// if the instance is missing, insert an unknown value
			val = cty.UnknownVal(ty)
		}

		ret = val
	}

	return ret, diags
}

func (d *evaluationStateData) getEphemeralResource(addr addrs.Resource, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if d.Operation == walkValidate || d.Operation == walkEval {
		// Ephemeral instances are never live during the validate walk. Eval is
		// similarly offline, and since there is no value stored we can't return
		// anything other than dynamic.
		return cty.DynamicVal.Mark(marks.Ephemeral), diags
	}

	// Now, we're going to build up a value that represents the resource
	// or resources that are in the state.
	instances := map[addrs.InstanceKey]cty.Value{}

	// First, we're going to load any instances that we have written into the
	// deferrals system. A deferred resource overrides anything that might be
	// in the state for the resource, so we do this first.
	for key, value := range d.Evaluator.Deferrals.GetDeferredResourceInstances(addr.Absolute(d.ModulePath)) {
		instances[key] = value
	}

	absAddr := addr.Absolute(d.ModulePath)
	keyType, keys, haveUnknownKeys := d.Evaluator.Instances.ResourceInstanceKeys(absAddr)
	if haveUnknownKeys {
		// We can probably do better than totally unknown at least for a
		// single-instance resource, but we'll just keep it simple for now.
		// Result must be marked as ephemeral so that we can still catch
		// attempts to use the results in non-ephemeral locations, so that
		// the operator doesn't end up trapped with an error on a subsequent
		// plan/apply round.
		return cty.DynamicVal.Mark(marks.Ephemeral), diags
	}

	ephems := d.Evaluator.EphemeralResources
	getInstValue := func(addr addrs.AbsResourceInstance) (cty.Value, tfdiags.Diagnostics) {
		var diags tfdiags.Diagnostics

		// If we have a deferred instance with this key we don't need to check if it is live or not,
		// it has not been created so we can just return the deferred value.
		if v, ok := instances[addr.Resource.Key]; ok {
			return v, diags
		}

		val, isLive := ephems.InstanceValue(addr)
		if !isLive {
			// If the instance is no longer "live" by the time we're accessing
			// it then that suggests that it needed renewal and renewal has
			// failed, and so the object's value is no longer usable. We'll
			// still return the value in case it's somehow useful for diagnosis,
			// but we return an error to prevent further evaluation of whatever
			// other expression depended on the liveness of this object.
			//
			// This error message is written on the assumption that it will
			// always appear alongside the provider's renewal error, but that'll
			// be exposed only once the (now-zombied) ephemeral resource is
			// eventually closed, so that we can avoid returning the same error
			// multiple times.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Ephemeral resource instance has expired",
				Detail: fmt.Sprintf(
					"The remote object for %s is no longer available due to a renewal error, so Terraform cannot evaluate this expression.",
					addr,
				),
				Subject: rng.ToHCL().Ptr(),
			})
		}
		if val == cty.NilVal {
			val = cty.DynamicVal.Mark(marks.Ephemeral)
		}
		return val, diags
	}

	switch keyType {
	case addrs.NoKeyType:
		// For "no key" we're returning just a single object representing
		// the single instance of this resource.
		instVal, moreDiags := getInstValue(absAddr.Instance(addrs.NoKey))
		diags = diags.Append(moreDiags)
		return instVal, diags
	case addrs.IntKeyType:
		// For integer keys we're returning a tuple-typed value whose
		// indices are the keys.
		elems := make([]cty.Value, len(keys))
		for _, key := range keys {
			idx := int(key.(addrs.IntKey))
			instAddr := absAddr.Instance(key)
			instVal, moreDiags := getInstValue(instAddr)
			diags = diags.Append(moreDiags)
			elems[idx] = instVal
		}
		return cty.TupleVal(elems), diags
	case addrs.StringKeyType:
		// For string keys we're returning an object-typed value whose
		// attributes are the keys.
		attrs := make(map[string]cty.Value, len(keys))
		for _, key := range keys {
			attrName := string(key.(addrs.StringKey))
			instAddr := absAddr.Instance(key)
			instVal, moreDiags := getInstValue(instAddr)
			diags = diags.Append(moreDiags)
			attrs[attrName] = instVal
		}
		return cty.ObjectVal(attrs), diags
	default:
		panic(fmt.Sprintf("unhandled instance key type %#v", keyType))
	}
}

func (d *evaluationStateData) getResourceSchema(addr addrs.Resource, providerAddr addrs.Provider) *configschema.Block {
	schema, _, err := d.Evaluator.Plugins.ResourceTypeSchema(providerAddr, addr.Mode, addr.Type)
	if err != nil {
		// We have plently other codepaths that will detect and report
		// schema lookup errors before we'd reach this point, so we'll just
		// treat a failure here the same as having no schema.
		return nil
	}
	return schema
}

func (d *evaluationStateData) GetOutput(addr addrs.OutputValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// First we'll make sure the requested value is declared in configuration,
	// so we can produce a nice message if not.
	moduleConfig := d.Evaluator.Config.DescendantForInstance(d.ModulePath)
	if moduleConfig == nil {
		// should never happen, since we can't be evaluating in a module
		// that wasn't mentioned in configuration.
		panic(fmt.Sprintf("output value read from %s, which has no configuration", d.ModulePath))
	}

	config := moduleConfig.Module.Outputs[addr.Name]
	if config == nil {
		var suggestions []string
		for k := range moduleConfig.Module.Outputs {
			suggestions = append(suggestions, k)
		}
		suggestion := didyoumean.NameSuggestion(addr.Name, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared output value`,
			Detail:   fmt.Sprintf(`An output value with the name %q has not been declared.%s`, addr.Name, suggestion),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	output := d.Evaluator.State.OutputValue(addr.Absolute(d.ModulePath))
	if output == nil {
		// Then the output itself returned null, so we'll package that up and
		// pass it on.
		output = &states.OutputValue{
			Addr:      addr.Absolute(d.ModulePath),
			Value:     cty.NilVal,
			Sensitive: config.Sensitive,
		}
	} else if output.Value == cty.NilVal || output.Value.IsNull() {
		// Then we did get a value but Terraform itself thought it was NilVal
		// so we treat this as if the value isn't yet known.
		output.Value = cty.DynamicVal
	}

	val := output.Value
	if output.Sensitive {
		val = val.Mark(marks.Sensitive)
	}

	return val, diags
}

// moduleDisplayAddr returns a string describing the given module instance
// address that is appropriate for returning to users in situations where the
// root module is possible. Specifically, it returns "the root module" if the
// root module instance is given, or a string representation of the module
// address otherwise.
func moduleDisplayAddr(addr addrs.ModuleInstance) string {
	switch {
	case addr.IsRoot():
		return "the root module"
	default:
		return addr.String()
	}
}
