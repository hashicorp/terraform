package statemgr

// Full is the union of all of the more-specific state interfaces.
//
// This interface may grow over time, so state implementations aiming to
// implement it may need to be modified for future changes. To ensure that
// this need can be detected, always include a statement nearby the declaration
// of the implementing type that will fail at compile time if the interface
// isn't satisfied, such as:
//
//     var _ statemgr.Full = (*ImplementingType)(nil)
type Full interface {
	Transient
	Persistent
	Locker
}
