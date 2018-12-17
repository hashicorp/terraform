package jsonplan

import (
	"encoding/json"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// stateValues is the common representation of resolved values for both the
// prior state (which is always complete) and the planned new state.
type stateValues struct {
	Outputs    map[string]output `json:"outputs,omitempty"`
	RootModule module            `json:"root_module,omitempty"`
}

// marshalPlannedValues takes the current state, planned changes, and schemas
// and populates the PlannedValues. Any unknown values will be omitted.
func (p *plan) marshalPlannedValues(
	changes *plans.Changes,
	s *states.State,
	schemas *terraform.Schemas,
) error {
	// marshal the current state into a stateValues
	_, err := marshalState(s, schemas)
	if err != nil {
		return err
	}

	// marshal the planned changes into a statesValues
	planned, err := marshalPlan(changes, schemas)
	if err != nil {
		return err
	}

	// TODO: smoosh them together

	// marshalPlannedOutputs
	outputs, err := marshalPlannedOutputs(changes, s)
	if err != nil {
		return err
	}
	p.PlannedValues.Outputs = outputs
	p.PlannedValues.RootModule = planned

	return nil
}

// attributeValues is the JSON representation of the attribute values of the
// resource, whose structure depends on the resource type schema.
type attributeValues map[string]interface{}

func marshalAttributeValues(value cty.Value, schema *configschema.Block) attributeValues {
	ret := make(attributeValues)

	it := value.ElementIterator()
	for it.Next() {
		k, v := it.Element()
		ret[k.AsString()] = v
	}
	return ret
}

func marshalPlannedOutputs(changes *plans.Changes, s *states.State) (map[string]output, error) {
	ret := make(map[string]output)

	// add the current state's outputs to the map
	if !s.Empty() {
		for k, v := range s.RootModule().OutputValues {
			if v.Value != cty.NilVal {
				outputVal, _ := ctyjson.Marshal(v.Value, v.Value.Type())
				ret[k] = output{
					Value:     outputVal,
					Sensitive: v.Sensitive,
				}
			}
		}
	}

	if changes.Outputs == nil {
		// No changes - we're done here!
		return ret, nil
	}

	// overwrite the current state's outputs with any changes
	// this will also add any outputs not in the state
	for _, oc := range changes.Outputs {
		if oc.ChangeSrc.Action == plans.Delete {
			delete(ret, oc.Addr.String())
		}

		var after []byte
		changeV, err := oc.Decode()
		if err != nil {
			return ret, err
		}

		if changeV.After != cty.NilVal {
			if changeV.After.IsWhollyKnown() {
				after, err = ctyjson.Marshal(changeV.After, changeV.After.Type())
				if err != nil {
					return ret, err
				}
			}
		}

		ret[oc.Addr.OutputValue.Name] = output{
			Value:     json.RawMessage(after),
			Sensitive: oc.Sensitive,
		}
	}

	return ret, nil

}

func marshalState(s *states.State, schemas *terraform.Schemas) (module, error) {
	var ret module
	if s.Empty() {
		return ret, nil
	}

	// start with the root module
	ret.Address = s.RootModule().Addr.String()
	rs, err := marshalStateResources(s.RootModule().Resources, schemas)
	if err != nil {
		return ret, err
	}
	ret.Resources = rs

	// build a map of module -> [child module addresses]
	moduleMap := make(map[string][]addrs.ModuleInstance)
	for _, mod := range s.Modules {
		if mod.Addr.IsRoot() {
			continue
		} else {
			parent := mod.Addr.Parent().String()
			moduleMap[parent] = append(moduleMap[parent], mod.Addr)
		}
	}

	// use the state and module map to build up the module structure
	modules, err := marshalStateModules(s, schemas, moduleMap[""], moduleMap)
	if err != nil {
		return ret, err
	}

	ret.ChildModules = modules

	return ret, nil
}

func marshalStateResources(resources map[string]*states.Resource, schemas *terraform.Schemas) ([]resource, error) {
	var ret []resource

	for _, r := range resources {
		for k, ri := range r.Instances {

			resource := resource{
				Address:      r.Addr.String(),
				Type:         r.Addr.Type,
				Name:         r.Addr.Name,
				ProviderName: r.ProviderConfig.ProviderConfig.String(),
			}

			switch r.Addr.Mode {
			case addrs.ManagedResourceMode:
				resource.Mode = "managed"
			case addrs.DataResourceMode:
				resource.Mode = "data"
			default:
				return ret, fmt.Errorf("resource %s has an unsupported mode %s",
					r.Addr.String(),
					r.Addr.Mode.String(),
				)
			}

			if r.EachMode != states.NoEach {
				resource.Index = k
			}

			schema, _ := schemas.ResourceTypeConfig(
				r.ProviderConfig.ProviderConfig.StringCompact(),
				r.Addr.Mode,
				r.Addr.Type,
			)
			resource.SchemaVersion = ri.Current.SchemaVersion

			if schema == nil {
				return nil, fmt.Errorf("no schema found for %s", r.Addr.String())
			}

			riObj, err := ri.Current.Decode(schema.ImpliedType())
			if err != nil {
				return nil, err
			}

			resource.AttributeValues = marshalAttributeValues(riObj.Value, schema)

			ret = append(ret, resource)
		}

	}

	return ret, nil
}

// marshalStateModules is an ungainly recursive function to build a module
// structure out of a teraform state.
func marshalStateModules(
	s *states.State,
	schemas *terraform.Schemas,
	modules []addrs.ModuleInstance,
	moduleMap map[string][]addrs.ModuleInstance,
) ([]module, error) {

	var ret []module

	for _, child := range modules {
		stateMod := s.Module(child)
		// cm for child module, naming things is hard.
		cm := module{Address: stateMod.Addr.String()}
		rs, err := marshalStateResources(stateMod.Resources, schemas)
		if err != nil {
			return nil, err
		}
		cm.Resources = rs
		if moduleMap[child.String()] != nil {
			moreChildModules, err := marshalStateModules(s, schemas, moduleMap[child.String()], moduleMap)
			if err != nil {
				return nil, err
			}
			cm.ChildModules = moreChildModules
		}

		ret = append(ret, cm)
	}

	return ret, nil
}

func marshalPlan(changes *plans.Changes, schemas *terraform.Schemas) (module, error) {
	var ret module
	if changes.Empty() {
		return ret, nil
	}

	// build two maps:
	// 		module name -> [resource addresses]
	// 		module -> [children modules]
	moduleResourceMap := make(map[string][]addrs.AbsResourceInstance)
	moduleMap := make(map[string][]addrs.ModuleInstance)

	for _, resource := range changes.Resources {
		containingModule := resource.Addr.Module.String()
		moduleResourceMap[containingModule] = append(moduleResourceMap[containingModule], resource.Addr)

		// root has no parents.
		// root is an orphan
		// root is BATMAN
		if containingModule != "" {
			parent := resource.Addr.Module.Parent().String()
			moduleMap[parent] = append(moduleMap[parent], resource.Addr.Module)
		}
	}

	// start with the root module
	rs, err := marshalPlanResources(changes, moduleResourceMap[""], schemas)
	if err != nil {
		return ret, err
	}
	ret.Resources = rs

	childModules, err := marshalPlanModules(changes, schemas, moduleMap[""], moduleMap, moduleResourceMap)
	if err != nil {
		return ret, err
	}
	ret.ChildModules = childModules

	return ret, nil
}

func marshalPlanResources(changes *plans.Changes, ris []addrs.AbsResourceInstance, schemas *terraform.Schemas) ([]resource, error) {
	var ret []resource

	for _, ri := range ris {
		r := changes.ResourceInstance(ri)
		resource := resource{
			Address:      r.Addr.String(),
			Type:         r.Addr.Resource.Resource.Type,
			Name:         r.Addr.Resource.Resource.Name,
			ProviderName: r.ProviderAddr.ProviderConfig.StringCompact(),
			Index:        r.Addr.Resource.Key,
		}

		switch r.Addr.Resource.Resource.Mode {
		case addrs.ManagedResourceMode:
			resource.Mode = "managed"
		case addrs.DataResourceMode:
			resource.Mode = "data"
		default:
			return ret, fmt.Errorf("resource %s has an unsupported mode %s",
				r.Addr.String(),
				r.Addr.Resource.Resource.Mode.String(),
			)
		}

		schema, schemaVer := schemas.ResourceTypeConfig(
			resource.ProviderName,
			r.Addr.Resource.Resource.Mode,
			resource.Type,
		)
		if schema == nil {
			return nil, fmt.Errorf("no schema found for %s", r.Addr.String())
		}
		resource.SchemaVersion = schemaVer
		changeV, err := r.Decode(schema.ImpliedType())
		if err != nil {
			return nil, err
		}

		// TODO:
		// What does this do if the values are unknown?
		// How about deletions?
		if changeV.After != cty.NilVal {
			if changeV.After.IsWhollyKnown() {
				resource.AttributeValues = marshalAttributeValues(changeV.After, schema)
			}
		}

		ret = append(ret, resource)
	}

	return ret, nil
}

// haha, and you thought marshalStateModules was ungainly!
func marshalPlanModules(
	changes *plans.Changes,
	schemas *terraform.Schemas,
	childModules []addrs.ModuleInstance,
	moduleMap map[string][]addrs.ModuleInstance,
	moduleResourceMap map[string][]addrs.AbsResourceInstance,
) ([]module, error) {

	var ret []module

	for _, child := range childModules {
		moduleResources := moduleResourceMap[child.String()]
		// cm for child module, naming things is hard.
		cm := module{Address: child.String()}
		rs, err := marshalPlanResources(changes, moduleResources, schemas)
		if err != nil {
			return nil, err
		}
		cm.Resources = rs

		if len(moduleMap[child.String()]) > 0 {
			moreChildModules, err := marshalPlanModules(changes, schemas, moduleMap[child.String()], moduleMap, moduleResourceMap)
			if err != nil {
				return nil, err
			}
			cm.ChildModules = moreChildModules
		}

		ret = append(ret, cm)
	}

	return ret, nil
}
