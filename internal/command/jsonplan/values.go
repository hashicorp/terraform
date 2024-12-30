// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonplan

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
)

// stateValues is the common representation of resolved values for both the
// prior state (which is always complete) and the planned new state.
type stateValues struct {
	Outputs    map[string]output `json:"outputs,omitempty"`
	RootModule module            `json:"root_module,omitempty"`
}

// AttributeValues is the JSON representation of the attribute values of the
// resource, whose structure depends on the resource type schema.
type attributeValues map[string]interface{}

func marshalAttributeValues(value cty.Value, schema *configschema.Block) attributeValues {
	if value == cty.NilVal || value.IsNull() {
		return nil
	}
	ret := make(attributeValues)

	it := value.ElementIterator()
	for it.Next() {
		k, v := it.Element()
		vJSON, _ := ctyjson.Marshal(v, v.Type())
		ret[k.AsString()] = json.RawMessage(vJSON)
	}
	return ret
}

// marshalPlannedOutputs takes a list of changes and returns a map of output
// values
func marshalPlannedOutputs(changes *plans.ChangesSrc) (map[string]output, error) {
	if changes.Outputs == nil {
		// No changes - we're done here!
		return nil, nil
	}

	ret := make(map[string]output)

	for _, oc := range changes.Outputs {
		if oc.ChangeSrc.Action == plans.Delete {
			continue
		}

		var after, afterType []byte
		changeV, err := oc.Decode()
		if err != nil {
			return ret, err
		}
		// The values may be marked, but we must rely on the Sensitive flag
		// as the decoded value is only an intermediate step in transcoding
		// this to a json format.
		changeV.After, _ = changeV.After.UnmarkDeep()

		if changeV.After != cty.NilVal && changeV.After.IsWhollyKnown() {
			ty := changeV.After.Type()
			after, err = ctyjson.Marshal(changeV.After, ty)
			if err != nil {
				return ret, err
			}
			afterType, err = ctyjson.MarshalType(ty)
			if err != nil {
				return ret, err
			}
		}

		ret[oc.Addr.OutputValue.Name] = output{
			Value:     json.RawMessage(after),
			Type:      json.RawMessage(afterType),
			Sensitive: oc.Sensitive,
		}
	}

	return ret, nil

}

func marshalPlannedValues(changes *plans.ChangesSrc, schemas *terraform.Schemas) (module, error) {
	var ret module

	// build two maps:
	// 		module name -> [resource addresses]
	// 		module -> [children modules]
	moduleResourceMap := make(map[string][]addrs.AbsResourceInstance)
	moduleMap := make(map[string][]addrs.ModuleInstance)
	seenModules := make(map[string]bool)

	for _, resource := range changes.Resources {
		// If the resource is being deleted, skip over it.
		// Deposed instances are always conceptually a destroy, but if they
		// were gone during refresh then the change becomes a noop.
		if resource.Action != plans.Delete && resource.DeposedKey == states.NotDeposed {
			containingModule := resource.Addr.Module.String()
			moduleResourceMap[containingModule] = append(moduleResourceMap[containingModule], resource.Addr)

			// the root module has no parents
			if !resource.Addr.Module.IsRoot() {
				parent := resource.Addr.Module.Parent().String()
				// we expect to see multiple resources in one module, so we
				// only need to report the "parent" module for each child module
				// once.
				if !seenModules[containingModule] {
					moduleMap[parent] = append(moduleMap[parent], resource.Addr.Module)
					seenModules[containingModule] = true
				}

				// If any given parent module has no resources, it needs to be
				// added to the moduleMap. This walks through the current
				// resources' modules' ancestors, taking advantage of the fact
				// that Ancestors() returns an ordered slice, and verifies that
				// each one is in the map.
				ancestors := resource.Addr.Module.Ancestors()
				for i, ancestor := range ancestors[:len(ancestors)-1] {
					aStr := ancestor.String()

					// childStr here is the immediate child of the current step
					childStr := ancestors[i+1].String()
					// we likely will see multiple resources in one module, so we
					// only need to report the "parent" module for each child module
					// once.
					if !seenModules[childStr] {
						moduleMap[aStr] = append(moduleMap[aStr], ancestors[i+1])
						seenModules[childStr] = true
					}
				}
			}
		}
	}

	// start with the root module
	resources, err := marshalPlanResources(changes, moduleResourceMap[""], schemas)
	if err != nil {
		return ret, err
	}
	ret.Resources = resources

	childModules, err := marshalPlanModules(changes, schemas, moduleMap[""], moduleMap, moduleResourceMap)
	if err != nil {
		return ret, err
	}
	sort.Slice(childModules, func(i, j int) bool {
		return childModules[i].Address < childModules[j].Address
	})

	ret.ChildModules = childModules

	return ret, nil
}

// marshalPlanResources
func marshalPlanResources(changes *plans.ChangesSrc, ris []addrs.AbsResourceInstance, schemas *terraform.Schemas) ([]resource, error) {
	var ret []resource

	for _, ri := range ris {
		r := changes.ResourceInstance(ri)
		if r.Action == plans.Delete {
			continue
		}

		resource := resource{
			Address:      r.Addr.String(),
			Type:         r.Addr.Resource.Resource.Type,
			Name:         r.Addr.Resource.Resource.Name,
			ProviderName: r.ProviderAddr.Provider.String(),
			Index:        r.Addr.Resource.Key,
		}

		switch r.Addr.Resource.Resource.Mode {
		case addrs.ManagedResourceMode:
			resource.Mode = "managed"
		case addrs.DataResourceMode:
			resource.Mode = "data"
		default:
			return nil, fmt.Errorf("resource %s has an unsupported mode %s",
				r.Addr.String(),
				r.Addr.Resource.Resource.Mode.String(),
			)
		}

		schema, schemaVer := schemas.ResourceTypeConfig(
			r.ProviderAddr.Provider,
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

		// copy the marked After values so we can use these in marshalSensitiveValues
		markedAfter := changeV.After

		// The values may be marked, but we must rely on the Sensitive flag
		// as the decoded value is only an intermediate step in transcoding
		// this to a json format.
		changeV.Before, _ = changeV.Before.UnmarkDeep()
		changeV.After, _ = changeV.After.UnmarkDeep()

		if changeV.After != cty.NilVal {
			if changeV.After.IsWhollyKnown() {
				resource.AttributeValues = marshalAttributeValues(changeV.After, schema)
			} else {
				knowns := omitUnknowns(changeV.After)
				resource.AttributeValues = marshalAttributeValues(knowns, schema)
			}
		}

		s := jsonstate.SensitiveAsBool(markedAfter)
		v, err := ctyjson.Marshal(s, s.Type())
		if err != nil {
			return nil, err
		}
		resource.SensitiveValues = v

		ret = append(ret, resource)
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Address < ret[j].Address
	})

	return ret, nil
}

// marshalPlanModules iterates over a list of modules to recursively describe
// the full module tree.
func marshalPlanModules(
	changes *plans.ChangesSrc,
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
