// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackstate

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
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

// AppliedChangeResourceInstanceObject announces the result of applying changes to
// a particular resource instance object.
type AppliedChangeResourceInstanceObject struct {
	ResourceInstanceObjectAddr stackaddrs.AbsResourceInstanceObject
	NewStateSrc                *states.ResourceInstanceObjectSrc
	ProviderConfigAddr         addrs.AbsProviderConfig

	// Schema MUST be the same schema that was used to encode the dynamic
	// values inside NewStateSrc. This can be left as nil if NewStateSrc
	// is nil, which represents that the object has been deleted.
	Schema *configschema.Block
}

var _ AppliedChange = (*AppliedChangeResourceInstanceObject)(nil)

// AppliedChangeProto implements AppliedChange.
func (ac *AppliedChangeResourceInstanceObject) AppliedChangeProto() (*terraform1.AppliedChange, error) {
	descs, raws, err := ac.protosForObject(ac.ResourceInstanceObjectAddr, ac.NewStateSrc)
	if err != nil {
		return nil, fmt.Errorf("encoding %s: %w", ac.ResourceInstanceObjectAddr, err)
	}
	return &terraform1.AppliedChange{
		Raw:          raws,
		Descriptions: descs,
	}, nil
}

func (ac *AppliedChangeResourceInstanceObject) protosForObject(addr stackaddrs.AbsResourceInstanceObject, objSrc *states.ResourceInstanceObjectSrc) ([]*terraform1.AppliedChange_ChangeDescription, []*terraform1.AppliedChange_RawChange, error) {
	var descs []*terraform1.AppliedChange_ChangeDescription
	var raws []*terraform1.AppliedChange_RawChange

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
	unmarkedValue, sensitivePaths := obj.Value.UnmarkDeepWithPaths()
	encValue, err := plans.NewDynamicValue(unmarkedValue, ty)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot encode new state for %s in preparation for saving it: %w", addr, err)
	}
	protoValue := terraform1.NewDynamicValue(encValue, sensitivePaths)

	descs = append(descs, &terraform1.AppliedChange_ChangeDescription{
		Key: objKeyRaw,
		Description: &terraform1.AppliedChange_ChangeDescription_ResourceInstance{
			ResourceInstance: &terraform1.AppliedChange_ResourceInstance{
				Addr:     terraform1.NewResourceInstanceObjectInStackAddr(addr),
				NewValue: protoValue,
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
	for _, key := range ac.DiscardRawKeys.Elems() {
		ret.Raw = append(ret.Raw, &terraform1.AppliedChange_RawChange{
			Key:   statekeys.String(key),
			Value: nil, // nil represents deletion
		})
	}
	for _, key := range ac.DiscardDescKeys.Elems() {
		ret.Descriptions = append(ret.Descriptions, &terraform1.AppliedChange_ChangeDescription{
			Key:         statekeys.String(key),
			Description: &terraform1.AppliedChange_ChangeDescription_Deleted{
				// Selection of this empty variant represents deletion
			},
		})
	}
	return ret, nil
}
