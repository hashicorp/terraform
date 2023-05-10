// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package response

// Redirect causes the frontend to perform a window redirect.
type Redirect struct {
	URL string `json:"url"`
}
