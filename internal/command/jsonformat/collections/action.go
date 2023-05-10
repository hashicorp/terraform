// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package collections

import "github.com/hashicorp/terraform/internal/plans"

// CompareActions will compare current and next, and return plans.Update if they
// are different, and current if they are the same.
func CompareActions(current, next plans.Action) plans.Action {
	if next == plans.NoOp {
		return current
	}

	if current != next {
		return plans.Update
	}
	return current
}
