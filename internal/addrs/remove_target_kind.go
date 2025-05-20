// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import "fmt"

// RemoveTargetKind represents the different kinds of object that a remove
// target address can refer to.
type RemoveTargetKind rune

//go:generate go tool golang.org/x/tools/cmd/stringer -type RemoveTargetKind

const (
	// RemoveTargetModule indicates that a remove target refers to
	// all instances of a particular module call.
	RemoveTargetModule RemoveTargetKind = 'M'

	// RemoveTargetResource indicates that a remove target refers to
	// all instances of a particular resource.
	RemoveTargetResource RemoveTargetKind = 'R'
)

func removeTargetKind(addr ConfigMoveable) RemoveTargetKind {
	switch addr := addr.(type) {
	case Module:
		return RemoveTargetModule
	case ConfigResource:
		return RemoveTargetResource
	default:
		// The above should be exhaustive for all ConfigMoveable types.
		panic(fmt.Sprintf("unsupported address type %T", addr))
	}
}
