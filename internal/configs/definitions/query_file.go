// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"
)

// QueryFile represents a single query file within a configuration directory.
//
// A query file is made up of a sequential list of List blocks, each defining a
// set of filters to apply when listing a List operation.
type QueryFile struct {
	// Providers defines a set of providers that are available to the list blocks
	// within this query file.
	Providers       map[string]*Provider
	ProviderConfigs []*Provider

	Locals    []*Local
	Variables []*Variable

	// ListResources is a slice of List blocks within the query file.
	ListResources []*Resource

	VariablesDeclRange hcl.Range
}
