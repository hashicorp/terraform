package stackstate

import (
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1"
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
