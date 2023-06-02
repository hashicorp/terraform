package funcs

import (
	"github.com/zclconf/go-cty/cty"
)

func refineNotNull(b *cty.RefinementBuilder) *cty.RefinementBuilder {
	return b.NotNull()
}
