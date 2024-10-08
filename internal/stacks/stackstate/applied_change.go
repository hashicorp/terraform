// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackstate

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate/statekeys"
	"github.com/hashicorp/terraform/internal/stacks/stackutils"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/internal/states"
)

// AppliedChange represents a single isolated change, emitted as
// part of a stream of applied changes during the ApplyStackChanges RPC API
// operation.
//
// Each AppliedChange becomes a single event in the RPC API, which itself
// has zero or more opaque raw plan messages that the caller must collect and
// provide verbatim during planning and zero or more "description" messages
// that are to give the caller realtime updates about the planning process.
type AppliedChange interface {
	// AppliedChangeProto returns the protocol buffers representation of
	// the change, ready to be sent verbatim to an RPC API client.
	AppliedChangeProto() (*stacks.AppliedChange, error)
}

// AppliedChangeResourceInstanceObject announces the result of applying changes to
// a particular resource instance object.
type AppliedChangeResourceInstanceObject struct {
	// ResourceInstanceObjectAddr is the absolute address of the resource
	// instance object within the component instance that declared it.
	//
	// Typically a stream of applied changes with a resource instance object
	// will also include a separate description of the component instance
	// that the resource instance belongs to, but that isn't guaranteed in
	// cases where problems occur during the apply phase and so consumers
	// should tolerate seeing a resource instance for a component instance
	// they don't know about yet, and should behave as if that component
	// instance had been previously announced.
	ResourceInstanceObjectAddr stackaddrs.AbsResourceInstanceObject
	NewStateSrc                *states.ResourceInstanceObjectSrc
	ProviderConfigAddr         addrs.AbsProviderConfig

	// PreviousResourceInstanceObjectAddr is the absolute address of the
	// resource instance object within the component instance if this object
	// was moved from another address. This will be nil if the object was not
	// moved.
	PreviousResourceInstanceObjectAddr *stackaddrs.AbsResourceInstanceObject

	// Schema MUST be the same schema that was used to encode the dynamic
	// values inside NewStateSrc. This can be left as nil if NewStateSrc
	// is nil, which represents that the object has been deleted.
	Schema *configschema.Block
}

var _ AppliedChange = (*AppliedChangeResourceInstanceObject)(nil)

// AppliedChangeProto implements AppliedChange.
func (ac *AppliedChangeResourceInstanceObject) AppliedChangeProto() (*stacks.AppliedChange, error) {
	descs, raws, err := ac.protosForObject()
	if err != nil {
		return nil, fmt.Errorf("encoding %s: %w", ac.ResourceInstanceObjectAddr, err)
	}
	return &stacks.AppliedChange{
		Raw:          raws,
		Descriptions: descs,
	}, nil
}

func (ac *AppliedChangeResourceInstanceObject) protosForObject() ([]*stacks.AppliedChange_ChangeDescription, []*stacks.AppliedChange_RawChange, error) {
	var descs []*stacks.AppliedChange_ChangeDescription
	var raws []*stacks.AppliedChange_RawChange

	var addr = ac.ResourceInstanceObjectAddr
	var provider = ac.ProviderConfigAddr
	var objSrc = ac.NewStateSrc

	// For resource instance objects we use the same key format for both the
	// raw and description representations, but callers MUST NOT rely on this.
	objKey := statekeys.ResourceInstanceObject{
		ResourceInstance: stackaddrs.AbsResourceInstance{
			Component: addr.Component,
			Item:      addr.Item.ResourceInstance,
		},
		DeposedKey: addr.Item.DeposedKey,
	}
	objKeyRaw := statekeys.String(objKey)

	if objSrc == nil {
		// If the new object is nil then we'll emit a "deleted" description
		// to ensure that any existing prior state value gets removed.
		descs = append(descs, &stacks.AppliedChange_ChangeDescription{
			Key: objKeyRaw,
			Description: &stacks.AppliedChange_ChangeDescription_Deleted{
				Deleted: &stacks.AppliedChange_Nothing{},
			},
		})
		raws = append(raws, &stacks.AppliedChange_RawChange{
			Key:   objKeyRaw,
			Value: nil, // unset Value field represents "delete" for raw changes
		})
		return descs, raws, nil
	}

	if ac.PreviousResourceInstanceObjectAddr != nil {
		// If the object was moved, we need to emit a "deleted" description
		// for the old address to ensure that any existing prior state value
		// gets removed.
		prevKey := statekeys.ResourceInstanceObject{
			ResourceInstance: stackaddrs.AbsResourceInstance{
				Component: ac.PreviousResourceInstanceObjectAddr.Component,
				Item:      ac.PreviousResourceInstanceObjectAddr.Item.ResourceInstance,
			},
			DeposedKey: ac.PreviousResourceInstanceObjectAddr.Item.DeposedKey,
		}
		prevKeyRaw := statekeys.String(prevKey)

		descs = append(descs, &stacks.AppliedChange_ChangeDescription{
			Key: prevKeyRaw,
			Description: &stacks.AppliedChange_ChangeDescription_Moved{
				Moved: &stacks.AppliedChange_Nothing{},
			},
		})
		raws = append(raws, &stacks.AppliedChange_RawChange{
			Key:   prevKeyRaw,
			Value: nil, // unset Value field represents "delete" for raw changes
		})

		// Don't return now - we'll still add the main change below.
	}

	// TRICKY: For historical reasons, a states.ResourceInstance
	// contains pre-JSON-encoded dynamic data ready to be
	// inserted verbatim into Terraform CLI's traditional
	// JSON-based state file format. However, our RPC API
	// exclusively uses MessagePack encoding for dynamic
	// values, and so we will need to use the ac.Schema to
	// transcode the data.
	ty := ac.Schema.ImpliedType()
	obj, err := objSrc.Decode(ty)
	if err != nil {
		// It would be _very_ strange to get here because we should just
		// be reversing the same encoding operation done earlier to
		// produce this object, using exactly the same schema.
		return nil, nil, fmt.Errorf("cannot decode new state for %s in preparation for saving it: %w", addr, err)
	}

	protoValue, err := stacks.ToDynamicValue(obj.Value, ty)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot encode new state for %s in preparation for saving it: %w", addr, err)
	}

	descs = append(descs, &stacks.AppliedChange_ChangeDescription{
		Key: objKeyRaw,
		Description: &stacks.AppliedChange_ChangeDescription_ResourceInstance{
			ResourceInstance: &stacks.AppliedChange_ResourceInstance{
				Addr:         stacks.NewResourceInstanceObjectInStackAddr(addr),
				NewValue:     protoValue,
				ResourceMode: stackutils.ResourceModeForProto(addr.Item.ResourceInstance.Resource.Resource.Mode),
				ResourceType: addr.Item.ResourceInstance.Resource.Resource.Type,
				ProviderAddr: provider.Provider.String(),
			},
		},
	})

	rawMsg := tfstackdata1.ResourceInstanceObjectStateToTFStackData1(objSrc, ac.ProviderConfigAddr)
	var raw anypb.Any
	err = anypb.MarshalFrom(&raw, rawMsg, proto.MarshalOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("encoding raw state object: %w", err)
	}
	raws = append(raws, &stacks.AppliedChange_RawChange{
		Key:   objKeyRaw,
		Value: &raw,
	})

	return descs, raws, nil
}

// AppliedChangeComponentInstanceRemoved is the equivalent of
// AppliedChangeComponentInstance but it represents the component instance
// being removed from state instead of created or updated.
type AppliedChangeComponentInstanceRemoved struct {
	ComponentAddr         stackaddrs.AbsComponent
	ComponentInstanceAddr stackaddrs.AbsComponentInstance
}

var _ AppliedChange = (*AppliedChangeComponentInstanceRemoved)(nil)

// AppliedChangeProto implements AppliedChange.
func (ac *AppliedChangeComponentInstanceRemoved) AppliedChangeProto() (*stacks.AppliedChange, error) {
	stateKey := statekeys.String(statekeys.ComponentInstance{
		ComponentInstanceAddr: ac.ComponentInstanceAddr,
	})
	return &stacks.AppliedChange{
		Raw: []*stacks.AppliedChange_RawChange{
			{
				Key:   stateKey,
				Value: nil,
			},
		},
		Descriptions: []*stacks.AppliedChange_ChangeDescription{
			{
				Key: stateKey,
				Description: &stacks.AppliedChange_ChangeDescription_Deleted{
					Deleted: &stacks.AppliedChange_Nothing{},
				},
			},
		},
	}, nil
}

// AppliedChangeComponentInstance announces the result of applying changes to
// an overall component instance.
//
// This deals with external-facing metadata about component instances, but
// does not directly track any resource instances inside. Those are tracked
// using individual [AppliedChangeResourceInstanceObject] objects for each.
type AppliedChangeComponentInstance struct {
	ComponentAddr         stackaddrs.AbsComponent
	ComponentInstanceAddr stackaddrs.AbsComponentInstance

	// Dependencies "remembers" the set of component instances that were
	// required by the most recent apply of this component instance.
	//
	// This will be used by the stacks runtime to determine the order in
	// which components should be destroyed when the original component block
	// is no longer available.
	Dependencies collections.Set[stackaddrs.AbsComponent]

	// Dependents "remembers" the set of component instances that depended on
	// this component instance at the most recent apply of this component
	// instance.
	//
	// This will be used by the stacks runtime to determine the order in
	// which components should be destroyed when the original component block
	// is no longer available.
	Dependents collections.Set[stackaddrs.AbsComponent]

	// OutputValues "remembers" the output values from the most recent
	// apply of the component instance. We store this primarily for external
	// consumption, since the stacks runtime is able to recalculate the
	// output values based on the prior state when needed, but we do have
	// the option of using this internally in certain special cases where it
	// would be too expensive to recalculate.
	//
	// If any output values are declared as sensitive then they should be
	// marked as such here using the usual cty marking strategy.
	OutputValues map[addrs.OutputValue]cty.Value

	// InputVariables "remembers" the input values from the most recent
	// apply of the component instance. We store this primarily for usage
	// within the removed blocks in which the input values from the last
	// applied state are required to destroy the existing resources.
	InputVariables map[addrs.InputVariable]cty.Value
}

var _ AppliedChange = (*AppliedChangeComponentInstance)(nil)

// AppliedChangeProto implements AppliedChange.
func (ac *AppliedChangeComponentInstance) AppliedChangeProto() (*stacks.AppliedChange, error) {
	stateKey := statekeys.String(statekeys.ComponentInstance{
		ComponentInstanceAddr: ac.ComponentInstanceAddr,
	})

	outputDescs := make(map[string]*stacks.DynamicValue, len(ac.OutputValues))
	for addr, val := range ac.OutputValues {
		protoValue, err := stacks.ToDynamicValue(val, cty.DynamicPseudoType)
		if err != nil {
			return nil, fmt.Errorf("encoding new state for %s in %s in preparation for saving it: %w", addr, ac.ComponentInstanceAddr, err)
		}
		outputDescs[addr.Name] = protoValue
	}

	inputDescs := make(map[string]*stacks.DynamicValue, len(ac.InputVariables))
	for addr, val := range ac.InputVariables {
		protoValue, err := stacks.ToDynamicValue(val, cty.DynamicPseudoType)
		if err != nil {
			return nil, fmt.Errorf("encoding new state for %s in %s in preparation for saving it: %w", addr, ac.ComponentInstanceAddr, err)
		}
		inputDescs[addr.Name] = protoValue
	}

	var raw anypb.Any
	if err := anypb.MarshalFrom(&raw, &tfstackdata1.StateComponentInstanceV1{
		OutputValues: func() map[string]*tfstackdata1.DynamicValue {
			outputs := make(map[string]*tfstackdata1.DynamicValue, len(outputDescs))
			for name, value := range outputDescs {
				outputs[name] = tfstackdata1.Terraform1ToStackDataDynamicValue(value)
			}
			return outputs
		}(),
		InputVariables: func() map[string]*tfstackdata1.DynamicValue {
			inputs := make(map[string]*tfstackdata1.DynamicValue, len(inputDescs))
			for name, value := range inputDescs {
				inputs[name] = tfstackdata1.Terraform1ToStackDataDynamicValue(value)
			}
			return inputs
		}(),
		DependencyAddrs: func() []string {
			var dependencies []string
			for dependency := range ac.Dependencies.All() {
				dependencies = append(dependencies, dependency.String())
			}
			return dependencies
		}(),
		DependentAddrs: func() []string {
			var dependents []string
			for dependent := range ac.Dependents.All() {
				dependents = append(dependents, dependent.String())
			}
			return dependents
		}(),
	}, proto.MarshalOptions{}); err != nil {
		return nil, fmt.Errorf("encoding raw state for %s: %w", ac.ComponentInstanceAddr, err)
	}

	return &stacks.AppliedChange{
		Raw: []*stacks.AppliedChange_RawChange{
			{
				Key:   stateKey,
				Value: &raw,
			},
		},
		Descriptions: []*stacks.AppliedChange_ChangeDescription{
			{
				Key: stateKey,
				Description: &stacks.AppliedChange_ChangeDescription_ComponentInstance{
					ComponentInstance: &stacks.AppliedChange_ComponentInstance{
						ComponentAddr:         ac.ComponentAddr.String(),
						ComponentInstanceAddr: ac.ComponentInstanceAddr.String(),
						OutputValues:          outputDescs,
					},
				},
			},
		},
	}, nil
}

type AppliedChangeInputVariable struct {
	Addr  stackaddrs.InputVariable
	Value cty.Value
}

var _ AppliedChange = (*AppliedChangeInputVariable)(nil)

func (ac *AppliedChangeInputVariable) AppliedChangeProto() (*stacks.AppliedChange, error) {
	key := statekeys.String(statekeys.Variable{
		VariableAddr: ac.Addr,
	})

	if ac.Value == cty.NilVal {
		// Then we're deleting this input variable from the state.
		return &stacks.AppliedChange{
			Raw: []*stacks.AppliedChange_RawChange{
				{
					Key:   key,
					Value: nil,
				},
			},
			Descriptions: []*stacks.AppliedChange_ChangeDescription{
				{
					Key: key,
					Description: &stacks.AppliedChange_ChangeDescription_Deleted{
						Deleted: &stacks.AppliedChange_Nothing{},
					},
				},
			},
		}, nil
	}

	var raw anypb.Any
	description := &stacks.AppliedChange_InputVariable{
		Name: ac.Addr.Name,
	}

	value, err := stacks.ToDynamicValue(ac.Value, cty.DynamicPseudoType)
	if err != nil {
		return nil, fmt.Errorf("encoding new state for %s in preparation for saving it: %w", ac.Addr, err)
	}
	description.NewValue = value
	if err := anypb.MarshalFrom(&raw, tfstackdata1.Terraform1ToStackDataDynamicValue(value), proto.MarshalOptions{}); err != nil {
		return nil, fmt.Errorf("encoding raw state for %s: %w", ac.Addr, err)
	}

	return &stacks.AppliedChange{
		Raw: []*stacks.AppliedChange_RawChange{
			{
				Key:   key,
				Value: &raw,
			},
		},
		Descriptions: []*stacks.AppliedChange_ChangeDescription{
			{
				Key: key,
				Description: &stacks.AppliedChange_ChangeDescription_InputVariable{
					InputVariable: description,
				},
			},
		},
	}, nil
}

type AppliedChangeOutputValue struct {
	Addr  stackaddrs.OutputValue
	Value cty.Value
}

var _ AppliedChange = (*AppliedChangeOutputValue)(nil)

func (ac *AppliedChangeOutputValue) AppliedChangeProto() (*stacks.AppliedChange, error) {
	key := statekeys.String(statekeys.Output{
		OutputAddr: ac.Addr,
	})

	if ac.Value == cty.NilVal {
		// Then we're deleting this output value from the state.
		return &stacks.AppliedChange{
			Raw: []*stacks.AppliedChange_RawChange{
				{
					Key:   key,
					Value: nil,
				},
			},
			Descriptions: []*stacks.AppliedChange_ChangeDescription{
				{
					Key: key,
					Description: &stacks.AppliedChange_ChangeDescription_Deleted{
						Deleted: &stacks.AppliedChange_Nothing{},
					},
				},
			},
		}, nil
	}

	value, err := stacks.ToDynamicValue(ac.Value, cty.DynamicPseudoType)
	if err != nil {
		return nil, fmt.Errorf("encoding new state for %s in preparation for saving it: %w", ac.Addr, err)
	}

	var raw anypb.Any
	if err := anypb.MarshalFrom(&raw, tfstackdata1.Terraform1ToStackDataDynamicValue(value), proto.MarshalOptions{}); err != nil {
		return nil, fmt.Errorf("encoding raw state for %s: %w", ac.Addr, err)
	}

	return &stacks.AppliedChange{
		Raw: []*stacks.AppliedChange_RawChange{
			{
				Key:   key,
				Value: &raw,
			},
		},
		Descriptions: []*stacks.AppliedChange_ChangeDescription{
			{
				Key: key,
				Description: &stacks.AppliedChange_ChangeDescription_OutputValue{
					OutputValue: &stacks.AppliedChange_OutputValue{
						Name:     ac.Addr.Name,
						NewValue: value,
					},
				},
			},
		},
	}, nil
}

type AppliedChangeDiscardKeys struct {
	DiscardRawKeys  collections.Set[statekeys.Key]
	DiscardDescKeys collections.Set[statekeys.Key]
}

var _ AppliedChange = (*AppliedChangeDiscardKeys)(nil)

// AppliedChangeProto implements AppliedChange.
func (ac *AppliedChangeDiscardKeys) AppliedChangeProto() (*stacks.AppliedChange, error) {
	ret := &stacks.AppliedChange{
		Raw:          make([]*stacks.AppliedChange_RawChange, 0, ac.DiscardRawKeys.Len()),
		Descriptions: make([]*stacks.AppliedChange_ChangeDescription, 0, ac.DiscardDescKeys.Len()),
	}
	for key := range ac.DiscardRawKeys.All() {
		ret.Raw = append(ret.Raw, &stacks.AppliedChange_RawChange{
			Key:   statekeys.String(key),
			Value: nil, // nil represents deletion
		})
	}
	for key := range ac.DiscardDescKeys.All() {
		ret.Descriptions = append(ret.Descriptions, &stacks.AppliedChange_ChangeDescription{
			Key:         statekeys.String(key),
			Description: &stacks.AppliedChange_ChangeDescription_Deleted{
				// Selection of this empty variant represents deletion
			},
		})
	}
	return ret, nil
}
