package addrs

// Debuggable is an interface implemented by all address types representing
// objects that can be "debugged", which in this context means that it can
// have a breakpoint (or similar) set on it, and other similar ideas that
// arise when mapping Terraform's execution model to the
// Debug Adapter Protocol.
type Debuggable interface {
	debuggableSigil()

	// DebugAncestorFrames returns a series of Debuggable addresses representing
	// ancestors of the reciever that, for the purpose of presenting a
	// debugging model to end-users, we consider to be a sequence of "callers"
	// that appear as a call stack. The first elements of the result are
	// deeper in the stack, and so the final element of the result is the
	// direct caller of the reciever.
	//
	// For a top-level debuggable object (one that has no callers as far as
	// our debug model is concerned), AncestorFrames returns an empty slice.
	DebugAncestorFrames() []Debuggable

	// String produces a string representation of the address that is
	// primarily for display in the debugger UI. It might not necessarily
	// be stable identifier across multiple debug runs, depending on the
	// particular address type.
	String() string
}

type debuggable struct {
}

func (a debuggable) debuggableSigil() {
}

// parentDebugAncestorFrames is a helper for implementing DebugAncestorFrames
// on non-root Debuggable implementations in terms of the parent.
func parentDebugAncestorFrames(parent Debuggable) []Debuggable {
	ancestors := parent.DebugAncestorFrames()
	// It's safe for us to append to ancestors here because our
	// DebugAncestorFrames methods all construct fresh slices on each
	// call, either directly or indirectly.
	return append(ancestors, parent)
}

// DebugStackTrace returns a user-oriented "stack trace" for the given
// debuggable address.
//
// The result will always have at least one element, and the final element
// is always the given address.
//
// Although the Terraform runtime isn't actually stack-oriented, in order to
// present a user model compatible with typical debugger frontends we behave
// as if both module calls and instantiation of module calls and of resources
// were function calls, and thus allow users to use the resulting virtual
// stack trace to see which specific instance of each module call and
// resource are currently active.
func DebugStackTrace(current Debuggable) []Debuggable {
	// This ends up being the same as our logic for finding the "parent"
	// frames.
	return parentDebugAncestorFrames(current)
}
