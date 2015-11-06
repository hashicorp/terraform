package terraform

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/lang/ast"
	"github.com/hashicorp/terraform/config/module"
)

const (
	// VarEnvPrefix is the prefix of variables that are read from
	// the environment to set variables here.
	VarEnvPrefix = "TF_VAR_"
)

// Interpolater is the structure responsible for determining the values
// for interpolations such as `aws_instance.foo.bar`.
type Interpolater struct {
	Operation walkOperation
	Module    *module.Tree
	State     *State
	StateLock *sync.RWMutex
	Variables map[string]string
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
	result := make(map[string]ast.Variable, len(vars))

	// Copy the default variables
	if i.Module != nil && scope != nil {
		mod := i.Module
		if len(scope.Path) > 1 {
			mod = i.Module.Child(scope.Path[1:])
		}
		for _, v := range mod.Config().Variables {
			for k, val := range v.DefaultsMap() {
				result[k] = ast.Variable{
					Value: val,
					Type:  ast.TypeString,
				}
			}
		}
	}

	for n, rawV := range vars {
		var err error
		switch v := rawV.(type) {
		case *config.CountVariable:
			err = i.valueCountVar(scope, n, v, result)
		case *config.ModuleVariable:
			err = i.valueModuleVar(scope, n, v, result)
		case *config.PathVariable:
			err = i.valuePathVar(scope, n, v, result)
		case *config.ResourceVariable:
			err = i.valueResourceVar(scope, n, v, result)
		case *config.SelfVariable:
			err = i.valueSelfVar(scope, n, v, result)
		case *config.UserVariable:
			err = i.valueUserVar(scope, n, v, result)
		default:
			err = fmt.Errorf("%s: unknown variable type: %T", n, rawV)
		}

		if err != nil {
			return nil, err
		}
	}

	return result, nil
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

func (i *Interpolater) valueModuleVar(
	scope *InterpolationScope,
	n string,
	v *config.ModuleVariable,
	result map[string]ast.Variable) error {
	// If we're computing all dynamic fields, then module vars count
	// and we mark it as computed.
	if i.Operation == walkValidate {
		result[n] = ast.Variable{
			Value: config.UnknownVariableValue,
			Type:  ast.TypeString,
		}
		return nil
	}

	// Build the path to the child module we want
	path := make([]string, len(scope.Path), len(scope.Path)+1)
	copy(path, scope.Path)
	path = append(path, v.Name)

	// Grab the lock so that if other interpolations are running or
	// state is being modified, we'll be safe.
	i.StateLock.RLock()
	defer i.StateLock.RUnlock()

	// Get the module where we're looking for the value
	var value string
	mod := i.State.ModuleByPath(path)
	if mod == nil {
		// If the module doesn't exist, then we can return an empty string.
		// This happens usually only in Refresh() when we haven't populated
		// a state. During validation, we semantically verify that all
		// modules reference other modules, and graph ordering should
		// ensure that the module is in the state, so if we reach this
		// point otherwise it really is a panic.
		value = config.UnknownVariableValue
	} else {
		// Get the value from the outputs
		var ok bool
		value, ok = mod.Outputs[v.Field]
		if !ok {
			// Same reasons as the comment above.
			value = config.UnknownVariableValue
		}
	}

	result[n] = ast.Variable{
		Value: value,
		Type:  ast.TypeString,
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
		result[n] = ast.Variable{
			Value: config.UnknownVariableValue,
			Type:  ast.TypeString,
		}
		return nil
	}

	var attr string
	var err error
	if v.Multi && v.Index == -1 {
		attr, err = i.computeResourceMultiVariable(scope, v)
	} else {
		attr, err = i.computeResourceVariable(scope, v)
	}
	if err != nil {
		return err
	}

	result[n] = ast.Variable{
		Value: attr,
		Type:  ast.TypeString,
	}
	return nil
}

func (i *Interpolater) valueSelfVar(
	scope *InterpolationScope,
	n string,
	v *config.SelfVariable,
	result map[string]ast.Variable) error {
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

func (i *Interpolater) valueUserVar(
	scope *InterpolationScope,
	n string,
	v *config.UserVariable,
	result map[string]ast.Variable) error {
	val, ok := i.Variables[v.Name]
	if ok {
		result[n] = ast.Variable{
			Value: val,
			Type:  ast.TypeString,
		}
		return nil
	}

	if _, ok := result[n]; !ok && i.Operation == walkValidate {
		result[n] = ast.Variable{
			Value: config.UnknownVariableValue,
			Type:  ast.TypeString,
		}
		return nil
	}

	// Look up if we have any variables with this prefix because
	// those are map overrides. Include those.
	for k, val := range i.Variables {
		if strings.HasPrefix(k, v.Name+".") {
			result["var."+k] = ast.Variable{
				Value: val,
				Type:  ast.TypeString,
			}
		}
	}

	return nil
}

func (i *Interpolater) computeResourceVariable(
	scope *InterpolationScope,
	v *config.ResourceVariable) (string, error) {
	id := v.ResourceId()
	if v.Multi {
		id = fmt.Sprintf("%s.%d", id, v.Index)
	}

	i.StateLock.RLock()
	defer i.StateLock.RUnlock()

	// Get the information about this resource variable, and verify
	// that it exists and such.
	module, _, err := i.resourceVariableInfo(scope, v)
	if err != nil {
		return "", err
	}

	// If we have no module in the state yet or count, return empty
	if module == nil || len(module.Resources) == 0 {
		return "", nil
	}

	// Get the resource out from the state. We know the state exists
	// at this point and if there is a state, we expect there to be a
	// resource with the given name.
	r, ok := module.Resources[id]
	if !ok && v.Multi && v.Index == 0 {
		r, ok = module.Resources[v.ResourceId()]
	}
	if !ok {
		r = nil
	}
	if r == nil {
		goto MISSING
	}

	if r.Primary == nil {
		goto MISSING
	}

	if attr, ok := r.Primary.Attributes[v.Field]; ok {
		return attr, nil
	}

	// computed list attribute
	if _, ok := r.Primary.Attributes[v.Field+".#"]; ok {
		return i.interpolateListAttribute(v.Field, r.Primary.Attributes)
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
				return attr, nil
			}

			// Maps make this
			key = fmt.Sprintf("%s", strings.Join(parts[:i], "."))
			if attr, ok := r.Primary.Attributes[key]; ok {
				return attr, nil
			}
		}
	}

MISSING:
	// Validation for missing interpolations should happen at a higher
	// semantic level. If we reached this point and don't have variables,
	// just return the computed value.
	if scope == nil && scope.Resource == nil {
		return config.UnknownVariableValue, nil
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
	if i.Operation == walkRefresh || i.Operation == walkPlanDestroy || i.Operation == walkDestroy || i.Operation == walkInput {
		return config.UnknownVariableValue, nil
	}

	return "", fmt.Errorf(
		"Resource '%s' does not have attribute '%s' "+
			"for variable '%s'",
		id,
		v.Field,
		v.FullKey())
}

func (i *Interpolater) computeResourceMultiVariable(
	scope *InterpolationScope,
	v *config.ResourceVariable) (string, error) {
	i.StateLock.RLock()
	defer i.StateLock.RUnlock()

	// Get the information about this resource variable, and verify
	// that it exists and such.
	module, cr, err := i.resourceVariableInfo(scope, v)
	if err != nil {
		return "", err
	}

	// Get the count so we know how many to iterate over
	count, err := cr.Count()
	if err != nil {
		return "", fmt.Errorf(
			"Error reading %s count: %s",
			v.ResourceId(),
			err)
	}

	// If we have no module in the state yet or count, return empty
	if module == nil || len(module.Resources) == 0 || count == 0 {
		return "", nil
	}

	var values []string
	for j := 0; j < count; j++ {
		id := fmt.Sprintf("%s.%d", v.ResourceId(), j)

		// If we're dealing with only a single resource, then the
		// ID doesn't have a trailing index.
		if count == 1 {
			id = v.ResourceId()
		}

		r, ok := module.Resources[id]
		if !ok {
			continue
		}

		if r.Primary == nil {
			continue
		}

		attr, ok := r.Primary.Attributes[v.Field]
		if !ok {
			// computed list attribute
			_, ok := r.Primary.Attributes[v.Field+".#"]
			if !ok {
				continue
			}
			attr, err = i.interpolateListAttribute(v.Field, r.Primary.Attributes)
			if err != nil {
				return "", err
			}
		}

		if config.IsStringList(attr) {
			for _, s := range config.StringList(attr).Slice() {
				values = append(values, s)
			}
			continue
		}

		// If any value is unknown, the whole thing is unknown
		if attr == config.UnknownVariableValue {
			return config.UnknownVariableValue, nil
		}

		values = append(values, attr)
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
			return config.UnknownVariableValue, nil
		}

		return "", fmt.Errorf(
			"Resource '%s' does not have attribute '%s' "+
				"for variable '%s'",
			v.ResourceId(),
			v.Field,
			v.FullKey())
	}

	return config.NewStringList(values).String(), nil
}

func (i *Interpolater) interpolateListAttribute(
	resourceID string,
	attributes map[string]string) (string, error) {

	attr := attributes[resourceID+".#"]
	log.Printf("[DEBUG] Interpolating computed list attribute %s (%s)",
		resourceID, attr)

	var members []string
	numberedListMember := regexp.MustCompile("^" + resourceID + "\\.[0-9]+$")
	for id, value := range attributes {
		if numberedListMember.MatchString(id) {
			members = append(members, value)
		}
	}

	sort.Strings(members)
	return config.NewStringList(members).String(), nil
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
	if cr == nil {
		return nil, nil, fmt.Errorf(
			"Resource '%s' not found for variable '%s'",
			v.ResourceId(),
			v.FullKey())
	}

	// Get the relevant module
	module := i.State.ModuleByPath(scope.Path)
	return module, cr, nil
}
