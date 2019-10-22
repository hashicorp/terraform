package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/agext/levenshtein"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
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

	// VariableValues is a map from variable names to their associated values,
	// within the module indicated by ModulePath. VariableValues is modified
	// concurrently, and so it must be accessed only while holding
	// VariableValuesLock.
	//
	// The first map level is string representations of addr.ModuleInstance
	// values, while the second level is variable names.
	VariableValues     map[string]map[string]cty.Value
	VariableValuesLock *sync.Mutex

	// Schemas is a repository of all of the schemas we should need to
	// evaluate expressions. This must be constructed by the caller to
	// include schemas for all of the providers, resource types, data sources
	// and provisioners used by the given configuration and state.
	//
	// This must not be mutated during evaluation.
	Schemas *Schemas

	// State is the current state, embedded in a wrapper that ensures that
	// it can be safely accessed and modified concurrently.
	State *states.SyncState

	// Changes is the set of proposed changes, embedded in a wrapper that
	// ensures they can be safely accessed and modified concurrently.
	Changes *plans.ChangesSync
}

// Scope creates an evaluation scope for the given module path and optional
// resource.
//
// If the "self" argument is nil then the "self" object is not available
// in evaluated expressions. Otherwise, it behaves as an alias for the given
// address.
func (e *Evaluator) Scope(data lang.Data, self addrs.Referenceable) *lang.Scope {
	return &lang.Scope{
		Data:     data,
		SelfAddr: self,
		PureOnly: e.Operation != walkApply && e.Operation != walkDestroy,
		BaseDir:  ".", // Always current working directory for now.
	}
}

// evaluationStateData is an implementation of lang.Data that resolves
// references primarily (but not exclusively) using information from a State.
type evaluationStateData struct {
	Evaluator *Evaluator

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

// InstanceKeyEvalData is used during evaluation to specify which values,
// if any, should be produced for count.index, each.key, and each.value.
type InstanceKeyEvalData struct {
	// CountIndex is the value for count.index, or cty.NilVal if evaluating
	// in a context where the "count" argument is not active.
	//
	// For correct operation, this should always be of type cty.Number if not
	// nil.
	CountIndex cty.Value

	// EachKey and EachValue are the values for each.key and each.value
	// respectively, or cty.NilVal if evaluating in a context where the
	// "for_each" argument is not active. These must either both be set
	// or neither set.
	//
	// For correct operation, EachKey must always be either of type cty.String
	// or cty.Number if not nil.
	EachKey, EachValue cty.Value
}

// EvalDataForInstanceKey constructs a suitable InstanceKeyEvalData for
// evaluating in a context that has the given instance key.
func EvalDataForInstanceKey(key addrs.InstanceKey, forEachMap map[string]cty.Value) InstanceKeyEvalData {
	var countIdx cty.Value
	var eachKey cty.Value
	var eachVal cty.Value

	if intKey, ok := key.(addrs.IntKey); ok {
		countIdx = cty.NumberIntVal(int64(intKey))
	}

	if stringKey, ok := key.(addrs.StringKey); ok {
		eachKey = cty.StringVal(string(stringKey))
		eachVal = forEachMap[string(stringKey)]
	}

	return InstanceKeyEvalData{
		CountIndex: countIdx,
		EachKey:    eachKey,
		EachValue:  eachVal,
	}
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
				Detail:   fmt.Sprintf(`The "count" object can be used only in "resource" and "data" blocks, and only when the "count" argument is set.`),
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
				Summary:  `each.value is unknown and cannot be used in this context`,
				Detail:   fmt.Sprintf(`A reference to "each.value" has been used in a context in which it unavailable, such as after the config no longer contains the value in its "for_each" expression. Remove this reference to each.value in your config to work around this error.`),
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
			Detail:   fmt.Sprintf(`The "each" object can be used only in "resource" blocks, and only when the "for_each" argument is set.`),
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
	moduleConfig := d.Evaluator.Config.DescendentForInstance(d.ModulePath)
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
		suggestion := nameSuggestion(addr.Name, suggestions)
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

	wantType := cty.DynamicPseudoType
	if config.Type != cty.NilType {
		wantType = config.Type
	}

	d.Evaluator.VariableValuesLock.Lock()
	defer d.Evaluator.VariableValuesLock.Unlock()

	// During the validate walk, input variables are always unknown so
	// that we are validating the configuration for all possible input values
	// rather than for a specific set. Checking against a specific set of
	// input values then happens during the plan walk.
	//
	// This is important because otherwise the validation walk will tend to be
	// overly strict, requiring expressions throughout the configuration to
	// be complicated to accommodate all possible inputs, whereas returning
	// known here allows for simpler patterns like using input values as
	// guards to broadly enable/disable resources, avoid processing things
	// that are disabled, etc. Terraform's static validation leans towards
	// being liberal in what it accepts because the subsequent plan walk has
	// more information available and so can be more conservative.
	if d.Operation == walkValidate {
		return cty.UnknownVal(wantType), diags
	}

	moduleAddrStr := d.ModulePath.String()
	vals := d.Evaluator.VariableValues[moduleAddrStr]
	if vals == nil {
		return cty.UnknownVal(wantType), diags
	}

	val, isSet := vals[addr.Name]
	if !isSet {
		if config.Default != cty.NilVal {
			return config.Default, diags
		}
		return cty.UnknownVal(wantType), diags
	}

	var err error
	val, err = convert.Convert(val, wantType)
	if err != nil {
		// We should never get here because this problem should've been caught
		// during earlier validation, but we'll do something reasonable anyway.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Incorrect variable type`,
			Detail:   fmt.Sprintf(`The resolved value of variable %q is not appropriate: %s.`, addr.Name, err),
			Subject:  &config.DeclRange,
		})
		// Stub out our return value so that the semantic checker doesn't
		// produce redundant downstream errors.
		val = cty.UnknownVal(wantType)
	}

	return val, diags
}

func (d *evaluationStateData) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// First we'll make sure the requested value is declared in configuration,
	// so we can produce a nice message if not.
	moduleConfig := d.Evaluator.Config.DescendentForInstance(d.ModulePath)
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
		suggestion := nameSuggestion(addr.Name, suggestions)
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

	val := d.Evaluator.State.LocalValue(addr.Absolute(d.ModulePath))
	if val == cty.NilVal {
		// Not evaluated yet?
		val = cty.DynamicVal
	}

	return val, diags
}

func (d *evaluationStateData) GetModuleInstance(addr addrs.ModuleCallInstance, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Output results live in the module that declares them, which is one of
	// the child module instances of our current module path.
	moduleAddr := addr.ModuleInstance(d.ModulePath)

	// We'll consult the configuration to see what output names we are
	// expecting, so we can ensure the resulting object is of the expected
	// type even if our data is incomplete for some reason.
	moduleConfig := d.Evaluator.Config.DescendentForInstance(moduleAddr)
	if moduleConfig == nil {
		// should never happen, since this should've been caught during
		// static validation.
		panic(fmt.Sprintf("output value read from %s, which has no configuration", moduleAddr))
	}
	outputConfigs := moduleConfig.Module.Outputs

	vals := map[string]cty.Value{}
	for n := range outputConfigs {
		addr := addrs.OutputValue{Name: n}.Absolute(moduleAddr)

		// If a pending change is present in our current changeset then its value
		// takes priority over what's in state. (It will usually be the same but
		// will differ if the new value is unknown during planning.)
		if changeSrc := d.Evaluator.Changes.GetOutputChange(addr); changeSrc != nil {
			change, err := changeSrc.Decode()
			if err != nil {
				// This should happen only if someone has tampered with a plan
				// file, so we won't bother with a pretty error for it.
				diags = diags.Append(fmt.Errorf("planned change for %s could not be decoded: %s", addr, err))
				vals[n] = cty.DynamicVal
				continue
			}
			// We care only about the "after" value, which is the value this output
			// will take on after the plan is applied.
			vals[n] = change.After
		} else {
			os := d.Evaluator.State.OutputValue(addr)
			if os == nil {
				// Not evaluated yet?
				vals[n] = cty.DynamicVal
				continue
			}
			vals[n] = os.Value
		}
	}
	return cty.ObjectVal(vals), diags
}

func (d *evaluationStateData) GetModuleInstanceOutput(addr addrs.ModuleCallOutput, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Output results live in the module that declares them, which is one of
	// the child module instances of our current module path.
	absAddr := addr.AbsOutputValue(d.ModulePath)
	moduleAddr := absAddr.Module

	// First we'll consult the configuration to see if an output of this
	// name is declared at all.
	moduleConfig := d.Evaluator.Config.DescendentForInstance(moduleAddr)
	if moduleConfig == nil {
		// this doesn't happen in normal circumstances due to our validation
		// pass, but it can turn up in some unusual situations, like in the
		// "terraform console" repl where arbitrary expressions can be
		// evaluated.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared module`,
			Detail:   fmt.Sprintf(`The configuration contains no %s.`, moduleAddr),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	config := moduleConfig.Module.Outputs[addr.Name]
	if config == nil {
		var suggestions []string
		for k := range moduleConfig.Module.Outputs {
			suggestions = append(suggestions, k)
		}
		suggestion := nameSuggestion(addr.Name, suggestions)
		if suggestion != "" {
			suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared output value`,
			Detail:   fmt.Sprintf(`An output value with the name %q has not been declared in %s.%s`, addr.Name, moduleDisplayAddr(moduleAddr), suggestion),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	// If a pending change is present in our current changeset then its value
	// takes priority over what's in state. (It will usually be the same but
	// will differ if the new value is unknown during planning.)
	if changeSrc := d.Evaluator.Changes.GetOutputChange(absAddr); changeSrc != nil {
		change, err := changeSrc.Decode()
		if err != nil {
			// This should happen only if someone has tampered with a plan
			// file, so we won't bother with a pretty error for it.
			diags = diags.Append(fmt.Errorf("planned change for %s could not be decoded: %s", absAddr, err))
			return cty.DynamicVal, diags
		}
		// We care only about the "after" value, which is the value this output
		// will take on after the plan is applied.
		return change.After, diags
	}

	os := d.Evaluator.State.OutputValue(absAddr)
	if os == nil {
		// Not evaluated yet?
		return cty.DynamicVal, diags
	}

	return os.Value, diags
}

func (d *evaluationStateData) GetPathAttr(addr addrs.PathAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	switch addr.Name {

	case "cwd":
		wd, err := os.Getwd()
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
		moduleConfig := d.Evaluator.Config.DescendentForInstance(d.ModulePath)
		if moduleConfig == nil {
			// should never happen, since we can't be evaluating in a module
			// that wasn't mentioned in configuration.
			panic(fmt.Sprintf("module.path read from module %s, which has no configuration", d.ModulePath))
		}
		sourceDir := moduleConfig.Module.SourceDir
		return cty.StringVal(filepath.ToSlash(sourceDir)), diags

	case "root":
		sourceDir := d.Evaluator.Config.Module.SourceDir
		return cty.StringVal(filepath.ToSlash(sourceDir)), diags

	default:
		suggestion := nameSuggestion(addr.Name, []string{"cwd", "module", "root"})
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
	moduleAddr := d.ModulePath
	moduleConfig := d.Evaluator.Config.DescendentForInstance(moduleAddr)
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

	rs := d.Evaluator.State.Resource(addr.Absolute(d.ModulePath))

	if rs == nil {
		// we must return DynamicVal so that both interpretations
		// can proceed without generating errors, and we'll deal with this
		// in a later step where more information is gathered.
		// (In practice we should only end up here during the validate walk,
		// since later walks should have at least partial states populated
		// for all resources in the configuration.)
		return cty.DynamicVal, diags
	}

	// Break out early during validation, because resource may not be expanded
	// yet and indexed references may show up as invalid.
	if d.Operation == walkValidate {
		return cty.DynamicVal, diags
	}

	return d.getResourceInstancesAll(addr, rng, config, rs, rs.ProviderConfig)
}

func (d *evaluationStateData) getResourceInstancesAll(addr addrs.Resource, rng tfdiags.SourceRange, config *configs.Resource, rs *states.Resource, providerAddr addrs.AbsProviderConfig) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	instAddr := addrs.ResourceInstance{Resource: addr, Key: addrs.NoKey}

	schema := d.getResourceSchema(addr, providerAddr)
	if schema == nil {
		// This shouldn't happen, since validation before we get here should've
		// taken care of it, but we'll show a reasonable error message anyway.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Missing resource type schema`,
			Detail:   fmt.Sprintf("No schema is available for %s in %s. This is a bug in Terraform and should be reported.", addr, providerAddr),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	switch rs.EachMode {
	case states.NoEach:
		ty := schema.ImpliedType()
		is := rs.Instances[addrs.NoKey]
		if is == nil || is.Current == nil {
			// Assume we're dealing with an instance that hasn't been created yet.
			return cty.UnknownVal(ty), diags
		}

		if is.Current.Status == states.ObjectPlanned {
			// If there's a pending change for this instance in our plan, we'll prefer
			// that. This is important because the state can't represent unknown values
			// and so its data is inaccurate when changes are pending.
			if change := d.Evaluator.Changes.GetResourceInstanceChange(instAddr.Absolute(d.ModulePath), states.CurrentGen); change != nil {
				val, err := change.After.Decode(ty)
				if err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid resource instance data in plan",
						Detail:   fmt.Sprintf("Instance %s data could not be decoded from the plan: %s.", addr.Absolute(d.ModulePath), err),
						Subject:  &config.DeclRange,
					})
					return cty.UnknownVal(ty), diags
				}
				return val, diags
			} else {
				// If the object is in planned status then we should not
				// get here, since we should've found a pending value
				// in the plan above instead.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Missing pending object in plan",
					Detail:   fmt.Sprintf("Instance %s is marked as having a change pending but that change is not recorded in the plan. This is a bug in Terraform; please report it.", addr),
					Subject:  &config.DeclRange,
				})
				return cty.UnknownVal(ty), diags
			}
		}

		ios, err := is.Current.Decode(ty)
		if err != nil {
			// This shouldn't happen, since by the time we get here
			// we should've upgraded the state data already.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid resource instance data in state",
				Detail:   fmt.Sprintf("Instance %s data could not be decoded from the state: %s.", addr.Absolute(d.ModulePath), err),
				Subject:  &config.DeclRange,
			})
			return cty.UnknownVal(ty), diags
		}

		return ios.Value, diags

	case states.EachList:
		// We need to infer the length of our resulting tuple by searching
		// for the max IntKey in our instances map.
		length := 0
		for k := range rs.Instances {
			if ik, ok := k.(addrs.IntKey); ok {
				if int(ik) >= length {
					length = int(ik) + 1
				}
			}
		}

		vals := make([]cty.Value, length)
		for i := 0; i < length; i++ {
			ty := schema.ImpliedType()
			key := addrs.IntKey(i)
			is, exists := rs.Instances[key]
			if exists && is.Current != nil {
				instAddr := addr.Instance(key).Absolute(d.ModulePath)

				// Prefer pending value in plan if present. See getResourceInstanceSingle
				// comment for the rationale.
				if is.Current.Status == states.ObjectPlanned {
					if change := d.Evaluator.Changes.GetResourceInstanceChange(instAddr, states.CurrentGen); change != nil {
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
						vals[i] = val
						continue
					} else {
						// If the object is in planned status then we should not
						// get here, since we should've found a pending value
						// in the plan above instead.
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Missing pending object in plan",
							Detail:   fmt.Sprintf("Instance %s is marked as having a change pending but that change is not recorded in the plan. This is a bug in Terraform; please report it.", instAddr),
							Subject:  &config.DeclRange,
						})
						continue
					}
				}

				ios, err := is.Current.Decode(ty)
				if err != nil {
					// This shouldn't happen, since by the time we get here
					// we should've upgraded the state data already.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid resource instance data in state",
						Detail:   fmt.Sprintf("Instance %s data could not be decoded from the state: %s.", instAddr, err),
						Subject:  &config.DeclRange,
					})
					continue
				}
				vals[i] = ios.Value
			} else {
				// There shouldn't normally be "gaps" in our list but we'll
				// allow it under the assumption that we're in a weird situation
				// where e.g. someone has run "terraform state mv" to reorder
				// a list and left a hole behind.
				vals[i] = cty.UnknownVal(schema.ImpliedType())
			}
		}

		// We use a tuple rather than a list here because resource schemas may
		// include dynamically-typed attributes, which will then cause each
		// instance to potentially have a different runtime type even though
		// they all conform to the static schema.
		return cty.TupleVal(vals), diags

	case states.EachMap:
		ty := schema.ImpliedType()
		vals := make(map[string]cty.Value, len(rs.Instances))
		for k, is := range rs.Instances {
			if sk, ok := k.(addrs.StringKey); ok {
				instAddr := addr.Instance(k).Absolute(d.ModulePath)

				// Prefer pending value in plan if present. See getResourceInstanceSingle
				// comment for the rationale.
				// Prefer pending value in plan if present. See getResourceInstanceSingle
				// comment for the rationale.
				if is.Current.Status == states.ObjectPlanned {
					if change := d.Evaluator.Changes.GetResourceInstanceChange(instAddr, states.CurrentGen); change != nil {
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
						vals[string(sk)] = val
						continue
					} else {
						// If the object is in planned status then we should not
						// get here, since we should've found a pending value
						// in the plan above instead.
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Missing pending object in plan",
							Detail:   fmt.Sprintf("Instance %s is marked as having a change pending but that change is not recorded in the plan. This is a bug in Terraform; please report it.", instAddr),
							Subject:  &config.DeclRange,
						})
						continue
					}
				}

				ios, err := is.Current.Decode(ty)
				if err != nil {
					// This shouldn't happen, since by the time we get here
					// we should've upgraded the state data already.
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid resource instance data in state",
						Detail:   fmt.Sprintf("Instance %s data could not be decoded from the state: %s.", instAddr, err),
						Subject:  &config.DeclRange,
					})
					continue
				}
				vals[string(sk)] = ios.Value
			}
		}

		// We use an object rather than a map here because resource schemas may
		// include dynamically-typed attributes, which will then cause each
		// instance to potentially have a different runtime type even though
		// they all conform to the static schema.
		return cty.ObjectVal(vals), diags

	default:
		// Should never happen since caller should deal with other modes
		panic(fmt.Sprintf("unsupported EachMode %s", rs.EachMode))
	}
}

func (d *evaluationStateData) getResourceSchema(addr addrs.Resource, providerAddr addrs.AbsProviderConfig) *configschema.Block {
	providerType := providerAddr.ProviderConfig.Type
	schemas := d.Evaluator.Schemas
	schema, _ := schemas.ResourceTypeConfig(providerType, addr.Mode, addr.Type)
	return schema
}

// coerceInstanceKey attempts to convert the given key to the type expected
// for the given EachMode.
//
// If the key is already of the correct type or if it cannot be converted then
// it is returned verbatim. If conversion is required and possible, the
// converted value is returned. Callers should not try to determine if
// conversion was possible, should instead just check if the result is of
// the expected type.
func (d *evaluationStateData) coerceInstanceKey(key addrs.InstanceKey, mode states.EachMode) addrs.InstanceKey {
	if key == addrs.NoKey {
		// An absent key can't be converted
		return key
	}

	switch mode {
	case states.NoEach:
		// No conversions possible at all
		return key
	case states.EachMap:
		if intKey, isInt := key.(addrs.IntKey); isInt {
			return addrs.StringKey(strconv.Itoa(int(intKey)))
		}
		return key
	case states.EachList:
		if strKey, isStr := key.(addrs.StringKey); isStr {
			i, err := strconv.Atoi(string(strKey))
			if err != nil {
				return key
			}
			return addrs.IntKey(i)
		}
		return key
	default:
		return key
	}
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
			Detail:   `The terraform.env attribute was deprecated in v0.10 and removed in v0.12. The "state environment" concept was rename to "workspace" in v0.12, and so the workspace name can now be accessed using the terraform.workspace attribute.`,
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

// nameSuggestion tries to find a name from the given slice of suggested names
// that is close to the given name and returns it if found. If no suggestion
// is close enough, returns the empty string.
//
// The suggestions are tried in order, so earlier suggestions take precedence
// if the given string is similar to two or more suggestions.
//
// This function is intended to be used with a relatively-small number of
// suggestions. It's not optimized for hundreds or thousands of them.
func nameSuggestion(given string, suggestions []string) string {
	for _, suggestion := range suggestions {
		dist := levenshtein.Distance(given, suggestion, nil)
		if dist < 3 { // threshold determined experimentally
			return suggestion
		}
	}
	return ""
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
