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
	curr, err := marshalState(s, schemas)
	if err != nil {
		return err
	}

	// TODO: marshal the changes into a statesValues
	// TODO: smoosh them together

	// marshalPlannedOutputs
	outputs, err := marshalPlannedOutputs(changes, s)
	if err != nil {
		return err
	}
	p.PlannedValues.Outputs = outputs
	p.PlannedValues.RootModule = curr

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
	var rs []resource

	for _, r := range resources {
		for k, ri := range r.Instances {

			ret := resource{
				Address:      r.Addr.String(),
				Type:         r.Addr.Type,
				Name:         r.Addr.Name,
				ProviderName: r.ProviderConfig.ProviderConfig.String(),
			}

			switch r.Addr.Mode {
			case addrs.ManagedResourceMode:
				ret.Mode = "managed"
			case addrs.DataResourceMode:
				ret.Mode = "data"
			default:
				return rs, fmt.Errorf("resource %s has an unsupported mode %s",
					r.Addr.String(),
					r.Addr.Mode.String(),
				)
			}

			if r.EachMode != states.NoEach {
				ret.Index = k
			}

			schema, _ := schemas.ResourceTypeConfig(
				r.ProviderConfig.ProviderConfig.StringCompact(),
				r.Addr.Mode,
				r.Addr.Type,
			)
			ret.SchemaVersion = ri.Current.SchemaVersion

			if schema == nil {
				return nil, fmt.Errorf("no schema found for %s", r.Addr.String())
			}

			riObj, err := ri.Current.Decode(schema.ImpliedType())
			if err != nil {
				fmt.Println("error in decode")
				return nil, err
			}

			ret.AttributeValues = marshalAttributeValues(riObj.Value, schema)

			rs = append(rs, ret)
		}

	}

	return rs, nil
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
