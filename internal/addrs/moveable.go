// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

// AbsMoveable is an interface implemented by address types that can be either
// the source or destination of a "moved" statement in configuration, along
// with any other similar cross-module state refactoring statements we might
// allow.
//
// Note that AbsMoveable represents an absolute address relative to the root
// of the configuration, which is different than the direct representation
// of these in configuration where the author gives an address relative to
// the current module where the address is defined. The type MoveEndpoint
type AbsMoveable interface {
	absMoveableSigil()
	UniqueKeyer

	String() string
}

// The following are all of the possible AbsMoveable address types:
var (
	_ AbsMoveable = AbsResource{}
	_ AbsMoveable = AbsResourceInstance{}
	_ AbsMoveable = ModuleInstance(nil)
	_ AbsMoveable = AbsModuleCall{}
)

// AbsMoveableResource is an AbsMoveable that is either a resource or a resource
// instance.
type AbsMoveableResource interface {
	AbsMoveable
	AffectedAbsResource() AbsResource
}

// The following are all of the possible AbsMoveableResource types:
var (
	_ AbsMoveableResource = AbsResource{}
	_ AbsMoveableResource = AbsResourceInstance{}
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
}

// The following are all of the possible ConfigMovable address types:
var (
	_ ConfigMoveable = ConfigResource{}
	_ ConfigMoveable = Module(nil)
)
