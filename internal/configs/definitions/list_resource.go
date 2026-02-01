// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/hcl/v2"
)

// ListResource represents the additional fields for a "list" resource mode.
type ListResource struct {
	// By default, the results of a list resource only include the identities of
	// the discovered resources. If the user specifies "include_resources = true",
	// then the provider should include the resource data in the result.
	IncludeResource hcl.Expression

	// Limit is an optional expression that can be used to limit the
	// number of results returned by the list resource.
	Limit hcl.Expression
}
