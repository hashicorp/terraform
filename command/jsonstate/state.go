package jsonstate

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
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
	FormatVersion    string      `json:"format_version,omitempty"`
	TerraformVersion string      `json:"terraform_version"`
	Values           stateValues `json:"values,omitempty"`
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
	SchemaVersion uint64 `json:"schema_version,omitempty"`

	// AttributeValues is the JSON representation of the attribute values of the
	// resource, whose structure depends on the resource type schema. Any
	// unknown values are omitted or set to null, making them indistinguishable
	// from absent values.
	AttributeValues attributeValues `json:"values,omitempty"`
}

// attributeValues is the JSON representation of the attribute values of the
// resource, whose structure depends on the resource type schema.
type attributeValues map[string]interface{}

func marshalAttributeValues(value cty.Value, schema *configschema.Block) attributeValues {
	if value == cty.NilVal {
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
	if sf == nil || sf.State.Empty() {
		return nil, nil
	}

	output := newState()
	output.TerraformVersion = sf.TerraformVersion.String()

	// output.StateValues
	err := output.marshalStateValues(sf.State, schemas)
	if err != nil {
		return nil, err
	}

	ret, err := json.MarshalIndent(output, "", "  ")
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

	jsonstate.Values = sv
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
	ret.Resources, err = marshalResources(s.RootModule().Resources, schemas)
	if err != nil {
		return ret, err
	}

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
	ret.ChildModules, err = marshalModules(s, schemas, moduleMap[""], moduleMap)
	return ret, err
}

// marshalModules is an ungainly recursive function to build a module
// structure out of a teraform state.
func marshalModules(
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
		rs, err := marshalResources(stateMod.Resources, schemas)
		if err != nil {
			return nil, err
		}
		cm.Resources = rs
		if moduleMap[child.String()] != nil {
			moreChildModules, err := marshalModules(s, schemas, moduleMap[child.String()], moduleMap)
			if err != nil {
				return nil, err
			}
			cm.ChildModules = moreChildModules
		}

		ret = append(ret, cm)
	}

	return ret, nil
}

func marshalResources(resources map[string]*states.Resource, schemas *terraform.Schemas) ([]resource, error) {
	var ret []resource

	for _, r := range resources {
		for k, ri := range r.Instances {

			resource := resource{
				Address:      r.Addr.String(),
				Type:         r.Addr.Type,
				Name:         r.Addr.Name,
				ProviderName: r.ProviderConfig.ProviderConfig.StringCompact(),
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

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Address < ret[j].Address
	})

	return ret, nil
}
