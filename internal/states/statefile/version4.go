package statefile

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	version "github.com/hashicorp/go-version"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
			// If ParseAbsProviderConfigStr returns an error, the state may have
			// been written before Provider FQNs were introduced and the
			// AbsProviderConfig string format will need normalization. If so,
			// we treat it like a legacy provider (namespace "-") and let the
			// provider installer handle detecting the FQN.
			var legacyAddrDiags tfdiags.Diagnostics
			providerAddr, legacyAddrDiags = addrs.ParseLegacyAbsProviderConfigStr(rsV4.ProviderConfig)
			if legacyAddrDiags.HasErrors() {
				continue
			}
		}

		ms := state.EnsureModule(moduleAddr)

		// Ensure the resource container object is present in the state.
		ms.SetResourceProvider(rAddr, providerAddr)

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

			obj := &states.ResourceInstanceObjectSrc{
				SchemaVersion:       isV4.SchemaVersion,
				CreateBeforeDestroy: isV4.CreateBeforeDestroy,
			}

			{
				// Instance attributes
				switch {
				case isV4.AttributesRaw != nil:
					obj.AttrsJSON = isV4.AttributesRaw
				case isV4.AttributesFlat != nil:
					obj.AttrsFlat = isV4.AttributesFlat
				default:
					// This is odd, but we'll accept it and just treat the
					// object has being empty. In practice this should arise
					// only from the contrived sort of state objects we tend
					// to hand-write inline in tests.
					obj.AttrsJSON = []byte{'{', '}'}
				}
			}

			// Sensitive paths
			if isV4.AttributeSensitivePaths != nil {
				paths, pathsDiags := unmarshalPaths([]byte(isV4.AttributeSensitivePaths))
				diags = diags.Append(pathsDiags)
				if pathsDiags.HasErrors() {
					continue
				}

				var pvm []cty.PathValueMarks
				for _, path := range paths {
					pvm = append(pvm, cty.PathValueMarks{
						Path:  path,
						Marks: cty.NewValueMarks(marks.Sensitive),
					})
				}
				obj.AttrSensitivePaths = pvm
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
				obj.Private = raw
			}

			{
				depsRaw := isV4.Dependencies
				deps := make([]addrs.ConfigResource, 0, len(depsRaw))
				for _, depRaw := range depsRaw {
					addr, addrDiags := addrs.ParseAbsResourceStr(depRaw)
					diags = diags.Append(addrDiags)
					if addrDiags.HasErrors() {
						continue
					}
					deps = append(deps, addr.Config())
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

				ms.SetResourceInstanceDeposed(instAddr, dk, obj, providerAddr)
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
		ms.SetResourceProvider(rAddr, providerAddr)
	}

	// The root module is special in that we persist its attributes and thus
	// need to reload them now. (For descendent modules we just re-calculate
	// them based on the latest configuration on each run.)
	{
		rootModule := state.RootModule()
		for name, fos := range sV4.RootOutputs {
			os := &states.OutputValue{
				Addr: addrs.AbsOutputValue{
					OutputValue: addrs.OutputValue{
						Name: name,
					},
				},
			}
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

	// Saved check results from the previous run, if any.
	// We differentiate absense from an empty array here so that we can
	// recognize if the previous run was with a version of Terraform that
	// didn't support checks yet, or if there just weren't any checkable
	// objects to record, in case that's important for certain messaging.
	if sV4.CheckResults != nil {
		var moreDiags tfdiags.Diagnostics
		state.CheckResults, moreDiags = decodeCheckResultsV4(sV4.CheckResults)
		diags = diags.Append(moreDiags)
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
	if file == nil || file.State == nil {
		panic("attempt to write nil state to file")
	}

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
			resourceAddr := rs.Addr.Resource

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

			sV4.Resources = append(sV4.Resources, resourceStateV4{
				Module:         moduleAddr.String(),
				Mode:           mode,
				Type:           resourceAddr.Type,
				Name:           resourceAddr.Name,
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

	sV4.CheckResults = encodeCheckResultsV4(file.State.CheckResults)

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

func appendInstanceObjectStateV4(rs *states.Resource, is *states.ResourceInstance, key addrs.InstanceKey, obj *states.ResourceInstanceObjectSrc, deposed states.DeposedKey, isV4s []instanceObjectStateV4) ([]instanceObjectStateV4, tfdiags.Diagnostics) {
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

	var privateRaw []byte
	if len(obj.Private) > 0 {
		privateRaw = obj.Private
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

	// Extract paths from path value marks
	var paths []cty.Path
	for _, vm := range obj.AttrSensitivePaths {
		paths = append(paths, vm.Path)
	}

	// Marshal paths to JSON
	attributeSensitivePaths, pathsDiags := marshalPaths(paths)
	diags = diags.Append(pathsDiags)

	return append(isV4s, instanceObjectStateV4{
		IndexKey:                rawKey,
		Deposed:                 string(deposed),
		Status:                  status,
		SchemaVersion:           obj.SchemaVersion,
		AttributesFlat:          obj.AttrsFlat,
		AttributesRaw:           obj.AttrsJSON,
		AttributeSensitivePaths: attributeSensitivePaths,
		PrivateRaw:              privateRaw,
		Dependencies:            deps,
		CreateBeforeDestroy:     obj.CreateBeforeDestroy,
	}), diags
}

func decodeCheckResultsV4(in []checkResultsV4) (*states.CheckResults, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	ret := &states.CheckResults{}
	if len(in) == 0 {
		return ret, diags
	}

	ret.ConfigResults = addrs.MakeMap[addrs.ConfigCheckable, *states.CheckResultAggregate]()
	for _, aggrIn := range in {
		objectKind := decodeCheckableObjectKindV4(aggrIn.ObjectKind)
		if objectKind == addrs.CheckableKindInvalid {
			diags = diags.Append(fmt.Errorf("unsupported checkable object kind %q", aggrIn.ObjectKind))
			continue
		}

		// Some trickiness here: we only have an address parser for
		// addrs.Checkable and not for addrs.ConfigCheckable, but that's okay
		// because once we have an addrs.Checkable we can always derive an
		// addrs.ConfigCheckable from it, and a ConfigCheckable should always
		// be the same syntax as a Checkable with no index information and
		// thus we can reuse the same parser for both here.
		configAddrProxy, moreDiags := addrs.ParseCheckableStr(objectKind, aggrIn.ConfigAddr)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			continue
		}
		configAddr := configAddrProxy.ConfigCheckable()
		if configAddr.String() != configAddrProxy.String() {
			// This is how we catch if the config address included index
			// information that would be allowed in a Checkable but not
			// in a ConfigCheckable.
			diags = diags.Append(fmt.Errorf("invalid checkable config address %s", aggrIn.ConfigAddr))
			continue
		}

		aggr := &states.CheckResultAggregate{
			Status: decodeCheckStatusV4(aggrIn.Status),
		}

		if len(aggrIn.Objects) != 0 {
			aggr.ObjectResults = addrs.MakeMap[addrs.Checkable, *states.CheckResultObject]()
			for _, objectIn := range aggrIn.Objects {
				objectAddr, moreDiags := addrs.ParseCheckableStr(objectKind, objectIn.ObjectAddr)
				diags = diags.Append(moreDiags)
				if moreDiags.HasErrors() {
					continue
				}

				obj := &states.CheckResultObject{
					Status:          decodeCheckStatusV4(objectIn.Status),
					FailureMessages: objectIn.FailureMessages,
				}
				aggr.ObjectResults.Put(objectAddr, obj)
			}
		}

		ret.ConfigResults.Put(configAddr, aggr)
	}

	return ret, diags
}

func encodeCheckResultsV4(in *states.CheckResults) []checkResultsV4 {
	// normalize empty and nil sets in the serialized state
	if in == nil || in.ConfigResults.Len() == 0 {
		return nil
	}

	ret := make([]checkResultsV4, 0, in.ConfigResults.Len())

	for _, configElem := range in.ConfigResults.Elems {
		configResultsOut := checkResultsV4{
			ObjectKind: encodeCheckableObjectKindV4(configElem.Key.CheckableKind()),
			ConfigAddr: configElem.Key.String(),
			Status:     encodeCheckStatusV4(configElem.Value.Status),
		}
		for _, objectElem := range configElem.Value.ObjectResults.Elems {
			configResultsOut.Objects = append(configResultsOut.Objects, checkResultsObjectV4{
				ObjectAddr:      objectElem.Key.String(),
				Status:          encodeCheckStatusV4(objectElem.Value.Status),
				FailureMessages: objectElem.Value.FailureMessages,
			})
		}

		ret = append(ret, configResultsOut)
	}

	return ret
}

func decodeCheckStatusV4(in string) checks.Status {
	switch in {
	case "pass":
		return checks.StatusPass
	case "fail":
		return checks.StatusFail
	case "error":
		return checks.StatusError
	default:
		// We'll treat anything else as unknown just as a concession to
		// forward-compatible parsing, in case a later version of Terraform
		// introduces a new status.
		return checks.StatusUnknown
	}
}

func encodeCheckStatusV4(in checks.Status) string {
	switch in {
	case checks.StatusPass:
		return "pass"
	case checks.StatusFail:
		return "fail"
	case checks.StatusError:
		return "error"
	case checks.StatusUnknown:
		return "unknown"
	default:
		panic(fmt.Sprintf("unsupported check status %s", in))
	}
}

func decodeCheckableObjectKindV4(in string) addrs.CheckableKind {
	switch in {
	case "resource":
		return addrs.CheckableResource
	case "output":
		return addrs.CheckableOutputValue
	default:
		// We'll treat anything else as invalid just as a concession to
		// forward-compatible parsing, in case a later version of Terraform
		// introduces a new status.
		return addrs.CheckableKindInvalid
	}
}

func encodeCheckableObjectKindV4(in addrs.CheckableKind) string {
	switch in {
	case addrs.CheckableResource:
		return "resource"
	case addrs.CheckableOutputValue:
		return "output"
	default:
		panic(fmt.Sprintf("unsupported checkable object kind %s", in))
	}
}

type stateV4 struct {
	Version          stateVersionV4           `json:"version"`
	TerraformVersion string                   `json:"terraform_version"`
	Serial           uint64                   `json:"serial"`
	Lineage          string                   `json:"lineage"`
	RootOutputs      map[string]outputStateV4 `json:"outputs"`
	Resources        []resourceStateV4        `json:"resources"`
	CheckResults     []checkResultsV4         `json:"check_results"`
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

	SchemaVersion           uint64            `json:"schema_version"`
	AttributesRaw           json.RawMessage   `json:"attributes,omitempty"`
	AttributesFlat          map[string]string `json:"attributes_flat,omitempty"`
	AttributeSensitivePaths json.RawMessage   `json:"sensitive_attributes,omitempty"`

	PrivateRaw []byte `json:"private,omitempty"`

	Dependencies []string `json:"dependencies,omitempty"`

	CreateBeforeDestroy bool `json:"create_before_destroy,omitempty"`
}

type checkResultsV4 struct {
	ObjectKind string                 `json:"object_kind"`
	ConfigAddr string                 `json:"config_addr"`
	Status     string                 `json:"status"`
	Objects    []checkResultsObjectV4 `json:"objects"`
}

type checkResultsObjectV4 struct {
	ObjectAddr      string   `json:"object_addr"`
	Status          string   `json:"status"`
	FailureMessages []string `json:"failure_messages,omitempty"`
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
	case sr[i].Module != sr[j].Module:
		return sr[i].Module < sr[j].Module
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

// pathStep is an intermediate representation of a cty.PathStep to facilitate
// consistent JSON serialization. The Value field can either be a cty.Value of
// dynamic type (for index steps), or a string (for get attr steps).
type pathStep struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value"`
}

const (
	indexPathStepType   = "index"
	getAttrPathStepType = "get_attr"
)

func unmarshalPaths(buf []byte) ([]cty.Path, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var jsonPaths [][]pathStep

	err := json.Unmarshal(buf, &jsonPaths)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error unmarshaling path steps",
			err.Error(),
		))
	}

	paths := make([]cty.Path, 0, len(jsonPaths))

unmarshalOuter:
	for _, jsonPath := range jsonPaths {
		var path cty.Path
		for _, jsonStep := range jsonPath {
			switch jsonStep.Type {
			case indexPathStepType:
				key, err := ctyjson.Unmarshal(jsonStep.Value, cty.DynamicPseudoType)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Error unmarshaling path step",
						fmt.Sprintf("Failed to unmarshal index step key: %s", err),
					))
					continue unmarshalOuter
				}
				path = append(path, cty.IndexStep{Key: key})
			case getAttrPathStepType:
				var name string
				if err := json.Unmarshal(jsonStep.Value, &name); err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Error unmarshaling path step",
						fmt.Sprintf("Failed to unmarshal get attr step name: %s", err),
					))
					continue unmarshalOuter
				}
				path = append(path, cty.GetAttrStep{Name: name})
			default:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Unsupported path step",
					fmt.Sprintf("Unsupported path step type %q", jsonStep.Type),
				))
				continue unmarshalOuter
			}
		}
		paths = append(paths, path)
	}

	return paths, diags
}

func marshalPaths(paths []cty.Path) ([]byte, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// cty.Path is a slice of cty.PathSteps, so our representation of a slice
	// of paths is a nested slice of our intermediate pathStep struct
	jsonPaths := make([][]pathStep, 0, len(paths))

marshalOuter:
	for _, path := range paths {
		jsonPath := make([]pathStep, 0, len(path))
		for _, step := range path {
			var jsonStep pathStep
			switch s := step.(type) {
			case cty.IndexStep:
				key, err := ctyjson.Marshal(s.Key, cty.DynamicPseudoType)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Error marshaling path step",
						fmt.Sprintf("Failed to marshal index step key %#v: %s", s.Key, err),
					))
					continue marshalOuter
				}
				jsonStep.Type = indexPathStepType
				jsonStep.Value = key
			case cty.GetAttrStep:
				name, err := json.Marshal(s.Name)
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Error marshaling path step",
						fmt.Sprintf("Failed to marshal get attr step name %s: %s", s.Name, err),
					))
					continue marshalOuter
				}
				jsonStep.Type = getAttrPathStepType
				jsonStep.Value = name
			default:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Unsupported path step",
					fmt.Sprintf("Unsupported path step %#v (%t)", step, step),
				))
				continue marshalOuter
			}
			jsonPath = append(jsonPath, jsonStep)
		}
		jsonPaths = append(jsonPaths, jsonPath)
	}

	buf, err := json.Marshal(jsonPaths)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error marshaling path steps",
			fmt.Sprintf("Failed to marshal path steps: %s", err),
		))
	}

	return buf, diags
}
