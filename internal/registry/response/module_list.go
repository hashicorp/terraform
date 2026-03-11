// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package response

// ModuleList is the response structure for a pageable list of modules.
type ModuleList struct {
	Meta    PaginationMeta `json:"meta"`
	Modules []*Module      `json:"modules"`
}
