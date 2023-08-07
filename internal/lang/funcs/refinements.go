// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package funcs

import (
	"github.com/zclconf/go-cty/cty"
)

func refineNotNull(b *cty.RefinementBuilder) *cty.RefinementBuilder {
	return b.NotNull()
}
