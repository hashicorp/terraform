package stackstate

import (
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
	if ac.NewStateSrc.Current != nil {
		descs = append(descs, &terraform1.AppliedChange_ChangeDescription{
			Key: tmpKey,
			Description: &terraform1.AppliedChange_ChangeDescription_ResourceInstance{
				ResourceInstance: &terraform1.AppliedChange_ResourceInstance{
					Addr: &terraform1.ResourceInstanceInStackAddr{
						ComponentInstanceAddr: ac.ResourceInstanceAddr.Component.String(),
						ResourceInstanceAddr:  ac.ResourceInstanceAddr.Item.String(),
					},

					// FIXME: The NewStateSrc values are serialized as JSON
					// for inclusion in traditional Terraform's JSON state
					// format, but we want MessagePack here for consistency
					// with the rest of the RPC API protocol. However, we
					// can't convert between the two without access to the
					// provider schema. We'll need to handle that conversion
					// upstream somewhere. For now we're just always returning
					// an empty object here as placeholder, which is very wrong
					// but is at least something we can use for early
					// development of client code concurrently with working on
					// the rest of this.
					NewValue: &terraform1.DynamicValue{
						Msgpack: []byte{
							0b1000_0000, // MessagePack coding of a zero-length "fixmap"
						},
					},
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
