// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ephemeral

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/lang/marks"
)

// EphemeralValuePaths returns the paths within the given value that are
// marked as ephemeral, if any.
func EphemeralValuePaths(v cty.Value) []cty.Path {
	_, pvms := v.UnmarkDeepWithPaths()
	ret, _ := marks.PathsWithMark(pvms, marks.Ephemeral)
	return ret
}
