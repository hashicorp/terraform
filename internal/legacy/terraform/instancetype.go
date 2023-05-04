// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

//go:generate go run golang.org/x/tools/cmd/stringer -type=InstanceType instancetype.go

// InstanceType is an enum of the various types of instances store in the State
type InstanceType int

const (
	TypeInvalid InstanceType = iota
	TypePrimary
	TypeTainted
	TypeDeposed
)
