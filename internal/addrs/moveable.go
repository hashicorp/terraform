package addrs

import "fmt"

// AbsMoveable is an interface implemented by address types that can be either
// the source or destination of a "moved" statement in configuration, along
// with any other similar cross-module state refactoring statements we might
// allow.
//
// Note that AbsMovable represents an absolute address relative to the root
// of the configuration, which is different than the direct representation
// of these in configuration where the author gives an address relative to
// the current module where the address is defined. The type MoveEndpoint

type AbsMoveable interface {
	absMoveableSigil()

	String() string
}

// The following are all of the possible AbsMovable address types:
var (
	_ AbsMoveable = AbsResource{}
	_ AbsMoveable = AbsResourceInstance{}
	_ AbsMoveable = ModuleInstance(nil)
	_ AbsMoveable = AbsModuleCall{}
)

// ConfigMoveable is similar to AbsMoveable but represents a static object in
// the configuration, rather than an instance of that object created by
// module expansion.
//
// Note that ConfigMovable represents an absolute address relative to the root
// of the configuration, which is different than the direct representation
// of these in configuration where the author gives an address relative to
// the current module where the address is defined. The type MoveEndpoint
// represents the relative form given directly in configuration.
type ConfigMoveable interface {
	configMoveableSigil()

	String() string
}

// The following are all of the possible ConfigMovable address types:
var (
	_ ConfigMoveable = ConfigResource{}
	_ ConfigMoveable = Module(nil)
)

func (r ConfigResource) IncludedInMoveable(moveable ConfigMoveable) bool {
	switch moveable := moveable.(type) {
	case ConfigResource:
		return r.Equal(moveable)
	case Module:
		// A resource is included in a module if the resource's module
		// address is a prefix of the given module.
		modAddr := r.Module
		if len(modAddr) < len(moveable) {
			return false // can't possibly be a prefix then
		}
		modAddr = modAddr[:len(moveable)]
		return modAddr.Equal(moveable)
	default:
		// The above cases should include all implementations of ConfigMoveable
		panic(fmt.Sprintf("unhandled ConfigMovable type %T", moveable))
	}
}

func (r Module) IncludedInMoveable(moveable ConfigMoveable) bool {
	switch moveable := moveable.(type) {
	case ConfigResource:
		// A whole module can never be selected by a resource address
		return false
	case Module:
		// The receiver is included in moveable if moveable is a prefix of it.
		if len(r) < len(moveable) {
			return false // can't possibly be a prefix then
		}
		modAddr := r[:len(moveable)]
		return modAddr.Equal(moveable)
	default:
		// The above cases should include all implementations of ConfigMoveable
		panic(fmt.Sprintf("unhandled ConfigMovable type %T", moveable))
	}
}
