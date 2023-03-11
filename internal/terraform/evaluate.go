// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

	// NamedValues is where we keep the values of already-evaluated input
	// variables, local values, and output values.
	NamedValues *namedvals.State

	// Instances tracks the "expansion" (from one configuration object into
	// multiple instances of that object) of modules and resources. Other
	// parts of Terraform Core update this object during the graph walk,
	// and the evaluator uses it to return correctly-shaped data structures
	// representing those expandable objects.
	Instances *instances.Expander

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

	// AlternateStates allows callers to reference states from outside this
	// evaluator.
	//
	// The main use case here is for the testing framework to call into other
	// run blocks.
	AlternateStates map[string]*evaluationStateData

	PlanTimestamp time.Time
}

// Scope creates an evaluation scope for the given module path and optional
// resource.
//
// If the "self" argument is nil then the "self" object is not available
// in evaluated expressions. Otherwise, it behaves as an alias for the given
// address.
func (e *Evaluator) Scope(data lang.Data, self addrs.Referenceable, source addrs.Referenceable) *lang.Scope {
	return &lang.Scope{
		Data:          data,
		ParseRef:      addrs.ParseRef,
		SelfAddr:      self,
		SourceAddr:    source,
		PureOnly:      e.Operation != walkApply && e.Operation != walkDestroy && e.Operation != walkEval,
		BaseDir:       ".", // Always current working directory for now.
		PlanTimestamp: e.PlanTimestamp,
	}
}

// evaluationStateData is an implementation of lang.Data that resolves
// references primarily (but not exclusively) using information from a State.
type evaluationStateData struct {
	Evaluator *Evaluator

	// ModulePath is the path through the dynamic module tree to the module
	// that references will be resolved relative to.
	//
	// PartialExpandedModulePath is the same but for situations when we're
	// doing partial evaluation inside hypothetical instances of a
	// not-yet-fully-expanded module.
	//
	// These two are mutually exclusive, and PartialEval is true when we're
	// using PartialExpandedModulePath instead of ModulePath.
	ModulePath                addrs.ModuleInstance
	PartialExpandedModulePath addrs.PartialExpandedModule
	PartialEval               bool

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

func (d *evaluationStateData) staticModulePath() addrs.Module {
	if d.PartialEval {
		return d.PartialExpandedModulePath.Module()
	} else {
		return d.ModulePath.Module()
	}
}

func (d *evaluationStateData) modulePathDisplayAddr() string {
	var ret string
	if d.PartialEval {
		ret = d.PartialExpandedModulePath.String()
	} else {
		ret = d.ModulePath.String()
	}
	if ret == "" {
		return "root module"
	}
	return ret
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
	moduleConfig := d.Evaluator.Config.Descendent(d.staticModulePath())
	if moduleConfig == nil {
		// should never happen, since we can't be evaluating in a module
		// that wasn't mentioned in configuration.
		panic(fmt.Sprintf("input variable read from %s, which has no configuration", d.modulePathDisplayAddr()))
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
		// Ensure variable sensitivity is captured in the validate walk
		if config.Sensitive {
			return cty.UnknownVal(config.Type).Mark(marks.Sensitive), diags
		}
		return cty.UnknownVal(config.Type), diags
	}

	var val cty.Value
	if d.PartialEval {
		val = d.Evaluator.NamedValues.GetInputVariablePlaceholder(addrs.ObjectInPartialExpandedModule(d.PartialExpandedModulePath, addr))
	} else {
		val = d.Evaluator.NamedValues.GetInputVariableValue(d.ModulePath.InputVariable(addr.Name))
	}

	// Mark if sensitive
	if config.Sensitive {
		val = val.Mark(marks.Sensitive)
	}

	return val, diags
}

func (d *evaluationStateData) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// First we'll make sure the requested value is declared in configuration,
	// so we can produce a nice message if not.
	moduleConfig := d.Evaluator.Config.Descendent(d.staticModulePath())
	if moduleConfig == nil {
		// should never happen, since we can't be evaluating in a module
		// that wasn't mentioned in configuration.
		panic(fmt.Sprintf("local value read from %s, which has no configuration", d.modulePathDisplayAddr()))
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

	var val cty.Value
	if d.PartialEval {
		val = d.Evaluator.NamedValues.GetLocalValuePlaceholder(addrs.ObjectInPartialExpandedModule(d.PartialExpandedModulePath, addr))
	} else {
		val = d.Evaluator.NamedValues.GetLocalValue(addr.Absolute(d.ModulePath))
	}

	return val, diags
}

func (d *evaluationStateData) GetModule(addr addrs.ModuleCall, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Output results live in the module that declares them, which is one of
	// the child module instances of our current module path.
	moduleAddr := d.staticModulePath().Child(addr.Name)

	parentCfg := d.Evaluator.Config.Descendent(d.staticModulePath())
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
	moduleConfig := d.Evaluator.Config.Descendent(moduleAddr)
	if moduleConfig == nil {
		// should never happen, since we have a valid module call above, this
		// should be caught during static validation.
		panic(fmt.Sprintf("output value read from %s, which has no configuration", moduleAddr))
	}
	outputConfigs := moduleConfig.Module.Outputs

	// The module won't be expanded during validation, so we need to return an
	// unknown value. This will ensure the types looks correct, since we built
	// the objects based on the configuration.
	// FIXME: If we arrange for all module calls to have unknown expansion
	// during the validate walk then we could avoid this special case by
	// entering the d.PartialEval branch below instead, but we've retained
	// this special case for now to limit the size of the change that
	// introduced the partial evaluation mode.
	if d.Operation == walkValidate {
		attrTypes := make(map[string]cty.Type, len(outputConfigs))
		for name := range outputConfigs {
			attrTypes[name] = cty.DynamicPseudoType
		}

		// While we know the type here and it would be nice to validate whether
		// indexes are valid or not, because tuples and objects have fixed
		// numbers of elements we can't simply return an unknown value of the
		// same type since we have not expanded any instances during
		// validation.
		//
		// In order to validate the expression a little precisely, we'll create
		// an unknown map or list here to get more type information. This
		// is technically a little incorrect because in all other cases we
		// return object types instead of maps and tuple types instead of lists,
		// but the validation results don't carry forward into planning so
		// this small untruth won't typically hurt anything and gives a little
		// more information to the type checker during validation.
		ty := cty.Object(attrTypes)
		switch {
		case callConfig.Count != nil:
			return cty.UnknownVal(cty.List(ty)), diags
		case callConfig.ForEach != nil:
			return cty.UnknownVal(cty.Map(ty)), diags
		default:
			return cty.UnknownVal(ty), diags
		}
	}

	if d.PartialEval {
		// If we are under an unexpanded module then we cannot predict our
		// instance keys unless this is a singleton module call, in which
		// has it has only one "no-key" instance.
		if callConfig.Count != nil || callConfig.ForEach != nil {
			// A multi-instance module call appears as an object type or
			// tuple type, which means we can't say anything at all about
			// the type until we know exactly what the instance keys will be.
			// TODO: If we ever support explicitly-type-constrained output
			// values like in https://github.com/hashicorp/terraform/pull/31728
			// then we could return unknown values of list of object and map of
			// object types here in any case where all of the output values
			// are exactly specified, which would then enable better type
			// checking of uses in the calling module.
			return cty.DynamicVal, diags
		}

		// If we get here then this _is_ a singleton and our result will be
		// an object type built from the placeholder values that represent
		// the results for _all_ instances of this module call.
		attrs := make(map[string]cty.Value, len(outputConfigs))
		for name := range outputConfigs {
			outputAddr := addrs.OutputValue{Name: name}
			placeholderAddr := addrs.ObjectInPartialExpandedModule(d.PartialExpandedModulePath.Child(addr), outputAddr)
			attrs[name] = d.Evaluator.NamedValues.GetOutputValuePlaceholder(placeholderAddr)
		}
		return cty.ObjectVal(attrs), diags
	} else {
		keyType, knownKeys, unknownKeys := d.Evaluator.Instances.GetModuleCallInstanceKeys(addr.Absolute(d.ModulePath))
		if keyType == addrs.UnknownKeyType || unknownKeys {
			// If we don't know our full set of instance keys then we can't say
			// anything at all about the expected final result.
			return cty.DynamicVal, diags
		}

		switch keyType {
		case addrs.NoKeyType:
			// The following two conditions should always hold for for a module
			// call which has addrs.NoKeyType as its key type.
			if len(knownKeys) != 1 {
				panic(fmt.Sprintf("instance expander produced %d instance keys for a singleton module call", len(knownKeys)))
			}
			if knownKeys[0] != addrs.NoKey {
				panic(fmt.Sprintf("instance expander produced instance key %#v for a singleton module call", knownKeys[0]))
			}

			// Now we've proven that we do indeed have only one instance key
			// and it's NoKey, we can safely disregard knownKeys when building
			// the result.
			// The output values actually live in the child module instance,
			// so we need to extend our module instance address to include that.
			moduleAddr := d.ModulePath.Child(addr.Name, addrs.NoKey)
			attrs := d.moduleCallInstanceOutputValues(moduleAddr, outputConfigs)
			return cty.ObjectVal(attrs), diags

		case addrs.IntKeyType:
			// For IntKeyType the expander should return the keys in index order,
			// and we'll assume that while constructing this tuple.
			insts := make([]cty.Value, len(knownKeys))
			for i, instKey := range knownKeys {
				// The output values actually live in the child module instance,
				// so we need to extend our module instance address to include that.
				moduleAddr := d.ModulePath.Child(addr.Name, instKey)
				attrs := d.moduleCallInstanceOutputValues(moduleAddr, outputConfigs)
				insts[i] = cty.ObjectVal(attrs)
			}
			return cty.TupleVal(insts), diags

		case addrs.StringKeyType:
			insts := make(map[string]cty.Value, len(knownKeys))
			for _, instKey := range knownKeys {
				// The output values actually live in the child module instance,
				// so we need to extend our module instance address to include that.
				moduleAddr := d.ModulePath.Child(addr.Name, instKey)
				attrs := d.moduleCallInstanceOutputValues(moduleAddr, outputConfigs)
				insts[string(instKey.(addrs.StringKey))] = cty.ObjectVal(attrs)
			}
			return cty.ObjectVal(insts), diags

		default:
			// The above cases should be exhaustive for all key types except
			// for UnknownKeyType, which we handled in the if statement above.
			panic(fmt.Sprintf("unsupported instance key type %s", keyType))
		}
	}
}

func (d *evaluationStateData) moduleCallInstanceOutputValues(moduleAddr addrs.ModuleInstance, outputConfigs map[string]*configs.Output) map[string]cty.Value {
	attrs := make(map[string]cty.Value, len(outputConfigs))
	for name, cfg := range outputConfigs {
		outputAddr := addrs.OutputValue{Name: name}.Absolute(moduleAddr)
		// In situations where Terraform can prove that a particular expression
		// refers only to a single output value of a module instance the
		// graph allows evaluating the object representing a module before
		// all of the output values have been evaluated, so we must generate
		// placeholders for any that aren't ready yet. This oddity is just
		// so we can reuse the GetModule method as a "get individual output"
		// method rather than having to implement essentially the same logic
		// in two different ways.
		if !d.Evaluator.NamedValues.HasOutputValue(outputAddr) {
			attrs[name] = cty.DynamicVal
		} else {
			attrs[name] = d.Evaluator.NamedValues.GetOutputValue(outputAddr)
		}
		if cfg.Sensitive {
			attrs[name] = attrs[name].Mark(marks.Sensitive)
		}
	}
	return attrs
}

func (d *evaluationStateData) GetPathAttr(addr addrs.PathAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	switch addr.Name {

	case "cwd":
		var err error
		var wd string
		if d.Evaluator.Meta != nil {
			// Meta is always non-nil in the normal case, but some test cases
			// are not so realistic.
			wd = d.Evaluator.Meta.OriginalWorkingDir
		}
		if wd == "" {
			wd, err = os.Getwd()
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Failed to get working directory`,
					Detail:   fmt.Sprintf(`The value for path.cwd cannot be determined due to a system error: %s`, err),
					Subject:  rng.ToHCL().Ptr(),
				})
				return cty.DynamicVal, diags
			}
		}
		// The current working directory should always be absolute, whether we
		// just looked it up or whether we were relying on ContextMeta's
		// (possibly non-normalized) path.
		wd, err = filepath.Abs(wd)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Failed to get working directory`,
				Detail:   fmt.Sprintf(`The value for path.cwd cannot be determined due to a system error: %s`, err),
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.DynamicVal, diags
		}

		return cty.StringVal(filepath.ToSlash(wd)), diags

	case "module":
		moduleConfig := d.Evaluator.Config.Descendent(d.staticModulePath())
		if moduleConfig == nil {
			// should never happen, since we can't be evaluating in a module
			// that wasn't mentioned in configuration.
			panic(fmt.Sprintf("module.path read from module %s, which has no configuration", d.modulePathDisplayAddr()))
		}
		sourceDir := moduleConfig.Module.SourceDir
		return cty.StringVal(filepath.ToSlash(sourceDir)), diags

	case "root":
		sourceDir := d.Evaluator.Config.Module.SourceDir
		return cty.StringVal(filepath.ToSlash(sourceDir)), diags

	default:
		suggestion := didyoumean.NameSuggestion(addr.Name, []string{"cwd", "module", "root"})
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "path" attribute`,
			Detail:   fmt.Sprintf(`The "path" object does not have an attribute named %q.%s`, addr.Name, suggestion),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}
}

func (d *evaluationStateData) GetResource(addr addrs.Resource, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	// First we'll consult the configuration to see if an resource of this
	// name is declared at all.
	moduleConfig := d.Evaluator.Config.Descendent(d.staticModulePath())
	if moduleConfig == nil {
		// should never happen, since we can't be evaluating in a module
		// that wasn't mentioned in configuration.
		panic(fmt.Sprintf("resource value read from %s, which has no configuration", d.modulePathDisplayAddr()))
	}

	config := moduleConfig.Module.ResourceByAddr(addr)
	if config == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared resource`,
			Detail:   fmt.Sprintf(`A resource %q %q has not been declared in %s`, addr.Type, addr.Name, d.modulePathDisplayAddr()),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
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
				log.Printf("[ERROR] unknown instance %s in %s referenced during %s", addr, d.modulePathDisplayAddr(), d.Operation)
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
			// We should only end up here during the validate walk,
			// since later walks should have at least partial states populated
			// for all resources in the configuration.
			return cty.DynamicVal, diags
		}
	}

	// Decode all instances in the current state
	instances := map[addrs.InstanceKey]cty.Value{}
	pendingDestroy := d.Operation == walkDestroy
	for key, is := range rs.Instances {
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
				// If the object is in planned status then we should not get
				// here, since we should have found a pending value in the plan
				// above instead.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing pending object in plan",
					Detail:   fmt.Sprintf("Instance %s is marked as having a change pending but that change is not recorded in the plan. This is a bug in Terraform; please report it.", instAddr),
					Subject:  &config.DeclRange,
				})
				continue
			}
			val, err := change.After.Decode(ty)
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid resource instance data in plan",
					Detail:   fmt.Sprintf("Instance %s data could not be decoded from the plan: %s.", instAddr, err),
					Subject:  &config.DeclRange,
				})
				continue
			}

			// If our provider schema contains sensitive values, mark those as sensitive
			afterMarks := change.AfterValMarks
			if schema.ContainsSensitive() {
				afterMarks = append(afterMarks, schema.ValueMarks(val, nil)...)
			}

			instances[key] = val.MarkWithPaths(afterMarks)
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

		// If our schema contains sensitive values, mark those as sensitive.
		// Since decoding the instance object can also apply sensitivity marks,
		// we must remove and combine those before remarking to avoid a double-
		// mark error.
		if schema.ContainsSensitive() {
			var marks []cty.PathValueMarks
			val, marks = val.UnmarkDeepWithPaths()
			marks = append(marks, schema.ValueMarks(val, nil)...)
			val = val.MarkWithPaths(marks)
		}
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

func (d *evaluationStateData) GetTerraformAttr(addr addrs.TerraformAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	switch addr.Name {

	case "workspace":
		workspaceName := d.Evaluator.Meta.Env
		return cty.StringVal(workspaceName), diags

	case "env":
		// Prior to Terraform 0.12 there was an attribute "env", which was
		// an alias name for "workspace". This was deprecated and is now
		// removed.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "terraform" attribute`,
			Detail:   `The terraform.env attribute was deprecated in v0.10 and removed in v0.12. The "state environment" concept was renamed to "workspace" in v0.12, and so the workspace name can now be accessed using the terraform.workspace attribute.`,
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "terraform" attribute`,
			Detail:   fmt.Sprintf(`The "terraform" object does not have an attribute named %q. The only supported attribute is terraform.workspace, the name of the currently-selected workspace.`, addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}
}

func (d *evaluationStateData) GetOutput(addr addrs.OutputValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// First we'll make sure the requested value is declared in configuration,
	// so we can produce a nice message if not.
	moduleConfig := d.Evaluator.Config.DescendentForInstance(d.ModulePath)
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

func (d *evaluationStateData) GetCheckBlock(addr addrs.Check, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	// For now, check blocks don't contain any meaningful data and can only
	// be referenced from the testing scope within an expect_failures attribute.
	//
	// We've added them into the scope explicitly since they are referencable,
	// but we'll actually just return an error message saying they can't be
	// referenced in this context.
	var diags tfdiags.Diagnostics
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  "Reference to \"check\" in invalid context",
		Detail:   "The \"check\" object can only be referenced from an \"expect_failures\" attribute within a Terraform testing \"run\" block.",
		Subject:  rng.ToHCL().Ptr(),
	})
	return cty.NilVal, diags
}

func (d *evaluationStateData) GetRunBlock(run addrs.Run, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	data, exists := d.Evaluator.AlternateStates[run.Name]
	if !exists {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to unavailable run block",
			Detail:   fmt.Sprintf("The current test file either contains no %s, or hasn't executed it yet.", run),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	outputs := make(map[string]cty.Value)
	for _, outputCfg := range data.Evaluator.Config.Module.Outputs {
		output, outputDiags := data.GetOutput(outputCfg.Addr(), rng)
		diags = diags.Append(outputDiags)
		if outputDiags.HasErrors() {
			continue
		}
		outputs[outputCfg.Name] = output
	}

	return cty.ObjectVal(outputs), diags
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
