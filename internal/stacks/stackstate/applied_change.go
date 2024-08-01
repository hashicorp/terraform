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
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate/statekeys"
	"github.com/hashicorp/terraform/internal/stacks/stackutils"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
	AppliedChangeProto() (*terraform1.AppliedChange, error)
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
func (ac *AppliedChangeResourceInstanceObject) AppliedChangeProto() (*terraform1.AppliedChange, error) {
	descs, raws, err := ac.protosForObject()
	if err != nil {
		return nil, fmt.Errorf("encoding %s: %w", ac.ResourceInstanceObjectAddr, err)
	}
	return &terraform1.AppliedChange{
		Raw:          raws,
		Descriptions: descs,
	}, nil
}

func (ac *AppliedChangeResourceInstanceObject) protosForObject() ([]*terraform1.AppliedChange_ChangeDescription, []*terraform1.AppliedChange_RawChange, error) {
	var descs []*terraform1.AppliedChange_ChangeDescription
	var raws []*terraform1.AppliedChange_RawChange

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
		descs = append(descs, &terraform1.AppliedChange_ChangeDescription{
			Key: objKeyRaw,
			Description: &terraform1.AppliedChange_ChangeDescription_Deleted{
				Deleted: &terraform1.AppliedChange_Nothing{},
			},
		})
		raws = append(raws, &terraform1.AppliedChange_RawChange{
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

		descs = append(descs, &terraform1.AppliedChange_ChangeDescription{
			Key: prevKeyRaw,
			Description: &terraform1.AppliedChange_ChangeDescription_Moved{
				Moved: &terraform1.AppliedChange_Nothing{},
			},
		})
		raws = append(raws, &terraform1.AppliedChange_RawChange{
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

	// Separate out sensitive marks from the decoded value so we can re-serialize it
	// with MessagePack. Sensitive paths get encoded separately in the final message.
	unmarkedValue, markses := obj.Value.UnmarkDeepWithPaths()
	sensitivePaths, otherMarkses := marks.PathsWithMark(markses, marks.Sensitive)
	if len(otherMarkses) != 0 {
		// Any other marks should've been dealt with by our caller before
		// getting here, since we only know how to preserve the sensitive
		// marking.
		return nil, nil, fmt.Errorf(
			"%s: unhandled value marks %#v (this is a bug in Terraform)",
			tfdiags.FormatCtyPath(otherMarkses[0].Path), otherMarkses[0].Marks,
		)
	}
	encValue, err := plans.NewDynamicValue(unmarkedValue, ty)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot encode new state for %s in preparation for saving it: %w", addr, err)
	}
	protoValue := terraform1.NewDynamicValue(encValue, sensitivePaths)

	descs = append(descs, &terraform1.AppliedChange_ChangeDescription{
		Key: objKeyRaw,
		Description: &terraform1.AppliedChange_ChangeDescription_ResourceInstance{
			ResourceInstance: &terraform1.AppliedChange_ResourceInstance{
				Addr:         terraform1.NewResourceInstanceObjectInStackAddr(addr),
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
	raws = append(raws, &terraform1.AppliedChange_RawChange{
		Key:   objKeyRaw,
		Value: &raw,
	})

	return descs, raws, nil
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
}

var _ AppliedChange = (*AppliedChangeComponentInstance)(nil)

// AppliedChangeProto implements AppliedChange.
func (ac *AppliedChangeComponentInstance) AppliedChangeProto() (*terraform1.AppliedChange, error) {
	ret := &terraform1.AppliedChange{
		Raw:          make([]*terraform1.AppliedChange_RawChange, 0, 1),
		Descriptions: make([]*terraform1.AppliedChange_ChangeDescription, 0, 1),
	}
	stateKey := statekeys.ComponentInstance{
		ComponentInstanceAddr: ac.ComponentInstanceAddr,
	}

	rawMsg, err := tfstackdata1.ComponentInstanceResultsToTFStackData1(ac.OutputValues)
	if err != nil {
		return nil, fmt.Errorf("encoding raw state for %s: %w", ac.ComponentInstanceAddr, err)
	}
	var raw anypb.Any
	err = anypb.MarshalFrom(&raw, rawMsg, proto.MarshalOptions{})
	if err != nil {
		return nil, fmt.Errorf("encoding raw state for %s: %w", ac.ComponentInstanceAddr, err)
	}

	outputDescs := make(map[string]*terraform1.DynamicValue, len(ac.OutputValues))
	for addr, val := range ac.OutputValues {
		unmarkedValue, markses := val.UnmarkDeepWithPaths()
		sensitivePaths, otherMarkses := marks.PathsWithMark(markses, marks.Sensitive)
		if len(otherMarkses) != 0 {
			// Any other marks should've been dealt with by our caller before
			// getting here, since we only know how to preserve the sensitive
			// marking.
			return nil, fmt.Errorf(
				"%s: unhandled value marks %#v (this is a bug in Terraform)",
				tfdiags.FormatCtyPath(otherMarkses[0].Path), otherMarkses[0].Marks,
			)
		}
		encValue, err := plans.NewDynamicValue(unmarkedValue, cty.DynamicPseudoType)
		if err != nil {
			return nil, fmt.Errorf("encoding new state for %s in %s in preparation for saving it: %w", addr, ac.ComponentInstanceAddr, err)
		}
		protoValue := terraform1.NewDynamicValue(encValue, sensitivePaths)
		outputDescs[addr.Name] = protoValue
	}

	ret.Raw = append(ret.Raw, &terraform1.AppliedChange_RawChange{
		Key:   statekeys.String(stateKey),
		Value: &raw,
	})
	ret.Descriptions = append(ret.Descriptions, &terraform1.AppliedChange_ChangeDescription{
		Key: statekeys.String(stateKey),
		Description: &terraform1.AppliedChange_ChangeDescription_ComponentInstance{
			ComponentInstance: &terraform1.AppliedChange_ComponentInstance{
				ComponentAddr:         ac.ComponentAddr.String(),
				ComponentInstanceAddr: ac.ComponentInstanceAddr.String(),
			},
		},
	})
	return ret, nil
}

type AppliedChangeDiscardKeys struct {
	DiscardRawKeys  collections.Set[statekeys.Key]
	DiscardDescKeys collections.Set[statekeys.Key]
}

var _ AppliedChange = (*AppliedChangeDiscardKeys)(nil)

// AppliedChangeProto implements AppliedChange.
func (ac *AppliedChangeDiscardKeys) AppliedChangeProto() (*terraform1.AppliedChange, error) {
	ret := &terraform1.AppliedChange{
		Raw:          make([]*terraform1.AppliedChange_RawChange, 0, ac.DiscardRawKeys.Len()),
		Descriptions: make([]*terraform1.AppliedChange_ChangeDescription, 0, ac.DiscardDescKeys.Len()),
	}
	for key := range ac.DiscardRawKeys.All() {
		ret.Raw = append(ret.Raw, &terraform1.AppliedChange_RawChange{
			Key:   statekeys.String(key),
			Value: nil, // nil represents deletion
		})
	}
	for key := range ac.DiscardDescKeys.All() {
		ret.Descriptions = append(ret.Descriptions, &terraform1.AppliedChange_ChangeDescription{
			Key:         statekeys.String(key),
			Description: &terraform1.AppliedChange_ChangeDescription_Deleted{
				// Selection of this empty variant represents deletion
			},
		})
	}
	return ret, nil
}
