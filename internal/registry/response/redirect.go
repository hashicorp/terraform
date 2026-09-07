// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package response

// Redirect causes the frontend to perform a window redirect.
type Redirect struct {
	URL string `json:"url"`
}
