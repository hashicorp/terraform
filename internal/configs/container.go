// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package configs

import "github.com/hashicorp/terraform/internal/addrs"

// Container provides an interface for scoped resources.
//
// Any resources contained within a Container should not be accessible from
// outside the container.
type Container interface {
	// Accessible should return true if the resource specified by addr can
	// reference other items within this Container.
	//
	// Typically, that means that addr will either be the container itself or
	// something within the container.
	Accessible(addr addrs.Referenceable) bool
}
