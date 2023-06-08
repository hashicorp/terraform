package stackeval

//lint:file-ignore U1000 This package is still WIP so not everything is here yet.

import (
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// InputVariable represents an input variable belonging to a [Stack].
type InputVariable struct {
	addr stackaddrs.AbsInputVariable

	main *Main
}

func (v *InputVariable) Addr() stackaddrs.AbsInputVariable {
	return v.addr
}
