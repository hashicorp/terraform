package jsonstate

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/terraform"
)

// FormatVersion1 represents the original version of the json format and will
// be incremented in its minor version only for any significant change that
// a consumer may wish to vary its behavior to benefit from.
const FormatVersion1 = "1.0"

// FormatVersion2 represents the current minor version of the second generation
// of this format.
const FormatVersion2 = "2.0"

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
	Type      json.RawMessage `json:"type,omitempty"`
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

// Resource is the representation of a resource instance object in the state.
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
	//
	// This is populated only in state JSON version 1. Later versions use
	// "Attributes" instead.
	AttributeValues attributeValues `json:"values,omitempty"`

	// Attributes is a JSON representation of the attribute values of the
	// resource instance object, whose structure depends on the resource
	// type schema.
	//
	// This is populated only in state JSON version 2. Unlike AttributeValues
	// for state JSON version 1, this is the raw JSON attribute data copied
	// verbatim from the latest state snapshot without any subsequent
	// transformations, and so consumers of this result must use the schema
	// for this resource type to robustly consume the attributes. In particular,
	// any attribute that is marked as dynamically-typed in the schema will
	// be represented as a JSON object with "type" and "value" properties,
	// which is ambiguous with a statically-typed nested object value and
	// can only be distinguished using the schema.
	Attributes json.RawMessage `json:"attributes,omitempty"`

	// AttributesLegacyFlatmap is set when the Attributes field contains a
	// legacy flatmap representation of resource data that hasn't yet been
	// updated to the v0.12-and-later nested JSON format.
	AttributesLegacyFlatmap bool `json:"attributes_legacy_flatmap,omitempty"`

	// SensitiveValues is similar to AttributeValues, but with all sensitive
	// values replaced with true, and all non-sensitive leaf values omitted.
	//
	// This is populated only in state JSON version 1. Later versions
	// populate SensitivePaths instead.
	SensitiveValues json.RawMessage `json:"sensitive_values,omitempty"`

	// SensitivePaths is a set of paths through the Attributes object to
	// any nested attributes that are marked as sensitive.
	//
	// This is populated only in state JSON version 2 and later. Older versions
	// populate SensitiveValues instead.
	SensitivePaths []Path `json:"sensitive_paths,omitempty"`

	// DependsOn contains a list of the resource's dependencies. The entries are
	// addresses relative to the containing module.
	DependsOn []string `json:"depends_on,omitempty"`

	// Tainted is true if the resource is tainted in terraform state.
	Tainted bool `json:"tainted,omitempty"`

	// Deposed is set if the resource is deposed in terraform state.
	DeposedKey string `json:"deposed_key,omitempty"`
}

// Path is a JSON-serializable equivalent of a cty.Path, serialized as an
// object containing an array rather than just an array to allow for potential
// backward-compatible future expansion.
//
// Path is slightly lossy when compared to cty.Path because it doesn't
// distinguish between indexing into a map and accessing an attribute of an
// object, both of which will be serialized as JSON strings. That's defensible
// for JSON because JSON itself does not distinguish between maps and objects,
// and so this simplification gives sufficient detail to traverse through any
// JSON data structure returned elsewhere in our JSON output. Callers who
// _do_ care to distinguish object and map traversal must refer to out-of-band
// information, such as a resource type schema.
type Path struct {
	Steps []addrs.InstanceKey `json:"steps"`
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

// newState1() returns a minimally-initialized state version 1
func newState1() *state {
	return &state{
		FormatVersion: FormatVersion1,
	}
}

// newState1() returns a minimally-initialized state version 1
func newState2() *state {
	return &state{
		FormatVersion: FormatVersion2,
	}
}

// Marshal1 generates a JSON state in the original format, which requires
// access to all of the provider schemas in order to make small transformations
// to the attribute serialization for resources.
func Marshal1(sf *statefile.File, schemas *terraform.Schemas) ([]byte, error) {
	if schemas == nil {
		panic("can't generate state version 1 with nil schemas")
	}

	output := newState1()

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

// Marshal2 generates a JSON state in a newer format, which is broadly the
// same shape as format 1.0 except that it encodes resource attribute data
// in a raw form that is exactly what's stored in the given state file,
// without any convenience transformations that would require schema access
// to implement.
func Marshal2(sf *statefile.File) ([]byte, error) {
	output := newState2()

	if sf == nil || sf.State.Empty() {
		ret, err := json.Marshal(output)
		return ret, err
	}

	if sf.TerraformVersion != nil {
		output.TerraformVersion = sf.TerraformVersion.String()
	}

	// output.StateValues
	err := output.marshalStateValues(sf.State, nil)
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
	sv.Outputs, err = MarshalOutputs(s.RootModule().OutputValues)
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

// MarshalOutputs translates a map of states.OutputValue to a map of jsonstate.output,
// which are defined for json encoding.
func MarshalOutputs(outputs map[string]*states.OutputValue) (map[string]output, error) {
	if outputs == nil {
		return nil, nil
	}

	ret := make(map[string]output)
	for k, v := range outputs {
		ty := v.Value.Type()
		ov, err := ctyjson.Marshal(v.Value, ty)
		if err != nil {
			return ret, err
		}
		ot, err := ctyjson.MarshalType(ty)
		if err != nil {
			return ret, err
		}
		ret[k] = output{
			Value:     ov,
			Type:      ot,
			Sensitive: v.Sensitive,
		}
	}

	return ret, nil
}

func marshalRootModule(s *states.State, schemas *terraform.Schemas) (module, error) {
	var ret module
	var err error

	ret.Address = ""
	rs, err := marshalResources(s.RootModule().Resources, addrs.RootModuleInstance, schemas)
	if err != nil {
		return ret, err
	}
	ret.Resources = rs

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

			// "schema" is populated only if we're marshalling to state JSON
			// version 1, which requires some transformations that can only
			// be done with access to the schema. For later versions we don't
			// use the schema and just export exactly what's written in the
			// underlying state snapshot.
			var schema *configschema.Block
			var version uint64
			if schemas != nil {
				schema, version = schemas.ResourceTypeConfig(
					r.ProviderConfig.Provider,
					resAddr.Mode,
					resAddr.Type,
				)
				if schema == nil {
					return nil, fmt.Errorf("no schema found for %s (in provider %s)", resAddr.String(), r.ProviderConfig.Provider)
				}
			}

			if schema != nil {
				// Generating JSON state version 1

				// It is possible that the only instance is deposed
				if ri.Current != nil {
					if version != ri.Current.SchemaVersion {
						return nil, fmt.Errorf("schema version %d for %s in state does not match version %d from the provider", ri.Current.SchemaVersion, resAddr, version)
					}

					current.SchemaVersion = ri.Current.SchemaVersion

					riObj, err := ri.Current.Decode(schema.ImpliedType())
					if err != nil {
						return nil, err
					}

					current.AttributeValues = marshalAttributeValues(riObj.Value)

					s := SensitiveAsBool(riObj.Value)
					v, err := ctyjson.Marshal(s, s.Type())
					if err != nil {
						return nil, err
					}
					current.SensitiveValues = v

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

					s := SensitiveAsBool(riObj.Value)
					v, err := ctyjson.Marshal(s, s.Type())
					if err != nil {
						return nil, err
					}
					deposed.SensitiveValues = v

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

			} else {
				// Generating JSON state version 2

				// It is possible that the only instance is deposed
				if ri.Current != nil {
					if version != ri.Current.SchemaVersion {
						return nil, fmt.Errorf("schema version %d for %s in state does not match version %d from the provider", ri.Current.SchemaVersion, resAddr, version)
					}

					current.SchemaVersion = ri.Current.SchemaVersion
					riObj := ri.Current

					switch {
					case len(riObj.AttrsJSON) != 0:
						current.Attributes = riObj.AttrsJSON
					default:
						// Legacy flatmap mode: we serialize this as a JSON
						// object with properties that are all string values,
						// directly exposing the flatmap form. This is an
						// annoying special case but thankfully now incredibly
						// rare since nothing has been generating this format
						// since Terraform v0.12.
						current.AttributesLegacyFlatmap = true

						attrsJSON, err := json.Marshal(riObj.AttrsFlat)
						if err != nil {
							// Should not get here, since there's no reason
							// for serializing a map[string]string to fail.
							return nil, fmt.Errorf("failed to serialize flatmap attributes for %s: %s", resAddr, err)
						}
						current.Attributes = attrsJSON
					}

					current.SensitivePaths = SensitivePaths(riObj.AttrSensitivePaths)

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

				for deposedKey, riObj := range ri.Deposed {
					// copy the base fields from the current instance
					deposed := resource{
						Address:      current.Address,
						Type:         current.Type,
						Name:         current.Name,
						ProviderName: current.ProviderName,
						Mode:         current.Mode,
						Index:        current.Index,
					}

					switch {
					case len(riObj.AttrsJSON) != 0:
						deposed.Attributes = riObj.AttrsJSON
					default:
						// Legacy flatmap mode: we serialize this as a JSON
						// object with properties that are all string values,
						// directly exposing the flatmap form. This is an
						// annoying special case but thankfully now incredibly
						// rare since nothing has been generating this format
						// since Terraform v0.12.
						deposed.AttributesLegacyFlatmap = true

						attrsJSON, err := json.Marshal(riObj.AttrsFlat)
						if err != nil {
							// Should not get here, since there's no reason
							// for serializing a map[string]string to fail.
							return nil, fmt.Errorf("failed to serialize flatmap attributes for %s: %s", resAddr, err)
						}
						deposed.Attributes = attrsJSON
					}

					deposed.SensitivePaths = SensitivePaths(riObj.AttrSensitivePaths)

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
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Address < ret[j].Address
	})

	return ret, nil
}

func SensitiveAsBool(val cty.Value) cty.Value {
	if val.HasMark(marks.Sensitive) {
		return cty.True
	}

	ty := val.Type()
	switch {
	case val.IsNull(), ty.IsPrimitiveType(), ty.Equals(cty.DynamicPseudoType):
		return cty.False
	case ty.IsListType() || ty.IsTupleType() || ty.IsSetType():
		if !val.IsKnown() {
			// If the collection is unknown we can't say anything about the
			// sensitivity of its contents
			return cty.EmptyTupleVal
		}
		length := val.LengthInt()
		if length == 0 {
			// If there are no elements then we can't have sensitive values
			return cty.EmptyTupleVal
		}
		vals := make([]cty.Value, 0, length)
		it := val.ElementIterator()
		for it.Next() {
			_, v := it.Element()
			vals = append(vals, SensitiveAsBool(v))
		}
		// The above transform may have changed the types of some of the
		// elements, so we'll always use a tuple here in case we've now made
		// different elements have different types. Our ultimate goal is to
		// marshal to JSON anyway, and all of these sequence types are
		// indistinguishable in JSON.
		return cty.TupleVal(vals)
	case ty.IsMapType() || ty.IsObjectType():
		if !val.IsKnown() {
			// If the map/object is unknown we can't say anything about the
			// sensitivity of its attributes
			return cty.EmptyObjectVal
		}
		var length int
		switch {
		case ty.IsMapType():
			length = val.LengthInt()
		default:
			length = len(val.Type().AttributeTypes())
		}
		if length == 0 {
			// If there are no elements then we can't have sensitive values
			return cty.EmptyObjectVal
		}
		vals := make(map[string]cty.Value)
		it := val.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			s := SensitiveAsBool(v)
			// Omit all of the "false"s for non-sensitive values for more
			// compact serialization
			if !s.RawEquals(cty.False) {
				vals[k.AsString()] = s
			}
		}
		// The above transform may have changed the types of some of the
		// elements, so we'll always use an object here in case we've now made
		// different elements have different types. Our ultimate goal is to
		// marshal to JSON anyway, and all of these mapping types are
		// indistinguishable in JSON.
		return cty.ObjectVal(vals)
	default:
		// Should never happen, since the above should cover all types
		panic(fmt.Sprintf("sensitiveAsBool cannot handle %#v", val))
	}
}

// SensitivePaths generates a JSON-serializable representation of the sensitive
// value paths in the given set of marked paths.
func SensitivePaths(markedPaths []cty.PathValueMarks) []Path {
	if len(markedPaths) == 0 {
		// Can't possibly have any sensitive values then
		return nil
	}
	ret := make([]Path, 0, len(markedPaths))
	for _, markedPath := range markedPaths {
		if _, ok := markedPath.Marks[marks.Sensitive]; !ok {
			continue // Only interested in "sensitive" marks
		}
		jsonPath := make([]addrs.InstanceKey, len(markedPath.Path))
		for i, step := range markedPath.Path {
			switch step := step.(type) {

			case cty.IndexStep:
				switch {
				case step.Key.Type() == cty.String:
					jsonPath[i] = addrs.StringKey(step.Key.AsString())
				case step.Key.Type() == cty.Number:
					var idx int
					err := gocty.FromCtyValue(step.Key, &idx)
					if err != nil {
						panic(fmt.Sprintf("invalid index in path: %s", err))
					}
					jsonPath[i] = addrs.IntKey(idx)
				default:
					panic(fmt.Sprintf("invalid index in path: %#v", step.Key.Type()))
				}
			case cty.GetAttrStep:
				jsonPath[i] = addrs.StringKey(step.Name)
			}
		}
		ret = append(ret, Path{
			Steps: jsonPath,
		})
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}
