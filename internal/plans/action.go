// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plans

type Action rune

const (
	NoOp             Action = 0
	Create           Action = '+'
	Read             Action = '←'
	Update           Action = '~'
	DeleteThenCreate Action = '∓'
	CreateThenDelete Action = '±'
	Delete           Action = '-'
	Forget           Action = '.'
	CreateThenForget Action = '⨥'
	Open             Action = '⟃'
	Renew            Action = '⟳'
	Close            Action = '⫏'
)

//go:generate go tool golang.org/x/tools/cmd/stringer -type Action

// IsReplace returns true if the action is one of the actions that
// represent replacing an existing object with a new object.
func (a Action) IsReplace() bool {
	return a == DeleteThenCreate || a == CreateThenDelete || a == CreateThenForget
}
