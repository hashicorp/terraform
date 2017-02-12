package terraform

// Shadow is the interface that any "shadow" structures must implement.
//
// A shadow structure is an interface implementation (typically) that
// shadows a real implementation and verifies that the same behavior occurs
// on both. The semantics of this behavior are up to the interface itself.
//
// A shadow NEVER modifies real values or state. It must always be safe to use.
//
// For example, a ResourceProvider shadow ensures that the same operations
// are done on the same resources with the same configurations.
//
// The typical usage of a shadow following this interface is to complete
// the real operations, then call CloseShadow which tells the shadow that
// the real side is done. Then, once the shadow is also complete, call
// ShadowError to find any errors that may have been caught.
type Shadow interface {
	// CloseShadow tells the shadow that the REAL implementation is
	// complete. Therefore, any calls that would block should now return
	// immediately since no more changes will happen to the real side.
	CloseShadow() error

	// ShadowError returns the errors that the shadow has found.
	// This should be called AFTER CloseShadow and AFTER the shadow is
	// known to be complete (no more calls to it).
	ShadowError() error
}
