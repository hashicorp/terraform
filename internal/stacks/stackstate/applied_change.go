// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackstate

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planproto"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate/statekeys"
	"github.com/hashicorp/terraform/internal/stacks/tfstackdata1"
	"github.com/hashicorp/terraform/internal/states"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
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

// AppliedChangeResourceInstance announces the result of applying changes to
// a particular resource instance.
type AppliedChangeResourceInstance struct {
	ResourceInstanceAddr stackaddrs.AbsResourceInstance
	NewStateSrc          *states.ResourceInstance

	// Schema MUST be the same schema that was used to encode the dynamic
	// values inside NewStateSrc.
	Schema *configschema.Block
}

var _ AppliedChange = (*AppliedChangeResourceInstance)(nil)

// AppliedChangeProto implements AppliedChange.
func (ac *AppliedChangeResourceInstance) AppliedChangeProto() (*terraform1.AppliedChange, error) {
	// FIXME: This is just a temporary stub to allow starting development of
	// RPC API clients that consume the description information. To implement
	// this fully we'll also need to emit a raw representation to save as
	// part of the raw state map, and also think a little harder about how
	// to structure the keys for the two state maps so we'll have the
	// flexibility to evolve things in future without making the client's
	// representation of the state maps become malformed.

	var descs []*terraform1.AppliedChange_ChangeDescription
	var raws []*terraform1.AppliedChange_RawChange

	moreDescs, moreRaws, err := ac.protosForObject(ac.NewStateSrc.Current, states.NotDeposed)
	if err != nil {
		return nil, fmt.Errorf("encoding current object for %s: %w", ac.ResourceInstanceAddr, err)
	}
	descs = append(descs, moreDescs...)
	raws = append(raws, moreRaws...)

	for dk, objSrc := range ac.NewStateSrc.Deposed {
		moreDescs, moreRaws, err := ac.protosForObject(objSrc, dk)
		if err != nil {
			return nil, fmt.Errorf("encoding deposed object %s for %s: %w", dk, ac.ResourceInstanceAddr, err)
		}
		descs = append(descs, moreDescs...)
		raws = append(raws, moreRaws...)
	}
	// FIXME: We also need to emit "deletion" entries for any deposed keys
	// that were present in the prior state but not present in the new state,
	// but we'll need stackeval to provide us with the prior state deposed key
	// information in order to achieve that.

	return &terraform1.AppliedChange{
		Raw:          raws,
		Descriptions: descs,
	}, nil
}

func (ac *AppliedChangeResourceInstance) protosForObject(objSrc *states.ResourceInstanceObjectSrc, deposedKey states.DeposedKey) ([]*terraform1.AppliedChange_ChangeDescription, []*terraform1.AppliedChange_RawChange, error) {
	var descs []*terraform1.AppliedChange_ChangeDescription
	var raws []*terraform1.AppliedChange_RawChange

	// For resource instance objects we use the same key format for both the
	// raw and description representations, but callers MUST NOT rely on this.
	objKey := statekeys.ResourceInstanceObject{
		ResourceInstance: ac.ResourceInstanceAddr,
		DeposedKey:       deposedKey,
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
		return nil, nil, fmt.Errorf("cannot decode new state for %s in preparation for saving it: %w", ac.ResourceInstanceAddr, err)
	}
	encValue, err := plans.NewDynamicValue(obj.Value, ty)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot encode new state for %s in preparation for saving it: %w", ac.ResourceInstanceAddr, err)
	}
	protoValue := terraform1.NewDynamicValue(encValue, objSrc.AttrSensitivePaths)

	descs = append(descs, &terraform1.AppliedChange_ChangeDescription{
		Key: objKeyRaw,
		Description: &terraform1.AppliedChange_ChangeDescription_ResourceInstance{
			ResourceInstance: &terraform1.AppliedChange_ResourceInstance{
				Addr: &terraform1.ResourceInstanceInStackAddr{
					ComponentInstanceAddr: ac.ResourceInstanceAddr.Component.String(),
					ResourceInstanceAddr:  ac.ResourceInstanceAddr.Item.String(),
				},
				NewValue: protoValue,
			},
		},
	})

	var raw anypb.Any
	err = anypb.MarshalFrom(&raw, &tfstackdata1.StateResourceInstanceObjectV1{
		Value: &planproto.DynamicValue{
			Msgpack: protoValue.Msgpack,
		},
		SensitivePaths: tfstackdata1.Terraform1ToPlanProtoAttributePaths(protoValue.Sensitive),
	}, proto.MarshalOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("encoding raw state object: %w", err)
	}
	raws = append(raws, &terraform1.AppliedChange_RawChange{
		Key:   objKeyRaw,
		Value: &raw,
	})

	return descs, raws, nil
}
