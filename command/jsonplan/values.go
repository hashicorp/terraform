package jsonplan

import (
	"encoding/json"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/terraform"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// stateValues is the common representation of resolved values for both the
// prior state (which is always complete) and the planned new state.
type stateValues struct {
	Outputs    map[string]output `json:"outputs,omitempty"`
	RootModule module            `json:"root_module,omitempty"`
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

// marshalAttributeValuesBool returns an attributeValues structure with "true" and
// "false" in place of the values indicating whether the value is known or not.
func marshalAttributeValuesBool(value cty.Value, schema *configschema.Block) attributeValues {
	ret := make(attributeValues)

	it := value.ElementIterator()
	for it.Next() {
		k, v := it.Element()
		if v.IsWhollyKnown() {
			ret[k.AsString()] = "true"
		}
		ret[k.AsString()] = "false"
	}
	return ret
}

// marshalPlannedOutputs takes a list of changes and returns two output maps,
// the former with output values and the latter with true/false in place of
// values indicating whether the values are known at plan time.
func marshalPlannedOutputs(changes *plans.Changes) (map[string]output, map[string]output, error) {
	if changes.Outputs == nil {
		// No changes - we're done here!
		return nil, nil, nil
	}

	ret := make(map[string]output)
	uRet := make(map[string]output)

	for _, oc := range changes.Outputs {
		if oc.ChangeSrc.Action == plans.Delete {
			continue
		}

		var after []byte
		changeV, err := oc.Decode()
		if err != nil {
			return ret, uRet, err
		}

		if changeV.After != cty.NilVal {
			if changeV.After.IsWhollyKnown() {
				after, err = ctyjson.Marshal(changeV.After, changeV.After.Type())
				if err != nil {
					return ret, uRet, err
				}
				uRet[oc.Addr.OutputValue.Name] = output{
					Value:     json.RawMessage("true"),
					Sensitive: oc.Sensitive,
				}
			} else {
				uRet[oc.Addr.OutputValue.Name] = output{
					Value:     json.RawMessage("false"),
					Sensitive: oc.Sensitive,
				}
			}
		}

		ret[oc.Addr.OutputValue.Name] = output{
			Value:     json.RawMessage(after),
			Sensitive: oc.Sensitive,
		}
	}

	return ret, uRet, nil

}

// marshalPlannedValues returns two modules:
// The former has attribute values populated and the latter has true/false in
// place of values indicating whether the values are known at plan time.
func marshalPlannedValues(changes *plans.Changes, schemas *terraform.Schemas) (module, module, error) {
	var ret, uRet module
	if changes.Empty() {
		return ret, uRet, nil
	}

	// build two maps:
	// 		module name -> [resource addresses]
	// 		module -> [children modules]
	moduleResourceMap := make(map[string][]addrs.AbsResourceInstance)
	moduleMap := make(map[string][]addrs.ModuleInstance)

	for _, resource := range changes.Resources {
		// if the resource is being deleted, skip over it.
		if resource.Action != plans.Delete {
			containingModule := resource.Addr.Module.String()
			moduleResourceMap[containingModule] = append(moduleResourceMap[containingModule], resource.Addr)

			// root has no parents.
			if containingModule != "" {
				parent := resource.Addr.Module.Parent().String()
				moduleMap[parent] = append(moduleMap[parent], resource.Addr.Module)
			}
		}
	}

	// start with the root module
	resources, uResources, err := marshalPlanResources(changes, moduleResourceMap[""], schemas)
	if err != nil {
		return ret, uRet, err
	}
	ret.Resources = resources
	uRet.Resources = uResources

	childModules, err := marshalPlanModules(changes, schemas, moduleMap[""], moduleMap, moduleResourceMap)
	if err != nil {
		return ret, uRet, err
	}
	ret.ChildModules = childModules

	return ret, uRet, nil
}

// marshalPlannedValues returns two resource slices:
// The former has attribute values populated and the latter has true/false in
// place of values indicating whether the values are known at plan time.
func marshalPlanResources(changes *plans.Changes, ris []addrs.AbsResourceInstance, schemas *terraform.Schemas) ([]resource, []resource, error) {
	var ret, uRet []resource

	for _, ri := range ris {
		r := changes.ResourceInstance(ri)
		if r.Action == plans.Delete || r.Action == plans.NoOp {
			continue
		}

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
			return nil, nil, fmt.Errorf("resource %s has an unsupported mode %s",
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
			return nil, nil, fmt.Errorf("no schema found for %s", r.Addr.String())
		}
		resource.SchemaVersion = schemaVer
		changeV, err := r.Decode(schema.ImpliedType())
		if err != nil {
			return nil, nil, err
		}

		var unknownAttributeValues attributeValues
		if changeV.After != cty.NilVal {
			if changeV.After.IsWhollyKnown() {
				resource.AttributeValues = marshalAttributeValues(changeV.After, schema)
			}
			unknownAttributeValues = marshalAttributeValuesBool(changeV.After, schema)
		}

		uResource := resource
		uResource.AttributeValues = unknownAttributeValues

		ret = append(ret, resource)
		uRet = append(uRet, uResource)
	}

	return ret, uRet, nil
}

// marshalPlanModules iterates over a list of modules to recursively describe
// the full module tree.
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
		var cm module
		// don't populate the address for the root module
		if child.String() != "" {
			cm.Address = child.String()
		}
		rs, _, err := marshalPlanResources(changes, moduleResources, schemas)
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
