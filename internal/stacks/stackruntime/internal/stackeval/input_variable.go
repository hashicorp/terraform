package stackeval

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
