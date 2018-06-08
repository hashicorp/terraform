package statefile

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	version "github.com/hashicorp/go-version"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

func readStateV4(src []byte) (*File, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	sV4 := &stateV4{}
	err := json.Unmarshal(src, sV4)
	if err != nil {
		diags = diags.Append(jsonUnmarshalDiags(err))
		return nil, diags
	}

	file, prepDiags := prepareStateV4(sV4)
	diags = diags.Append(prepDiags)
	return file, diags
}

func prepareStateV4(sV4 *stateV4) (*File, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var tfVersion *version.Version
	if sV4.TerraformVersion != "" {
		var err error
		tfVersion, err = version.NewVersion(sV4.TerraformVersion)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid Terraform version string",
				fmt.Sprintf("State file claims to have been written by Terraform version %q, which is not a valid version string.", sV4.TerraformVersion),
			))
		}
	}

	file := &File{
		TerraformVersion: tfVersion,
		Serial:           sV4.Serial,
		Lineage:          sV4.Lineage,
	}

	state := states.NewState()

	for _, rsV4 := range sV4.Resources {
		rAddr := addrs.Resource{
			Type: rsV4.Type,
			Name: rsV4.Name,
		}
		switch rsV4.Mode {
		case "managed":
			rAddr.Mode = addrs.ManagedResourceMode
		case "data":
			rAddr.Mode = addrs.DataResourceMode
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid resource mode in state",
				fmt.Sprintf("State contains a resource with mode %q (%q %q) which is not supported.", rsV4.Mode, rAddr.Type, rAddr.Name),
			))
			continue
		}

		moduleAddr := addrs.RootModuleInstance
		if rsV4.Module != "" {
			var addrDiags tfdiags.Diagnostics
			moduleAddr, addrDiags = addrs.ParseModuleInstanceStr(rsV4.Module)
			diags = diags.Append(addrDiags)
			if addrDiags.HasErrors() {
				continue
			}
		}

		providerAddr, addrDiags := addrs.ParseAbsProviderConfigStr(rsV4.ProviderConfig)
		diags.Append(addrDiags)
		if addrDiags.HasErrors() {
			continue
		}

		var eachMode states.EachMode
		switch rsV4.EachMode {
		case "":
			eachMode = states.NoEach
		case "list":
			eachMode = states.EachList
		case "map":
			eachMode = states.EachMap
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid resource metadata in state",
				fmt.Sprintf("Resource %s has invalid \"each\" value %q in state.", rAddr.Absolute(moduleAddr), eachMode),
			))
			continue
		}

		ms := state.EnsureModule(moduleAddr)

		// Ensure the resource container object is present in the state.
		ms.SetResourceMeta(rAddr, eachMode, providerAddr)

		for _, isV4 := range rsV4.Instances {
			keyRaw := isV4.IndexKey
			var key addrs.InstanceKey
			switch tk := keyRaw.(type) {
			case int:
				key = addrs.IntKey(tk)
			case float64:
				// Since JSON only has one number type, reading from encoding/json
				// gives us a float64 here even if the number is whole.
				// float64 has a smaller integer range than int, but in practice
				// we rarely have more than a few tens of instances and so
				// it's unlikely that we'll exhaust the 52 bits in a float64.
				key = addrs.IntKey(int(tk))
			case string:
				key = addrs.StringKey(tk)
			default:
				if keyRaw != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid resource instance metadata in state",
						fmt.Sprintf("Resource %s has an instance with the invalid instance key %#v.", rAddr.Absolute(moduleAddr), keyRaw),
					))
					continue
				}
				key = addrs.NoKey
			}

			instAddr := rAddr.Instance(key)

			obj := &states.ResourceInstanceObject{
				SchemaVersion: isV4.SchemaVersion,
			}

			{
				// Instance attributes
				switch {
				case isV4.AttributesRaw != nil:
					obj.AttrsJSON = isV4.AttributesRaw
				case isV4.AttributesFlat != nil:
					obj.AttrsFlat = isV4.AttributesFlat
				default:
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid resource instance attributes in state",
						fmt.Sprintf("Instance %s does not have any stored attributes.", instAddr.Absolute(moduleAddr)),
					))
					continue
				}
			}

			{
				// Status
				raw := isV4.Status
				switch raw {
				case "":
					obj.Status = states.ObjectReady
				case "tainted":
					obj.Status = states.ObjectTainted
				default:
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid resource instance metadata in state",
						fmt.Sprintf("Instance %s has invalid status %q.", instAddr.Absolute(moduleAddr), raw),
					))
					continue
				}
			}

			if raw := isV4.PrivateRaw; len(raw) > 0 {
				// Private metadata
				ty, err := ctyjson.ImpliedType(raw)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid resource instance metadata in state",
						fmt.Sprintf("Instance %s has invalid private metadata: %s.", instAddr.Absolute(moduleAddr), err),
					))
					continue
				}

				val, err := ctyjson.Unmarshal(raw, ty)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid resource instance metadata in state",
						fmt.Sprintf("Instance %s has invalid private metadata: %s.", instAddr.Absolute(moduleAddr), err),
					))
					continue
				}

				obj.Private = val
			}

			{
				depsRaw := isV4.Dependencies
				deps := make([]addrs.Referenceable, 0, len(depsRaw))
				for _, depRaw := range depsRaw {
					ref, refDiags := addrs.ParseRefStr(depRaw)
					diags = diags.Append(refDiags)
					if refDiags.HasErrors() {
						continue
					}
					if len(ref.Remaining) != 0 {
						diags = diags.Append(tfdiags.Sourceless(
							tfdiags.Error,
							"Invalid resource instance metadata in state",
							fmt.Sprintf("Instance %s declares dependency on %q, which is not a reference to a dependable object.", instAddr.Absolute(moduleAddr), depRaw),
						))
					}
					deps = append(deps, ref.Subject)
				}
				obj.Dependencies = deps
			}

			switch {
			case isV4.Deposed != "":
				dk := states.DeposedKey(isV4.Deposed)
				if len(dk) != 8 {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Invalid resource instance metadata in state",
						fmt.Sprintf("Instance %s has an object with deposed key %q, which is not correctly formatted.", instAddr.Absolute(moduleAddr), isV4.Deposed),
					))
					continue
				}
				is := ms.ResourceInstance(instAddr)
				if is.HasDeposed(dk) {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Duplicate resource instance in state",
						fmt.Sprintf("Instance %s deposed object %q appears multiple times in the state file.", instAddr.Absolute(moduleAddr), dk),
					))
					continue
				}

				ms.SetResourceInstanceDeposed(instAddr, dk, obj)
			default:
				is := ms.ResourceInstance(instAddr)
				if is.HasCurrent() {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Duplicate resource instance in state",
						fmt.Sprintf("Instance %s appears multiple times in the state file.", instAddr.Absolute(moduleAddr)),
					))
					continue
				}

				ms.SetResourceInstanceCurrent(instAddr, obj, providerAddr)
			}
		}

		// We repeat this after creating the instances because
		// SetResourceInstanceCurrent automatically resets this metadata based
		// on the incoming objects. That behavior is useful when we're making
		// piecemeal updates to the state during an apply, but when we're
		// reading the state file we want to reflect its contents exactly.
		ms.SetResourceMeta(rAddr, eachMode, providerAddr)
	}

	// The root module is special in that we persist its attributes and thus
	// need to reload them now. (For descendent modules we just re-calculate
	// them based on the latest configuration on each run.)
	{
		rootModule := state.RootModule()
		for name, fos := range sV4.RootOutputs {
			os := &states.OutputValue{}
			os.Sensitive = fos.Sensitive

			ty, err := ctyjson.UnmarshalType([]byte(fos.ValueTypeRaw))
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid output value type in state",
					fmt.Sprintf("The state file has an invalid type specification for output %q: %s.", name, err),
				))
				continue
			}

			val, err := ctyjson.Unmarshal([]byte(fos.ValueRaw), ty)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid output value saved in state",
					fmt.Sprintf("The state file has an invalid value for output %q: %s.", name, err),
				))
				continue
			}

			os.Value = val
			rootModule.OutputValues[name] = os
		}
	}

	file.State = state
	return file, diags
}

func writeStateV4(file *File, w io.Writer) tfdiags.Diagnostics {
	// Here we'll convert back from the "File" representation to our
	// stateV4 struct representation and write that.
	//
	// While we support legacy state formats for reading, we only support the
	// latest for writing and so if a V5 is added in future then this function
	// should be deleted and replaced with a writeStateV5, even though the
	// read/prepare V4 functions above would stick around.

	var diags tfdiags.Diagnostics

	var terraformVersion string
	if file.TerraformVersion != nil {
		terraformVersion = file.TerraformVersion.String()
	}

	sV4 := &stateV4{
		TerraformVersion: terraformVersion,
		Serial:           file.Serial,
		Lineage:          file.Lineage,
		RootOutputs:      map[string]outputStateV4{},
		Resources:        []resourceStateV4{},
	}

	for name, os := range file.State.RootModule().OutputValues {
		src, err := ctyjson.Marshal(os.Value, os.Value.Type())
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to serialize output value in state",
				fmt.Sprintf("An error occured while serializing output value %q: %s.", name, err),
			))
			continue
		}

		typeSrc, err := ctyjson.MarshalType(os.Value.Type())
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to serialize output value in state",
				fmt.Sprintf("An error occured while serializing the type of output value %q: %s.", name, err),
			))
			continue
		}

		sV4.RootOutputs[name] = outputStateV4{
			Sensitive:    os.Sensitive,
			ValueRaw:     json.RawMessage(src),
			ValueTypeRaw: json.RawMessage(typeSrc),
		}
	}

	for _, ms := range file.State.Modules {
		moduleAddr := ms.Addr
		for _, rs := range ms.Resources {
			resourceAddr := rs.Addr

			var mode string
			switch resourceAddr.Mode {
			case addrs.ManagedResourceMode:
				mode = "managed"
			case addrs.DataResourceMode:
				mode = "data"
			default:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to serialize resource in state",
					fmt.Sprintf("Resource %s has mode %s, which cannot be serialized in state", resourceAddr.Absolute(moduleAddr), resourceAddr.Mode),
				))
				continue
			}

			var eachMode string
			switch rs.EachMode {
			case states.NoEach:
				eachMode = ""
			case states.EachList:
				eachMode = "list"
			case states.EachMap:
				eachMode = "map"
			default:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to serialize resource in state",
					fmt.Sprintf("Resource %s has \"each\" mode %s, which cannot be serialized in state", resourceAddr.Absolute(moduleAddr), rs.EachMode),
				))
				continue
			}

			sV4.Resources = append(sV4.Resources, resourceStateV4{
				Module:         moduleAddr.String(),
				Mode:           mode,
				Type:           resourceAddr.Type,
				Name:           resourceAddr.Name,
				EachMode:       eachMode,
				ProviderConfig: rs.ProviderConfig.String(),
				Instances:      []instanceObjectStateV4{},
			})
			rsV4 := &(sV4.Resources[len(sV4.Resources)-1])

			for key, is := range rs.Instances {
				if is.HasCurrent() {
					var objDiags tfdiags.Diagnostics
					rsV4.Instances, objDiags = appendInstanceObjectStateV4(
						rs, is, key, is.Current, states.NotDeposed,
						rsV4.Instances,
					)
					diags = diags.Append(objDiags)
				}
				for dk, obj := range is.Deposed {
					var objDiags tfdiags.Diagnostics
					rsV4.Instances, objDiags = appendInstanceObjectStateV4(
						rs, is, key, obj, dk,
						rsV4.Instances,
					)
					diags = diags.Append(objDiags)
				}
			}
		}
	}

	sV4.normalize()

	src, err := json.MarshalIndent(sV4, "", "  ")
	if err != nil {
		// Shouldn't happen if we do our conversion to *stateV4 correctly above.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to serialize state",
			fmt.Sprintf("An error occured while serializing the state to save it. This is a bug in Terraform and should be reported: %s.", err),
		))
		return diags
	}
	src = append(src, '\n')

	_, err = w.Write(src)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to write state",
			fmt.Sprintf("An error occured while writing the serialized state: %s.", err),
		))
		return diags
	}

	return diags
}

func appendInstanceObjectStateV4(rs *states.Resource, is *states.ResourceInstance, key addrs.InstanceKey, obj *states.ResourceInstanceObject, deposed states.DeposedKey, isV4s []instanceObjectStateV4) ([]instanceObjectStateV4, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var status string
	switch obj.Status {
	case states.ObjectReady:
		status = ""
	case states.ObjectTainted:
		status = "tainted"
	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to serialize resource instance in state",
			fmt.Sprintf("Instance %s has status %s, which cannot be saved in state.", rs.Addr.Instance(key), obj.Status),
		))
	}

	var privateRaw json.RawMessage
	if obj.Private != cty.NilVal {
		var err error
		privateRaw, err = ctyjson.Marshal(obj.Private, obj.Private.Type())
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to serialize resource instance in state",
				fmt.Sprintf("Failed to serialize instance %s private metadata: %s.", rs.Addr.Instance(key), err),
			))
		}
	}

	deps := make([]string, len(obj.Dependencies))
	for i, depAddr := range obj.Dependencies {
		deps[i] = depAddr.String()
	}

	var rawKey interface{}
	switch tk := key.(type) {
	case addrs.IntKey:
		rawKey = int(tk)
	case addrs.StringKey:
		rawKey = string(tk)
	default:
		if key != addrs.NoKey {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to serialize resource instance in state",
				fmt.Sprintf("Instance %s has an unsupported instance key: %#v.", rs.Addr.Instance(key), key),
			))
		}
	}

	return append(isV4s, instanceObjectStateV4{
		IndexKey:       rawKey,
		Deposed:        string(deposed),
		Status:         status,
		SchemaVersion:  obj.SchemaVersion,
		AttributesFlat: obj.AttrsFlat,
		AttributesRaw:  obj.AttrsJSON,
		PrivateRaw:     privateRaw,
		Dependencies:   deps,
	}), diags
}

type stateV4 struct {
	Version          stateVersionV4           `json:"version"`
	TerraformVersion string                   `json:"terraform_version"`
	Serial           uint64                   `json:"serial"`
	Lineage          string                   `json:"lineage"`
	RootOutputs      map[string]outputStateV4 `json:"outputs"`
	Resources        []resourceStateV4        `json:"resources"`
}

// normalize makes some in-place changes to normalize the way items are
// stored to ensure that two functionally-equivalent states will be stored
// identically.
func (s *stateV4) normalize() {
	sort.Stable(sortResourcesV4(s.Resources))
	for _, rs := range s.Resources {
		sort.Stable(sortInstancesV4(rs.Instances))
	}
}

type outputStateV4 struct {
	ValueRaw     json.RawMessage `json:"value"`
	ValueTypeRaw json.RawMessage `json:"type"`
	Sensitive    bool            `json:"sensitive,omitempty"`
}

type resourceStateV4 struct {
	Module         string                  `json:"module,omitempty"`
	Mode           string                  `json:"mode"`
	Type           string                  `json:"type"`
	Name           string                  `json:"name"`
	EachMode       string                  `json:"each,omitempty"`
	ProviderConfig string                  `json:"provider"`
	Instances      []instanceObjectStateV4 `json:"instances"`
}

type instanceObjectStateV4 struct {
	IndexKey interface{} `json:"index_key,omitempty"`
	Status   string      `json:"status,omitempty"`
	Deposed  string      `json:"deposed,omitempty"`

	SchemaVersion  uint64            `json:"schema_version"`
	AttributesRaw  json.RawMessage   `json:"attributes,omitempty"`
	AttributesFlat map[string]string `json:"attributes_flat,omitempty"`

	PrivateRaw json.RawMessage `json:"private,omitempty"`

	Dependencies []string `json:"depends_on,omitempty"`
}

// stateVersionV4 is a weird special type we use to produce our hard-coded
// "version": 4 in the JSON serialization.
type stateVersionV4 struct{}

func (sv stateVersionV4) MarshalJSON() ([]byte, error) {
	return []byte{'4'}, nil
}

func (sv stateVersionV4) UnmarshalJSON([]byte) error {
	// Nothing to do: we already know we're version 4
	return nil
}

type sortResourcesV4 []resourceStateV4

func (sr sortResourcesV4) Len() int      { return len(sr) }
func (sr sortResourcesV4) Swap(i, j int) { sr[i], sr[j] = sr[j], sr[i] }
func (sr sortResourcesV4) Less(i, j int) bool {
	switch {
	case sr[i].Mode != sr[j].Mode:
		return sr[i].Mode < sr[j].Mode
	case sr[i].Type != sr[j].Type:
		return sr[i].Type < sr[j].Type
	case sr[i].Name != sr[j].Name:
		return sr[i].Name < sr[j].Name
	default:
		return false
	}
}

type sortInstancesV4 []instanceObjectStateV4

func (si sortInstancesV4) Len() int      { return len(si) }
func (si sortInstancesV4) Swap(i, j int) { si[i], si[j] = si[j], si[i] }
func (si sortInstancesV4) Less(i, j int) bool {
	ki := si[i].IndexKey
	kj := si[j].IndexKey
	if ki != kj {
		if (ki == nil) != (kj == nil) {
			return ki == nil
		}
		if kii, isInt := ki.(int); isInt {
			if kji, isInt := kj.(int); isInt {
				return kii < kji
			}
			return true
		}
		if kis, isStr := ki.(string); isStr {
			if kjs, isStr := kj.(string); isStr {
				return kis < kjs
			}
			return true
		}
	}
	if si[i].Deposed != si[j].Deposed {
		return si[i].Deposed < si[j].Deposed
	}
	return false
}
