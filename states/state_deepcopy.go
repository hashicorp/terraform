package states

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

// Taking deep copies of states is an important operation because state is
// otherwise a mutable data structure that is challenging to share across
// many separate callers. It is important that the DeepCopy implementations
// in this file comprehensively copy all parts of the state data structure
// that could be mutated via pointers.

// DeepCopy returns a new state that contains equivalent data to the reciever
// but shares no backing memory in common.
//
// As with all methods on State, this method is not safe to use concurrently
// with writing to any portion of the recieving data structure. It is the
// caller's responsibility to ensure mutual exclusion for the duration of the
// operation, but may then freely modify the receiver and the returned copy
// independently once this method returns.
func (s *State) DeepCopy() *State {
	if s == nil {
		return nil
	}

	modules := make(map[string]*Module, len(s.Modules))
	for k, m := range s.Modules {
		modules[k] = m.DeepCopy()
	}
	return &State{
		Modules: modules,
	}
}

// DeepCopy returns a new module state that contains equivalent data to the
// receiver but shares no backing memory in common.
//
// As with all methods on Module, this method is not safe to use concurrently
// with writing to any portion of the recieving data structure. It is the
// caller's responsibility to ensure mutual exclusion for the duration of the
// operation, but may then freely modify the receiver and the returned copy
// independently once this method returns.
func (ms *Module) DeepCopy() *Module {
	if ms == nil {
		return nil
	}

	resources := make(map[string]*Resource, len(ms.Resources))
	for k, r := range ms.Resources {
		resources[k] = r.DeepCopy()
	}
	outputValues := make(map[string]*OutputValue, len(ms.OutputValues))
	for k, v := range ms.OutputValues {
		outputValues[k] = v.DeepCopy()
	}
	localValues := make(map[string]cty.Value, len(ms.LocalValues))
	for k, v := range ms.LocalValues {
		// cty.Value is immutable, so we don't need to copy these.
		localValues[k] = v
	}

	return &Module{
		Addr:         ms.Addr, // technically mutable, but immutable by convention
		Resources:    resources,
		OutputValues: outputValues,
		LocalValues:  localValues,
	}
}

// DeepCopy returns a new resource state that contains equivalent data to the
// receiver but shares no backing memory in common.
//
// As with all methods on Resource, this method is not safe to use concurrently
// with writing to any portion of the recieving data structure. It is the
// caller's responsibility to ensure mutual exclusion for the duration of the
// operation, but may then freely modify the receiver and the returned copy
// independently once this method returns.
func (rs *Resource) DeepCopy() *Resource {
	if rs == nil {
		return nil
	}

	instances := make(map[addrs.InstanceKey]*ResourceInstance, len(rs.Instances))
	for k, i := range rs.Instances {
		instances[k] = i.DeepCopy()
	}

	return &Resource{
		Addr:           rs.Addr,
		EachMode:       rs.EachMode,
		Instances:      instances,
		ProviderConfig: rs.ProviderConfig, // technically mutable, but immutable by convention
	}
}

// DeepCopy returns a new resource instance state that contains equivalent data
// to the receiver but shares no backing memory in common.
//
// As with all methods on ResourceInstance, this method is not safe to use
// concurrently with writing to any portion of the recieving data structure. It
// is the caller's responsibility to ensure mutual exclusion for the duration
// of the operation, but may then freely modify the receiver and the returned
// copy independently once this method returns.
func (is *ResourceInstance) DeepCopy() *ResourceInstance {
	if is == nil {
		return nil
	}

	deposed := make(map[DeposedKey]*ResourceInstanceObjectSrc, len(is.Deposed))
	for k, obj := range is.Deposed {
		deposed[k] = obj.DeepCopy()
	}

	return &ResourceInstance{
		Current: is.Current.DeepCopy(),
		Deposed: deposed,
	}
}

// DeepCopy returns a new resource instance object that contains equivalent data
// to the receiver but shares no backing memory in common.
//
// As with all methods on ResourceInstanceObject, this method is not safe to use
// concurrently with writing to any portion of the recieving data structure. It
// is the caller's responsibility to ensure mutual exclusion for the duration
// of the operation, but may then freely modify the receiver and the returned
// copy independently once this method returns.
func (obj *ResourceInstanceObjectSrc) DeepCopy() *ResourceInstanceObjectSrc {
	if obj == nil {
		return nil
	}

	var attrsFlat map[string]string
	if obj.AttrsFlat != nil {
		attrsFlat = make(map[string]string, len(obj.AttrsFlat))
		for k, v := range obj.AttrsFlat {
			attrsFlat[k] = v
		}
	}

	var attrsJSON []byte
	if obj.AttrsJSON != nil {
		attrsJSON = make([]byte, len(obj.AttrsJSON))
		copy(attrsJSON, obj.AttrsJSON)
	}

	// Some addrs.Referencable implementations are technically mutable, but
	// we treat them as immutable by convention and so we don't deep-copy here.
	dependencies := make([]addrs.Referenceable, len(obj.Dependencies))
	copy(dependencies, obj.Dependencies)

	return &ResourceInstanceObjectSrc{
		Status:        obj.Status,
		SchemaVersion: obj.SchemaVersion,
		Private:       obj.Private,
		AttrsFlat:     attrsFlat,
		AttrsJSON:     attrsJSON,
		Dependencies:  dependencies,
	}
}

// DeepCopy returns a new output value state that contains equivalent data
// to the receiver but shares no backing memory in common.
//
// As with all methods on OutputValue, this method is not safe to use
// concurrently with writing to any portion of the recieving data structure. It
// is the caller's responsibility to ensure mutual exclusion for the duration
// of the operation, but may then freely modify the receiver and the returned
// copy independently once this method returns.
func (os *OutputValue) DeepCopy() *OutputValue {
	if os == nil {
		return nil
	}

	return &OutputValue{
		Value:     os.Value,
		Sensitive: os.Sensitive,
	}
}
