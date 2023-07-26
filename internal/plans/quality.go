// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plans

// Quality represents facts about the nature of a plan that might be relevant
// when rendering it, like whether it errored or contains no changes. A plan can
// have multiple qualities.
type Quality int

//go:generate go run golang.org/x/tools/cmd/stringer -type Quality

const (
	// Errored plans did not successfully complete, and cannot be applied.
	Errored Quality = iota
	// NoChanges plans won't result in any actions on infrastructure, or any
	// semantically meaningful updates to state. They can sometimes still affect
	// the format of state if applied.
	NoChanges
)
