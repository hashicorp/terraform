package jsonstate

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/terraform"
)

// FormatVersion represents the version of the json format and will be
// incremented for any change to this format that requires changes to a
// consuming parser.
const FormatVersion = "0.1"

// state is the top-level representation of the json format of a terraform
// state.
type state struct {
	FormatVersion    string       `json:"format_version,omitempty"`
	TerraformVersion string       `json:"terraform_version,omitempty"`
	Values           *stateValues `json:"values,omitempty"`
}

// stateValues is the common representation of resolved values for both the prior
// state (which is always complete) and the planned new state.
type stateValues struct {
	Outputs    map[string]output `json:"outputs,omitempty"`
	RootModule module            `json:"root_module,omitempty"`
}

type output struct {
	Sensitive bool            `json:"sensitive"`
	Value     json.RawMessage `json:"value,omitempty"`
}

// module is the representation of a module in state. This can be the root module
// or a child module
type module struct {
	// Resources are sorted in a user-friendly order that is undefined at this
	// time, but consistent.
	Resources []resource `json:"resources,omitempty"`

	// Address is the absolute module address, omitted for the root module
	Address string `json:"address,omitempty"`

	// Each module object can optionally have its own nested "child_modules",
	// recursively describing the full module tree.
	ChildModules []module `json:"child_modules,omitempty"`
}

// Resource is the representation of a resource in the state.
type resource struct {
	// Address is the absolute resource address
	Address string `json:"address,omitempty"`

	// Mode can be "managed" or "data"
	Mode string `json:"mode,omitempty"`

	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`

	// Index is omitted for a resource not using `count` or `for_each`.
	Index addrs.InstanceKey `json:"index,omitempty"`

	// ProviderName allows the property "type" to be interpreted unambiguously
	// in the unusual situation where a provider offers a resource type whose
	// name does not start with its own name, such as the "googlebeta" provider
	// offering "google_compute_instance".
	ProviderName string `json:"provider_name"`

	// SchemaVersion indicates which version of the resource type schema the
	// "values" property conforms to.
	SchemaVersion uint64 `json:"schema_version"`

	// AttributeValues is the JSON representation of the attribute values of the
	// resource, whose structure depends on the resource type schema. Any
	// unknown values are omitted or set to null, making them indistinguishable
	// from absent values.
	AttributeValues attributeValues `json:"values,omitempty"`

	// DependsOn contains a list of the resource's dependencies. The entries are
	// addresses relative to the containing module.
	DependsOn []string `json:"depends_on,omitempty"`

	// Tainted is true if the resource is tainted in terraform state.
	Tainted bool `json:"tainted,omitempty"`

	// Deposed is set if the resource is deposed in terraform state.
	DeposedKey string `json:"deposed_key,omitempty"`
}

// attributeValues is the JSON representation of the attribute values of the
// resource, whose structure depends on the resource type schema.
type attributeValues map[string]interface{}

func marshalAttributeValues(value cty.Value) attributeValues {
	// unmark our value to show all values
	value, _ = value.UnmarkDeep()

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

// newState() returns a minimally-initialized state
func newState() *state {
	return &state{
		FormatVersion: FormatVersion,
	}
}

// Marshal returns the json encoding of a terraform state.
func Marshal(sf *statefile.File, schemas *terraform.Schemas) ([]byte, error) {
	output := newState()

	if sf == nil || sf.State.Empty() {
		ret, err := json.Marshal(output)
		return ret, err
	}

	if sf.TerraformVersion != nil {
		output.TerraformVersion = sf.TerraformVersion.String()
	}

	// output.StateValues
	err := output.marshalStateValues(sf.State, schemas)
	if err != nil {
		return nil, err
	}

	ret, err := json.Marshal(output)
	return ret, err
}

func (jsonstate *state) marshalStateValues(s *states.State, schemas *terraform.Schemas) error {
	var sv stateValues
	var err error

	// only marshal the root module outputs
	sv.Outputs, err = marshalOutputs(s.RootModule().OutputValues)
	if err != nil {
		return err
	}

	// use the state and module map to build up the module structure
	sv.RootModule, err = marshalRootModule(s, schemas)
	if err != nil {
		return err
	}

	jsonstate.Values = &sv
	return nil
}

func marshalOutputs(outputs map[string]*states.OutputValue) (map[string]output, error) {
	if outputs == nil {
		return nil, nil
	}

	ret := make(map[string]output)
	for k, v := range outputs {
		ov, err := ctyjson.Marshal(v.Value, v.Value.Type())
		if err != nil {
			return ret, err
		}
		ret[k] = output{
			Value:     ov,
			Sensitive: v.Sensitive,
		}
	}

	return ret, nil
}

func marshalRootModule(s *states.State, schemas *terraform.Schemas) (module, error) {
	var ret module
	var err error

	ret.Address = ""
	ret.Resources, err = marshalResources(s.RootModule().Resources, addrs.RootModuleInstance, schemas)
	if err != nil {
		return ret, err
	}

	// build a map of module -> set[child module addresses]
	moduleChildSet := make(map[string]map[string]struct{})
	for _, mod := range s.Modules {
		if mod.Addr.IsRoot() {
			continue
		} else {
			for childAddr := mod.Addr; !childAddr.IsRoot(); childAddr = childAddr.Parent() {
				if _, ok := moduleChildSet[childAddr.Parent().String()]; !ok {
					moduleChildSet[childAddr.Parent().String()] = map[string]struct{}{}
				}
				moduleChildSet[childAddr.Parent().String()][childAddr.String()] = struct{}{}
			}
		}
	}

	// transform the previous map into map of module -> [child module addresses]
	moduleMap := make(map[string][]addrs.ModuleInstance)
	for parent, children := range moduleChildSet {
		for child := range children {
			childModuleInstance, diags := addrs.ParseModuleInstanceStr(child)
			if diags.HasErrors() {
				return ret, diags.Err()
			}
			moduleMap[parent] = append(moduleMap[parent], childModuleInstance)
		}
	}

	// use the state and module map to build up the module structure
	ret.ChildModules, err = marshalModules(s, schemas, moduleMap[""], moduleMap)
	return ret, err
}

// marshalModules is an ungainly recursive function to build a module structure
// out of terraform state.
func marshalModules(
	s *states.State,
	schemas *terraform.Schemas,
	modules []addrs.ModuleInstance,
	moduleMap map[string][]addrs.ModuleInstance,
) ([]module, error) {
	var ret []module
	for _, child := range modules {
		// cm for child module, naming things is hard.
		cm := module{Address: child.String()}

		// the module may be resourceless and contain only submodules, it will then be nil here
		stateMod := s.Module(child)
		if stateMod != nil {
			rs, err := marshalResources(stateMod.Resources, stateMod.Addr, schemas)
			if err != nil {
				return nil, err
			}
			cm.Resources = rs
		}

		if moduleMap[child.String()] != nil {
			moreChildModules, err := marshalModules(s, schemas, moduleMap[child.String()], moduleMap)
			if err != nil {
				return nil, err
			}
			cm.ChildModules = moreChildModules
		}

		ret = append(ret, cm)
	}

	// sort the child modules by address for consistency.
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Address < ret[j].Address
	})

	return ret, nil
}

func marshalResources(resources map[string]*states.Resource, module addrs.ModuleInstance, schemas *terraform.Schemas) ([]resource, error) {
	var ret []resource

	for _, r := range resources {
		for k, ri := range r.Instances {

			resAddr := r.Addr.Resource

			current := resource{
				Address:      r.Addr.Instance(k).String(),
				Index:        k,
				Type:         resAddr.Type,
				Name:         resAddr.Name,
				ProviderName: r.ProviderConfig.Provider.String(),
			}

			switch resAddr.Mode {
			case addrs.ManagedResourceMode:
				current.Mode = "managed"
			case addrs.DataResourceMode:
				current.Mode = "data"
			default:
				return ret, fmt.Errorf("resource %s has an unsupported mode %s",
					resAddr.String(),
					resAddr.Mode.String(),
				)
			}

			schema, version := schemas.ResourceTypeConfig(
				r.ProviderConfig.Provider,
				resAddr.Mode,
				resAddr.Type,
			)

			// It is possible that the only instance is deposed
			if ri.Current != nil {
				if version != ri.Current.SchemaVersion {
					return nil, fmt.Errorf("schema version %d for %s in state does not match version %d from the provider", ri.Current.SchemaVersion, resAddr, version)
				}

				current.SchemaVersion = ri.Current.SchemaVersion

				if schema == nil {
					return nil, fmt.Errorf("no schema found for %s (in provider %s)", resAddr.String(), r.ProviderConfig.Provider)
				}
				riObj, err := ri.Current.Decode(schema.ImpliedType())
				if err != nil {
					return nil, err
				}

				current.AttributeValues = marshalAttributeValues(riObj.Value)

				if len(riObj.Dependencies) > 0 {
					dependencies := make([]string, len(riObj.Dependencies))
					for i, v := range riObj.Dependencies {
						dependencies[i] = v.String()
					}
					current.DependsOn = dependencies
				}

				if riObj.Status == states.ObjectTainted {
					current.Tainted = true
				}
				ret = append(ret, current)
			}

			for deposedKey, rios := range ri.Deposed {
				// copy the base fields from the current instance
				deposed := resource{
					Address:      current.Address,
					Type:         current.Type,
					Name:         current.Name,
					ProviderName: current.ProviderName,
					Mode:         current.Mode,
					Index:        current.Index,
				}

				riObj, err := rios.Decode(schema.ImpliedType())
				if err != nil {
					return nil, err
				}

				deposed.AttributeValues = marshalAttributeValues(riObj.Value)

				if len(riObj.Dependencies) > 0 {
					dependencies := make([]string, len(riObj.Dependencies))
					for i, v := range riObj.Dependencies {
						dependencies[i] = v.String()
					}
					deposed.DependsOn = dependencies
				}

				if riObj.Status == states.ObjectTainted {
					deposed.Tainted = true
				}
				deposed.DeposedKey = deposedKey.String()
				ret = append(ret, deposed)
			}
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Address < ret[j].Address
	})

	return ret, nil
}
