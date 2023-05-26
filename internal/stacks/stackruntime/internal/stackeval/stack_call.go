package stackeval

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
