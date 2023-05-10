// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package moduletest

// A Suite is a set of tests run together as a single Terraform configuration.
type Suite struct {
	Name       string
	Components map[string]*Component
}
