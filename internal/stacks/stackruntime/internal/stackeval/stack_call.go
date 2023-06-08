package stackeval

//lint:file-ignore U1000 This package is still WIP so not everything is here yet.

import (
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// StackCall represents a "stack" block in a stack configuration after
// its containing stacks have been expanded into stack instances.
type StackCall struct {
	addr stackaddrs.AbsStackCall

	main *Main
}

func (c *StackCall) Addr() stackaddrs.AbsStackCall {
	return c.addr
}
