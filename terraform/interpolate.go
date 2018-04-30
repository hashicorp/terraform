package terraform

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/flatmap"
)

const (
	// VarEnvPrefix is the prefix of variables that are read from
	// the environment to set variables here.
	VarEnvPrefix = "TF_VAR_"
)

// Interpolater is the structure responsible for determining the values
// for interpolations such as `aws_instance.foo.bar`.
type Interpolater struct {
	Operation          walkOperation
	Meta               *ContextMeta
	Module             *module.Tree
	State              *State
	StateLock          *sync.RWMutex
	VariableValues     map[string]interface{}
	VariableValuesLock *sync.Mutex
}

// InterpolationScope is the current scope of execution. This is required
// since some variables which are interpolated are dependent on what we're
// operating on and where we are.
type InterpolationScope struct {
	Path     []string
	Resource *Resource
}

// Values returns the values for all the variables in the given map.
func (i *Interpolater) Values(
	scope *InterpolationScope,
	vars map[string]config.InterpolatedVariable) (map[string]ast.Variable, error) {
	return nil, fmt.Errorf("type Interpolator is no longer supported; use the evaluator API instead")
}

func (i *Interpolater) valueCountVar(
	scope *InterpolationScope,
	n string,
	v *config.CountVariable,
	result map[string]ast.Variable) error {
	switch v.Type {
	case config.CountValueIndex:
		if scope.Resource == nil {
			return fmt.Errorf("%s: count.index is only valid within resources", n)
		}
		result[n] = ast.Variable{
			Value: scope.Resource.CountIndex,
			Type:  ast.TypeInt,
		}
		return nil
	default:
		return fmt.Errorf("%s: unknown count type: %#v", n, v.Type)
	}
}

func unknownVariable() ast.Variable {
	return ast.Variable{
		Type:  ast.TypeUnknown,
		Value: config.UnknownVariableValue,
	}
}

func unknownValue() string {
	return hil.UnknownValue
}

func (i *Interpolater) valueModuleVar(
	scope *InterpolationScope,
	n string,
	v *config.ModuleVariable,
	result map[string]ast.Variable) error {
	// Build the path to the child module we want
	path := make([]string, len(scope.Path), len(scope.Path)+1)
	copy(path, scope.Path)
	path = append(path, v.Name)

	// Grab the lock so that if other interpolations are running or
	// state is being modified, we'll be safe.
	i.StateLock.RLock()
	defer i.StateLock.RUnlock()

	// Get the module where we're looking for the value
	mod := i.State.ModuleByPath(normalizeModulePath(path))
	if mod == nil {
		// If the module doesn't exist, then we can return an empty string.
		// This happens usually only in Refresh() when we haven't populated
		// a state. During validation, we semantically verify that all
		// modules reference other modules, and graph ordering should
		// ensure that the module is in the state, so if we reach this
		// point otherwise it really is a panic.
		result[n] = unknownVariable()

		// During apply this is always an error
		if i.Operation == walkApply {
			return fmt.Errorf(
				"Couldn't find module %q for var: %s",
				v.Name, v.FullKey())
		}
	} else {
		// Get the value from the outputs
		if outputState, ok := mod.Outputs[v.Field]; ok {
			output, err := hil.InterfaceToVariable(outputState.Value)
			if err != nil {
				return err
			}
			result[n] = output
		} else {
			// Same reasons as the comment above.
			result[n] = unknownVariable()

			// During apply this is always an error
			if i.Operation == walkApply {
				return fmt.Errorf(
					"Couldn't find output %q for module var: %s",
					v.Field, v.FullKey())
			}
		}
	}

	return nil
}

func (i *Interpolater) valuePathVar(
	scope *InterpolationScope,
	n string,
	v *config.PathVariable,
	result map[string]ast.Variable) error {
	switch v.Type {
	case config.PathValueCwd:
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf(
				"Couldn't get cwd for var %s: %s",
				v.FullKey(), err)
		}

		result[n] = ast.Variable{
			Value: wd,
			Type:  ast.TypeString,
		}
	case config.PathValueModule:
		if t := i.Module.Child(scope.Path[1:]); t != nil {
			result[n] = ast.Variable{
				Value: t.Config().Dir,
				Type:  ast.TypeString,
			}
		}
	case config.PathValueRoot:
		result[n] = ast.Variable{
			Value: i.Module.Config().Dir,
			Type:  ast.TypeString,
		}
	default:
		return fmt.Errorf("%s: unknown path type: %#v", n, v.Type)
	}

	return nil

}

func (i *Interpolater) valueResourceVar(
	scope *InterpolationScope,
	n string,
	v *config.ResourceVariable,
	result map[string]ast.Variable) error {
	// If we're computing all dynamic fields, then module vars count
	// and we mark it as computed.
	if i.Operation == walkValidate {
		result[n] = unknownVariable()
		return nil
	}

	var variable *ast.Variable
	var err error

	if v.Multi && v.Index == -1 {
		variable, err = i.computeResourceMultiVariable(scope, v)
	} else {
		variable, err = i.computeResourceVariable(scope, v)
	}

	if err != nil {
		return err
	}

	if variable == nil {
		// During the input walk we tolerate missing variables because
		// we haven't yet had a chance to refresh state, so dynamic data may
		// not yet be complete.
		// If it truly is missing, we'll catch it on a later walk.
		// This applies only to graph nodes that interpolate during the
		// config walk, e.g. providers.
		if i.Operation == walkInput || i.Operation == walkRefresh {
			result[n] = unknownVariable()
			return nil
		}

		return fmt.Errorf("variable %q is nil, but no error was reported", v.Name)
	}

	result[n] = *variable
	return nil
}

func (i *Interpolater) valueSelfVar(
	scope *InterpolationScope,
	n string,
	v *config.SelfVariable,
	result map[string]ast.Variable) error {
	if scope == nil || scope.Resource == nil {
		return fmt.Errorf(
			"%s: invalid scope, self variables are only valid on resources", n)
	}

	rv, err := config.NewResourceVariable(fmt.Sprintf(
		"%s.%s.%d.%s",
		scope.Resource.Type,
		scope.Resource.Name,
		scope.Resource.CountIndex,
		v.Field))
	if err != nil {
		return err
	}

	return i.valueResourceVar(scope, n, rv, result)
}

func (i *Interpolater) valueSimpleVar(
	scope *InterpolationScope,
	n string,
	v *config.SimpleVariable,
	result map[string]ast.Variable) error {
	// This error message includes some information for people who
	// relied on this for their template_file data sources. We should
	// remove this at some point but there isn't any rush.
	return fmt.Errorf(
		"invalid variable syntax: %q. Did you mean 'var.%s'? If this is part of inline `template` parameter\n"+
			"then you must escape the interpolation with two dollar signs. For\n"+
			"example: ${a} becomes $${a}.",
		n, n)
}

func (i *Interpolater) valueTerraformVar(
	scope *InterpolationScope,
	n string,
	v *config.TerraformVariable,
	result map[string]ast.Variable) error {
	// "env" is supported for backward compatibility, but it's deprecated and
	// so we won't advertise it as being allowed in the error message. It will
	// be removed in a future version of Terraform.
	if v.Field != "workspace" && v.Field != "env" {
		return fmt.Errorf(
			"%s: only supported key for 'terraform.X' interpolations is 'workspace'", n)
	}

	if i.Meta == nil {
		return fmt.Errorf(
			"%s: internal error: nil Meta. Please report a bug.", n)
	}

	result[n] = ast.Variable{Type: ast.TypeString, Value: i.Meta.Env}
	return nil
}

func (i *Interpolater) valueLocalVar(
	scope *InterpolationScope,
	n string,
	v *config.LocalVariable,
	result map[string]ast.Variable,
) error {
	i.StateLock.RLock()
	defer i.StateLock.RUnlock()

	modTree := i.Module
	if len(scope.Path) > 1 {
		modTree = i.Module.Child(scope.Path[1:])
	}

	// Get the resource from the configuration so we can verify
	// that the resource is in the configuration and so we can access
	// the configuration if we need to.
	var cl *config.Local
	for _, l := range modTree.Config().Locals {
		if l.Name == v.Name {
			cl = l
			break
		}
	}

	if cl == nil {
		return fmt.Errorf("%s: no local value of this name has been declared", n)
	}

	// Get the relevant module
	module := i.State.ModuleByPath(normalizeModulePath(scope.Path))
	if module == nil {
		result[n] = unknownVariable()
		return nil
	}

	rawV, exists := module.Locals[v.Name]
	if !exists {
		result[n] = unknownVariable()
		return nil
	}

	varV, err := hil.InterfaceToVariable(rawV)
	if err != nil {
		// Should never happen, since interpolation should always produce
		// something we can feed back in to interpolation.
		return fmt.Errorf("%s: %s", n, err)
	}

	result[n] = varV
	return nil
}

func (i *Interpolater) valueUserVar(
	scope *InterpolationScope,
	n string,
	v *config.UserVariable,
	result map[string]ast.Variable) error {
	i.VariableValuesLock.Lock()
	defer i.VariableValuesLock.Unlock()
	val, ok := i.VariableValues[v.Name]
	if ok {
		varValue, err := hil.InterfaceToVariable(val)
		if err != nil {
			return fmt.Errorf("cannot convert %s value %q to an ast.Variable for interpolation: %s",
				v.Name, val, err)
		}
		result[n] = varValue
		return nil
	}

	if _, ok := result[n]; !ok && i.Operation == walkValidate {
		result[n] = unknownVariable()
		return nil
	}

	// Look up if we have any variables with this prefix because
	// those are map overrides. Include those.
	for k, val := range i.VariableValues {
		if strings.HasPrefix(k, v.Name+".") {
			keyComponents := strings.Split(k, ".")
			overrideKey := keyComponents[len(keyComponents)-1]

			mapInterface, ok := result["var."+v.Name]
			if !ok {
				return fmt.Errorf("override for non-existent variable: %s", v.Name)
			}

			mapVariable := mapInterface.Value.(map[string]ast.Variable)

			varValue, err := hil.InterfaceToVariable(val)
			if err != nil {
				return fmt.Errorf("cannot convert %s value %q to an ast.Variable for interpolation: %s",
					v.Name, val, err)
			}
			mapVariable[overrideKey] = varValue
		}
	}

	return nil
}

func (i *Interpolater) computeResourceVariable(
	scope *InterpolationScope,
	v *config.ResourceVariable) (*ast.Variable, error) {
	id := v.ResourceId()
	if v.Multi {
		id = fmt.Sprintf("%s.%d", id, v.Index)
	}

	i.StateLock.RLock()
	defer i.StateLock.RUnlock()

	unknownVariable := unknownVariable()

	// These variables must be declared early because of the use of GOTO
	var isList bool
	var isMap bool

	// Get the information about this resource variable, and verify
	// that it exists and such.
	module, cr, err := i.resourceVariableInfo(scope, v)
	if err != nil {
		return nil, err
	}

	// If we're requesting "count" its a special variable that we grab
	// directly from the config itself.
	if v.Field == "count" {
		var count int
		if cr != nil {
			count, err = cr.Count()
		} else {
			count, err = i.resourceCountMax(module, cr, v)
		}
		if err != nil {
			return nil, fmt.Errorf(
				"Error reading %s count: %s",
				v.ResourceId(),
				err)
		}

		return &ast.Variable{Type: ast.TypeInt, Value: count}, nil
	}

	// Get the resource out from the state. We know the state exists
	// at this point and if there is a state, we expect there to be a
	// resource with the given name.
	var r *ResourceState
	if module != nil && len(module.Resources) > 0 {
		var ok bool
		r, ok = module.Resources[id]
		if !ok && v.Multi && v.Index == 0 {
			r, ok = module.Resources[v.ResourceId()]
		}
		if !ok {
			r = nil
		}
	}
	if r == nil || r.Primary == nil {
		if i.Operation == walkApply || i.Operation == walkPlan {
			return nil, fmt.Errorf(
				"Resource '%s' not found for variable '%s'",
				v.ResourceId(),
				v.FullKey())
		}

		// If we have no module in the state yet or count, return empty.
		// NOTE(@mitchellh): I actually don't know why this is here. During
		// a refactor I kept this here to maintain the same behavior, but
		// I'm not sure why its here.
		if module == nil || len(module.Resources) == 0 {
			return nil, nil
		}

		goto MISSING
	}

	if attr, ok := r.Primary.Attributes[v.Field]; ok {
		v, err := hil.InterfaceToVariable(attr)
		return &v, err
	}

	// special case for the "id" field which is usually also an attribute
	if v.Field == "id" && r.Primary.ID != "" {
		// This is usually pulled from the attributes, but is sometimes missing
		// during destroy. We can return the ID field in this case.
		// FIXME: there should only be one ID to rule them all.
		log.Printf("[WARN] resource %s missing 'id' attribute", v.ResourceId())
		v, err := hil.InterfaceToVariable(r.Primary.ID)
		return &v, err
	}

	// computed list or map attribute
	_, isList = r.Primary.Attributes[v.Field+".#"]
	_, isMap = r.Primary.Attributes[v.Field+".%"]
	if isList || isMap {
		variable, err := i.interpolateComplexTypeAttribute(v.Field, r.Primary.Attributes)
		return &variable, err
	}

	// At apply time, we can't do the "maybe has it" check below
	// that we need for plans since parent elements might be computed.
	// Therefore, it is an error and we're missing the key.
	//
	// TODO: test by creating a state and configuration that is referencing
	// a non-existent variable "foo.bar" where the state only has "foo"
	// and verify plan works, but apply doesn't.
	if i.Operation == walkApply || i.Operation == walkDestroy {
		goto MISSING
	}

	// We didn't find the exact field, so lets separate the dots
	// and see if anything along the way is a computed set. i.e. if
	// we have "foo.0.bar" as the field, check to see if "foo" is
	// a computed list. If so, then the whole thing is computed.
	if parts := strings.Split(v.Field, "."); len(parts) > 1 {
		for i := 1; i < len(parts); i++ {
			// Lists and sets make this
			key := fmt.Sprintf("%s.#", strings.Join(parts[:i], "."))
			if attr, ok := r.Primary.Attributes[key]; ok {
				v, err := hil.InterfaceToVariable(attr)
				return &v, err
			}

			// Maps make this
			key = fmt.Sprintf("%s", strings.Join(parts[:i], "."))
			if attr, ok := r.Primary.Attributes[key]; ok {
				v, err := hil.InterfaceToVariable(attr)
				return &v, err
			}
		}
	}

MISSING:
	// Validation for missing interpolations should happen at a higher
	// semantic level. If we reached this point and don't have variables,
	// just return the computed value.
	if scope == nil && scope.Resource == nil {
		return &unknownVariable, nil
	}

	// If the operation is refresh, it isn't an error for a value to
	// be unknown. Instead, we return that the value is computed so
	// that the graph can continue to refresh other nodes. It doesn't
	// matter because the config isn't interpolated anyways.
	//
	// For a Destroy, we're also fine with computed values, since our goal is
	// only to get destroy nodes for existing resources.
	//
	// For an input walk, computed values are okay to return because we're only
	// looking for missing variables to prompt the user for.
	if i.Operation == walkRefresh || i.Operation == walkPlanDestroy || i.Operation == walkInput {
		return &unknownVariable, nil
	}

	return nil, fmt.Errorf(
		"Resource '%s' does not have attribute '%s' "+
			"for variable '%s'",
		id,
		v.Field,
		v.FullKey())
}

func (i *Interpolater) computeResourceMultiVariable(
	scope *InterpolationScope,
	v *config.ResourceVariable) (*ast.Variable, error) {
	i.StateLock.RLock()
	defer i.StateLock.RUnlock()

	unknownVariable := unknownVariable()

	// If we're only looking for input, we don't need to expand a
	// multi-variable. This prevents us from encountering things that should be
	// known but aren't because the state has yet to be refreshed.
	if i.Operation == walkInput {
		return &unknownVariable, nil
	}

	// Get the information about this resource variable, and verify
	// that it exists and such.
	module, cr, err := i.resourceVariableInfo(scope, v)
	if err != nil {
		return nil, err
	}

	// Get the keys for all the resources that are created for this resource
	countMax, err := i.resourceCountMax(module, cr, v)
	if err != nil {
		return nil, err
	}

	// If count is zero, we return an empty list
	if countMax == 0 {
		return &ast.Variable{Type: ast.TypeList, Value: []ast.Variable{}}, nil
	}

	// If we have no module in the state yet or count, return unknown
	if module == nil || len(module.Resources) == 0 {
		return &unknownVariable, nil
	}

	var values []interface{}
	for idx := 0; idx < countMax; idx++ {
		id := fmt.Sprintf("%s.%d", v.ResourceId(), idx)

		// ID doesn't have a trailing index. We try both here, but if a value
		// without a trailing index is found we prefer that. This choice
		// is for legacy reasons: older versions of TF preferred it.
		if id == v.ResourceId()+".0" {
			potential := v.ResourceId()
			if _, ok := module.Resources[potential]; ok {
				id = potential
			}
		}

		r, ok := module.Resources[id]
		if !ok {
			continue
		}

		if r.Primary == nil {
			continue
		}

		if singleAttr, ok := r.Primary.Attributes[v.Field]; ok {
			values = append(values, singleAttr)
			continue
		}

		if v.Field == "id" && r.Primary.ID != "" {
			log.Printf("[WARN] resource %s missing 'id' attribute", v.ResourceId())
			values = append(values, r.Primary.ID)
		}

		// computed list or map attribute
		_, isList := r.Primary.Attributes[v.Field+".#"]
		_, isMap := r.Primary.Attributes[v.Field+".%"]
		if !(isList || isMap) {
			continue
		}
		multiAttr, err := i.interpolateComplexTypeAttribute(v.Field, r.Primary.Attributes)
		if err != nil {
			return nil, err
		}

		values = append(values, multiAttr)
	}

	if len(values) == 0 {
		// If the operation is refresh, it isn't an error for a value to
		// be unknown. Instead, we return that the value is computed so
		// that the graph can continue to refresh other nodes. It doesn't
		// matter because the config isn't interpolated anyways.
		//
		// For a Destroy, we're also fine with computed values, since our goal is
		// only to get destroy nodes for existing resources.
		//
		// For an input walk, computed values are okay to return because we're only
		// looking for missing variables to prompt the user for.
		if i.Operation == walkRefresh || i.Operation == walkPlanDestroy || i.Operation == walkDestroy || i.Operation == walkInput {
			return &unknownVariable, nil
		}

		return nil, fmt.Errorf(
			"Resource '%s' does not have attribute '%s' "+
				"for variable '%s'",
			v.ResourceId(),
			v.Field,
			v.FullKey())
	}

	variable, err := hil.InterfaceToVariable(values)
	return &variable, err
}

func (i *Interpolater) interpolateComplexTypeAttribute(
	resourceID string,
	attributes map[string]string) (ast.Variable, error) {
	// We can now distinguish between lists and maps in state by the count field:
	//    - lists (and by extension, sets) use the traditional .# notation
	//    - maps use the newer .% notation
	// Consequently here we can decide how to deal with the keys appropriately
	// based on whether the type is a map of list.
	if lengthAttr, isList := attributes[resourceID+".#"]; isList {
		log.Printf("[DEBUG] Interpolating computed list element attribute %s (%s)",
			resourceID, lengthAttr)

		// In Terraform's internal dotted representation of list-like attributes, the
		// ".#" count field is marked as unknown to indicate "this whole list is
		// unknown". We must honor that meaning here so computed references can be
		// treated properly during the plan phase.
		if lengthAttr == config.UnknownVariableValue {
			return unknownVariable(), nil
		}

		expanded := flatmap.Expand(attributes, resourceID)
		return hil.InterfaceToVariable(expanded)
	}

	if lengthAttr, isMap := attributes[resourceID+".%"]; isMap {
		log.Printf("[DEBUG] Interpolating computed map element attribute %s (%s)",
			resourceID, lengthAttr)

		// In Terraform's internal dotted representation of map attributes, the
		// ".%" count field is marked as unknown to indicate "this whole list is
		// unknown". We must honor that meaning here so computed references can be
		// treated properly during the plan phase.
		if lengthAttr == config.UnknownVariableValue {
			return unknownVariable(), nil
		}

		expanded := flatmap.Expand(attributes, resourceID)
		return hil.InterfaceToVariable(expanded)
	}

	return ast.Variable{}, fmt.Errorf("No complex type %s found", resourceID)
}

func (i *Interpolater) resourceVariableInfo(
	scope *InterpolationScope,
	v *config.ResourceVariable) (*ModuleState, *config.Resource, error) {
	// Get the module tree that contains our current path. This is
	// either the current module (path is empty) or a child.
	modTree := i.Module
	if len(scope.Path) > 1 {
		modTree = i.Module.Child(scope.Path[1:])
	}

	// Get the resource from the configuration so we can verify
	// that the resource is in the configuration and so we can access
	// the configuration if we need to.
	var cr *config.Resource
	for _, r := range modTree.Config().Resources {
		if r.Id() == v.ResourceId() {
			cr = r
			break
		}
	}

	// Get the relevant module
	module := i.State.ModuleByPath(normalizeModulePath(scope.Path))
	return module, cr, nil
}

func (i *Interpolater) resourceCountMax(
	ms *ModuleState,
	cr *config.Resource,
	v *config.ResourceVariable) (int, error) {
	id := v.ResourceId()

	// If we're NOT applying, then we assume we can read the count
	// from the state. Plan and so on may not have any state yet so
	// we do a full interpolation.
	if i.Operation != walkApply {
		if cr == nil {
			return 0, nil
		}

		count, err := cr.Count()
		if err != nil {
			return 0, err
		}

		return count, nil
	}

	// If we have no module state in the apply walk, that suggests we've hit
	// a rather awkward edge-case: the resource this variable refers to
	// has count = 0 and is the only resource processed so far on this walk,
	// and so we've ended up not creating any resource states yet. We don't
	// create a module state until the first resource is written into it,
	// so the module state doesn't exist when we get here.
	//
	// In this case we act as we would if we had been passed a module
	// with an empty resource state map.
	if ms == nil {
		return 0, nil
	}

	// We need to determine the list of resource keys to get values from.
	// This needs to be sorted so the order is deterministic. We used to
	// use "cr.Count()" but that doesn't work if the count is interpolated
	// and we can't guarantee that so we instead depend on the state.
	max := -1
	for k, _ := range ms.Resources {
		// Get the index number for this resource
		index := ""
		if k == id {
			// If the key is the id, then its just 0 (no explicit index)
			index = "0"
		} else if strings.HasPrefix(k, id+".") {
			// Grab the index number out of the state
			index = k[len(id+"."):]
			if idx := strings.IndexRune(index, '.'); idx >= 0 {
				index = index[:idx]
			}
		}

		// If there was no index then this resource didn't match
		// the one we're looking for, exit.
		if index == "" {
			continue
		}

		// Turn the index into an int
		raw, err := strconv.ParseInt(index, 0, 0)
		if err != nil {
			return 0, fmt.Errorf(
				"%s: error parsing index %q as int: %s",
				id, index, err)
		}

		// Keep track of this index if its the max
		if new := int(raw); new > max {
			max = new
		}
	}

	// If we never found any matching resources in the state, we
	// have zero.
	if max == -1 {
		return 0, nil
	}

	// The result value is "max+1" because we're returning the
	// max COUNT, not the max INDEX, and we zero-index.
	return max + 1, nil
}
