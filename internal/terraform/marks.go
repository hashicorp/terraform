// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/zclconf/go-cty/cty"
)

// valueMarksEqual compares the marks of 2 cty.Values for equality.
func valueMarksEqual(a, b cty.Value) bool {
	_, aMarks := a.UnmarkDeepWithPaths()
	_, bMarks := b.UnmarkDeepWithPaths()
	return marks.MarksEqual(aMarks, bMarks)
}
