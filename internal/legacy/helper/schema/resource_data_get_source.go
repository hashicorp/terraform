// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package schema

//go:generate go tool golang.org/x/tools/cmd/stringer -type=getSource resource_data_get_source.go

// getSource represents the level we want to get for a value (internally).
// Any source less than or equal to the level will be loaded (whichever
// has a value first).
type getSource byte

const (
	getSourceState getSource = 1 << iota
	getSourceConfig
	getSourceDiff
	getSourceSet
	getSourceExact               // Only get from the _exact_ level
	getSourceLevelMask getSource = getSourceState | getSourceConfig | getSourceDiff | getSourceSet
)
