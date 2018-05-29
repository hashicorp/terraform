package terraform

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/agext/levenshtein"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/lang"
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

	// ProviderSchemas is a map of schemas for all provider configurations
	// that have been initialized so far. This is mutated concurrently, so
	// it must be accessed only while holding ProvidersLock.
	ProviderSchemas map[string]*ProviderSchema
	ProvidersLock   *sync.Mutex

	// State is the current state. During some operations this structure
	// is mutated concurrently, and so it must be accessed only while holding
	// StateLock.
	State     *State
	StateLock *sync.RWMutex
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

	// InstanceKey is the instance key for the object being evaluated, if any.
	// Set to addrs.NoKey if no object repetition is in progress.
	InstanceKey addrs.InstanceKey
}

// evaluationStateData must implement lang.Data
var _ lang.Data = (*evaluationStateData)(nil)

func (d *evaluationStateData) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	switch addr.Name {

	case "index":
		key := d.InstanceKey
		// key might not be set at all (addrs.NoKey) or it might be a string
		// if we're actually in a for_each block, so we'll check first and
		// produce a nice error if this is being used in the wrong context.
		intKey, ok := key.(addrs.IntKey)
		if !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Reference to "count" in non-counted context`,
				Detail:   fmt.Sprintf(`The "count" object can be used only in "resource" and "data" blocks, and only when the "count" argument is set.`),
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.UnknownVal(cty.Number), diags
		}
		return cty.NumberIntVal(int64(intKey)), diags

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

	// Now we'll retrieve the value from the state, which means we need to hold
	// the state lock.
	d.Evaluator.StateLock.RLock()
	defer d.Evaluator.StateLock.RUnlock()

	ms := d.Evaluator.State.ModuleByPath(d.ModulePath)
	if ms == nil {
		// Not evaluated yet?
		return cty.DynamicVal, diags
	}

	rawV, exists := ms.Locals[addr.Name]
	if !exists {
		// Not evaluated yet?
		return cty.DynamicVal, diags
	}

	// The state structures haven't yet been updated to the new type system,
	// so we'll need to shim here.
	// FIXME: Remove this once ms.Locals is itself a map[string]cty.Value.
	val := hcl2shim.HCL2ValueFromConfigValue(rawV)

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
		// should never happen, since we can't be evaluating in a module
		// that wasn't mentioned in configuration.
		panic(fmt.Sprintf("output value read from %s, which has no configuration", moduleAddr))
	}
	outputConfigs := moduleConfig.Module.Outputs

	// Now we'll retrieve the values from the state, which means we need to hold
	// the state lock.
	d.Evaluator.StateLock.RLock()
	defer d.Evaluator.StateLock.RUnlock()

	ms := d.Evaluator.State.ModuleByPath(moduleAddr)
	if ms == nil {
		// Not evaluated yet?
		// We'll return an unknown value of a suitable object type so that we
		// can still detect attempts to access outputs that aren't defined.
		attrs := map[string]cty.Type{}
		for name := range outputConfigs {
			attrs[name] = cty.DynamicPseudoType
		}
		return cty.UnknownVal(cty.Object(attrs)), diags
	}

	vals := map[string]cty.Value{}
	for name := range outputConfigs {
		os, exists := ms.Outputs[name]
		if !exists {
			// Not evaluated yet?
			vals[name] = cty.DynamicVal
			continue
		}

		// The state structures haven't yet been updated to the new type system,
		// so we'll need to shim here.
		// FIXME: Remove this once ms.Outputs itself contains cty.Value.
		vals[name] = hcl2shim.HCL2ValueFromConfigValue(os.Value)
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
		// should never happen, since we can't be evaluating in a module
		// that wasn't mentioned in configuration.
		panic(fmt.Sprintf("output value read from %s, which has no configuration", moduleAddr))
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
			Detail:   fmt.Sprintf(`An output value with the name %q has not been declared in %s.%s`, addr.Name, moduleAddr, suggestion),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	// Now we'll retrieve the value from the state, which means we need to hold
	// the state lock.
	d.Evaluator.StateLock.RLock()
	defer d.Evaluator.StateLock.RUnlock()

	ms := d.Evaluator.State.ModuleByPath(moduleAddr)
	if ms == nil {
		// Not evaluated yet?
		return cty.DynamicVal, diags
	}

	os, exists := ms.Outputs[addr.Name]
	if !exists {
		// Not evaluated yet?
		return cty.DynamicVal, diags
	}

	// The state structures haven't yet been updated to the new type system,
	// so we'll need to shim here.
	// FIXME: Remove this once ms.Outputs itself contains cty.Value.
	val := hcl2shim.HCL2ValueFromConfigValue(os.Value)

	return val, diags

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
		return cty.StringVal(wd), diags

	case "module":
		moduleConfig := d.Evaluator.Config.DescendentForInstance(d.ModulePath)
		if moduleConfig == nil {
			// should never happen, since we can't be evaluating in a module
			// that wasn't mentioned in configuration.
			panic(fmt.Sprintf("module.path read from module %s, which has no configuration", d.ModulePath))
		}
		sourceDir := moduleConfig.Module.SourceDir
		return cty.StringVal(sourceDir), diags

	case "root":
		sourceDir := d.Evaluator.Config.Module.SourceDir
		return cty.StringVal(sourceDir), diags

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

func (d *evaluationStateData) GetResourceInstance(addr addrs.ResourceInstance, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Although we are giving a ResourceInstance address here, if it has
	// a key of addrs.NoKey then it might actually be a request for all of
	// the instances of a particular resource. The reference resolver can't
	// resolve the ambiguity itself, so we must do it in here.

	// First we'll consult the configuration to see if an output of this
	// name is declared at all.
	moduleAddr := d.ModulePath
	moduleConfig := d.Evaluator.Config.DescendentForInstance(moduleAddr)
	if moduleConfig == nil {
		// should never happen, since we can't be evaluating in a module
		// that wasn't mentioned in configuration.
		panic(fmt.Sprintf("resource value read from %s, which has no configuration", moduleAddr))
	}

	config := moduleConfig.Module.ResourceByAddr(addr.ContainingResource())
	if config == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared resource`,
			Detail:   fmt.Sprintf(`A resource %q %q has not been declared in %s`, addr.Resource.Type, addr.Resource.Name, moduleAddr),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	// We need to shim our address to the legacy form still used in the state structs.
	addrKey := NewLegacyResourceInstanceAddress(addr.Absolute(d.ModulePath)).stateId()

	// We'll get the values for the instance(s) from state, so we'll need a read lock.
	d.Evaluator.StateLock.RLock()
	defer d.Evaluator.StateLock.RUnlock()

	ms := d.Evaluator.State.ModuleByPath(d.ModulePath)
	if ms == nil {
		// Not evaluated yet?
		return cty.DynamicVal, diags
	}

	// Note that the state structs currently have confusing legacy names:
	// ResourceState is actually the state for what we call an "instance"
	// elsewhere, and then InstanceState is the state for a particular _phase_
	// of that instance (primary vs. deposed). This should be addressed when
	// we revise the state structs to natively support the HCL type system.
	rs := ms.Resources[addrKey]

	var providerAddr addrs.AbsProviderConfig
	if rs != nil {
		var err error
		providerAddr, err = rs.ProviderAddr()
		if err != nil {
			// This indicates corruption of or tampering with the state file
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid provider address in state`,
				Detail:   fmt.Sprintf("The state for the referenced resource refers to a syntactically-invalid provider address %q. This can occur if the state data is incorrectly edited by hand.", rs.Provider),
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.DynamicVal, diags
		}
	} else {
		// Must assume a provider address from the config, then.
		// This result is usually ignored since we'll probably end up in
		// the getResourceInstancesAll path after this (if our instance
		// actually has a key). However, we can also end up here in strange
		// cases like "terraform console", which might be used before a
		// particular resource has been created in state at all.
		providerAddr = config.ProviderConfigAddr().Absolute(d.ModulePath)
	}

	// If we have an exact match for the requested instance and it has non-nil
	// primary data then we'll use it directly. This is the easy path.
	if rs != nil && rs.Primary != nil {
		log.Printf("[TRACE] GetResourceInstance: %s is a single instance", addr)
		return d.getResourceInstanceSingle(addr, rng, rs.Primary, providerAddr)
	}

	// If we get down here then we might have a request for the list of all
	// instances of a particular resource, but only if we have a no-key address.
	// If we have a _keyed_ address then instead it's a single instance that
	// isn't evaluated yet.
	if addr.Key != addrs.NoKey {
		log.Printf("[TRACE] GetResourceInstance: %s is pending", addr)
		return d.getResourceInstancePending(addr, rng, providerAddr)
	}

	return d.getResourceInstancesAll(addr.ContainingResource(), config, ms, providerAddr)
}

func (d *evaluationStateData) getResourceInstanceSingle(addr addrs.ResourceInstance, rng tfdiags.SourceRange, is *InstanceState, providerAddr addrs.AbsProviderConfig) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// To properly decode the "flatmap"-based values from the state, we need
	// to know the resource's schema, which we should already have cached
	// from when the provider was initialized.
	schema := d.getResourceSchema(addr.ContainingResource(), providerAddr)
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

	ty := schema.ImpliedType()
	if is == nil {
		// Assume we're dealing with an instance that hasn't been created yet.
		return cty.UnknownVal(ty), diags
	}

	flatmapVal := is.Attributes
	val, err := hcl2shim.HCL2ValueFromFlatmap(flatmapVal, ty)
	if err != nil {
		// A value in the flatmap value could not be conformed to the schema
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid value in state`,
			Detail:   fmt.Sprintf("The state data stored for %s does not conform to the resource schema: %s", addr, err),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.UnknownVal(ty), diags
	}

	return val, diags
}

func (d *evaluationStateData) getResourceInstancesAll(addr addrs.Resource, config *configs.Resource, ms *ModuleState, providerAddr addrs.AbsProviderConfig) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	rng := tfdiags.SourceRangeFromHCL(config.DeclRange)
	hasCount := config.Count != nil

	// Currently the only multi-instance construct we support is "count", which
	// ensures that all of the instances will have integer keys, and so we
	// can produce a tuple value of them.
	//
	// The legacy state structs are not designed to unambigiously represent
	// a list of instances associated with a resource, and so we need to infer
	// what exists based on which keys we find. Our returned tuple is therefore
	// long enough to accommodate the highest index we find, and may contain
	// unknown values filling in any "gaps" for instances that have been
	// tainted or not yet created.

	// Keys in the resources map are resource addresses followed by a period
	// and then an integer index. Keys without an integer index are possible
	// too, but we already took care of those in GetResourceInstance by
	// branching directly into getResourceInstanceSingle, so we know that
	// we're dealing with keyed instances here.
	prefix := addr.String() + "."
	length := 0
	instanceVals := map[addrs.InstanceKey]cty.Value{}
	for fullKey, rs := range ms.Resources {
		if !strings.HasPrefix(fullKey, prefix) {
			continue
		}
		if rs.Primary == nil {
			continue
		}

		keyStr := fullKey[len(prefix):]
		var key addrs.InstanceKey
		if i, err := strconv.Atoi(keyStr); err == nil {
			key = addrs.IntKey(i)
			if i >= length {
				length = i + 1
			}
		} else {
			key = addrs.StringKey(keyStr)
		}

		// In this case we'll ignore our given providerAddr, since it was
		// for a single unkeyed ResourceState, not the keyed one we have now.
		providerAddr, err := rs.ProviderAddr()
		if err != nil {
			// This indicates corruption of or tampering with the state file
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid provider address in state`,
				Detail:   fmt.Sprintf("The state for %s refers to a syntactically-invalid provider address %q. This can occur if the state data is incorrectly edited by hand.", addr.Instance(key), rs.Provider),
				Subject:  rng.ToHCL().Ptr(),
			})
			continue
		}

		val, instanceDiags := d.getResourceInstanceSingle(addr.Instance(key), rng, rs.Primary, providerAddr)
		diags = diags.Append(instanceDiags)

		instanceVals[key] = val
	}

	if length == 0 && !hasCount {
		// If we have nothing at all and the configuration lacks a count
		// argument then we'll assume that we're dealing with a resource that
		// is pending creation (e.g. during the validate walk) and that it
		// will eventually have only one unkeyed instance.
		// In this case we _do_ use the given providerAddr, since that
		// is for the unkeyed instance we found in GetResourceInstance.
		log.Printf("[TRACE] GetResourceInstance: %s has no instances yet", addr)
		return d.getResourceInstanceSingle(addr.Instance(addrs.NoKey), rng, nil, providerAddr)
	}

	log.Printf("[TRACE] GetResourceInstance: %s has multiple keyed instances (%d)", addr, length)

	// TODO: In future, when for_each is implemented, we'll need to decide here
	// whether to return a tuple value or an object value. However, by that
	// time we should've revised the state structs so we can see unambigously
	// which to use, rather than trying to guess based on the presence of
	// keys.

	valsSeq := make([]cty.Value, length)
	for i := 0; i < length; i++ {
		val, exists := instanceVals[addrs.IntKey(i)]
		if exists {
			valsSeq[i] = val
		} else {
			// FIXME: Ideally we'd return an unknown value of the schema's
			// implied type here, but this shim-ish implementation of resource
			// evaluation is already tricky enough so we'll just cheat for
			// now. Once we refactor for the new state format, reorganize this
			// code so that the schema is available here.
			valsSeq[i] = cty.DynamicVal // not yet known
		}
	}

	// We use a tuple rather than a list here because resource schemas may
	// include dynamically-typed attributes, which will then cause each
	// instance to potentially have a different runtime type.
	return cty.TupleVal(valsSeq), diags
}

func (d *evaluationStateData) getResourceInstancePending(addr addrs.ResourceInstance, rng tfdiags.SourceRange, providerAddr addrs.AbsProviderConfig) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// We'd ideally like to return a properly-typed unknown value here, in
	// order to give the type checker maximum information to detect type
	// mismatches even if concrete values aren't yet known.
	//
	// To do this we need to know the resource's schema, which we should
	// already have cached from when the provider was initialized.  However, we
	// first need to look in configuration to find out which provider address
	// will be responsible for creating this.
	moduleConfig := d.Evaluator.Config.DescendentForInstance(d.ModulePath)
	if moduleConfig == nil {
		// should never happen, since we can't be evaluating in a module
		// that wasn't mentioned in configuration.
		panic(fmt.Sprintf("reference to instance from %s, which has no configuration", d.ModulePath))
	}

	// Everything after here is best-effort: if we can't gather enough
	// information to return a typed value then we'll give up and return an
	// entirely-untyped value, assuming that we're in a special situation
	// such as accessing an orphaned resource, which should get error-checked
	// elsewhere.
	rc := moduleConfig.Module.ResourceByAddr(addr.ContainingResource())
	if rc == nil {
		return cty.DynamicVal, diags
	}
	schema := d.getResourceSchema(addr.ContainingResource(), providerAddr)
	if schema == nil {
		return cty.DynamicVal, diags
	}

	return cty.UnknownVal(schema.ImpliedType()), diags
}

func (d *evaluationStateData) getResourceSchema(addr addrs.Resource, providerAddr addrs.AbsProviderConfig) *configschema.Block {
	d.Evaluator.ProvidersLock.Lock()
	defer d.Evaluator.ProvidersLock.Unlock()

	log.Printf("[TRACE] Need provider schema for %s", providerAddr)
	providerSchema := d.Evaluator.ProviderSchemas[providerAddr.ProviderConfig.Type]
	if providerSchema == nil {
		return nil
	}

	var schema *configschema.Block
	switch addr.Mode {
	case addrs.ManagedResourceMode:
		schema = providerSchema.ResourceTypes[addr.Type]
	case addrs.DataResourceMode:
		schema = providerSchema.DataSources[addr.Type]
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
