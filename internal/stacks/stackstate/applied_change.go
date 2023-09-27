package stackstate

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
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

	// FIXME: In practice we'll need to pack more information into the keys
	// than just the naked resource instance id, since we'll need to also
	// represent other things that aren't current resource instance objects,
	// but this is sufficient for this early stub since we're not yet emitting
	// any other change types.
	tmpKey := ac.ResourceInstanceAddr.String()
	if currentObjSrc := ac.NewStateSrc.Current; currentObjSrc != nil {
		// TRICKY: For historical reasons, a states.ResourceInstance
		// contains pre-JSON-encoded dynamic data ready to be
		// inserted verbatim into Terraform CLI's traditional
		// JSON-based state file format. However, our RPC API
		// exclusively uses MessagePack encoding for dynamic
		// values, and so we will need to use the ac.Schema to
		// transcode the data.
		ty := ac.Schema.ImpliedType()
		currentObj, err := currentObjSrc.Decode(ty)
		if err != nil {
			// It would be _very_ strange to get here because we should just
			// be reversing the same encoding operation done earlier to
			// produce this object, using exactly the same schema.
			return nil, fmt.Errorf("cannot decode new state for %s in preparation for saving it: %w", ac.ResourceInstanceAddr, err)
		}
		encValue, err := plans.NewDynamicValue(currentObj.Value, ty)
		if err != nil {
			return nil, fmt.Errorf("cannot encode new state for %s in preparation for saving it: %w", ac.ResourceInstanceAddr, err)
		}
		protoValue := terraform1.NewDynamicValue(encValue, currentObjSrc.AttrSensitivePaths)

		descs = append(descs, &terraform1.AppliedChange_ChangeDescription{
			Key: tmpKey,
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
	} else {
		descs = append(descs, &terraform1.AppliedChange_ChangeDescription{
			Key: tmpKey,
			Description: &terraform1.AppliedChange_ChangeDescription_Deleted{
				Deleted: &terraform1.AppliedChange_Nothing{},
			},
		})
	}
	// TODO: Also deposed objects

	return &terraform1.AppliedChange{
		Descriptions: descs,
	}, nil
}
