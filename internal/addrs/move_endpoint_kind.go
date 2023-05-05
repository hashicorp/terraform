// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import "fmt"

// MoveEndpointKind represents the different kinds of object that a movable
// address can refer to.
type MoveEndpointKind rune

//go:generate go run golang.org/x/tools/cmd/stringer -type MoveEndpointKind

const (
	// MoveEndpointModule indicates that a move endpoint either refers to
	// an individual module instance or to all instances of a particular
	// module call.
	MoveEndpointModule MoveEndpointKind = 'M'

	// MoveEndpointResource indicates that a move endpoint either refers to
	// an individual resource instance or to all instances of a particular
	// resource.
	MoveEndpointResource MoveEndpointKind = 'R'
)

func absMoveableEndpointKind(addr AbsMoveable) MoveEndpointKind {
	switch addr := addr.(type) {
	case ModuleInstance, AbsModuleCall:
		return MoveEndpointModule
	case AbsResourceInstance, AbsResource:
		return MoveEndpointResource
	default:
		// The above should be exhaustive for all AbsMoveable types.
		panic(fmt.Sprintf("unsupported address type %T", addr))
	}
}
