package stackeval

import (
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// Removed encapsulates the somewhat complicated logic for tracking and
// managing the removed block instances in a given stack.
//
// All addresses within Removed are relative to the current stack.
type Removed struct {
	stackCallComponents collections.Map[stackaddrs.ConfigComponent, []*RemovedComponent]
	localComponents     map[stackaddrs.Component][]*RemovedComponent
}

func newRemoved(localComponents map[stackaddrs.Component][]*RemovedComponent, stackCallComponents collections.Map[stackaddrs.ConfigComponent, []*RemovedComponent]) *Removed {
	return &Removed{
		stackCallComponents: stackCallComponents,
		localComponents:     localComponents,
	}
}

// ForStackCall returns all removed component blocks that target the given
// stack call. The addresses are transformed to be relative to the stack
// created by the stack call.
func (r *Removed) ForStackCall(addr stackaddrs.StackCall) collections.Map[stackaddrs.ConfigComponent, []*RemovedComponent] {
	ret := collections.NewMap[stackaddrs.ConfigComponent, []*RemovedComponent]()
	for target, blocks := range r.stackCallComponents.All() {
		step := target.Stack[0]
		rest := target.Stack[1:]

		if step.Name != addr.Name {
			continue
		}

		ret.Put(stackaddrs.ConfigComponent{
			Stack: rest,
			Item:  target.Item,
		}, blocks)
	}
	return ret
}
