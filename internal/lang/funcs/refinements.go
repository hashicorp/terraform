// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package funcs

import (
	"github.com/zclconf/go-cty/cty"
)

func refineNotNull(b *cty.RefinementBuilder) *cty.RefinementBuilder {
	return b.NotNull()
}
