package stackaddrs

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

// Stack represents the address of a stack within the tree of stacks.
//
// The root stack [RootStack] represents the top-level stack and then any
// other value of this type represents an embedded stack descending from it.
type Stack []StackStep

type StackStep struct {
	Name string
}

var RootStack Stack

// IsRoot returns true if this object represents the root stack, or false
// otherwise.
func (s Stack) IsRoot() bool {
	return len(s) == 0
}

// Parent returns the parent of the reciever, or panics if the receiver is
// representing the root stack.
func (s Stack) Parent() Stack {
	newLen := len(s) - 1
	if newLen < 0 {
		panic("root stack has no parent")
	}
	return s[:newLen:newLen]
}

// Child constructs the address of an embedded stack that's a child of the
// receiver.
func (s Stack) Child(name string) Stack {
	ret := make([]StackStep, len(s), len(s)+1)
	copy(ret, s)
	return append(ret, StackStep{name})
}

// StackInstance represents the address of an instance of a stack within
// the tree of stacks.
//
// [RootStackInstance] represents the singleton instance of the top-level stack
// and then any other value of this type represents an instance of an embedded
// stack descending from it.
type StackInstance []StackInstanceStep

type StackInstanceStep struct {
	Name string
	Key  addrs.InstanceKey
}

var RootStackInstance StackInstance

// IsRoot returns true if this object represents the singleton instance of the
// root stack, or false otherwise.
func (s StackInstance) IsRoot() bool {
	return len(s) == 0
}

// Parent returns the parent of the reciever, or panics if the receiver is
// representing the root stack.
func (s StackInstance) Parent() StackInstance {
	newLen := len(s) - 1
	if newLen < 0 {
		panic("root stack has no parent")
	}
	return s[:newLen:newLen]
}

// Child constructs the address of an embedded stack that's a child of the
// receiver.
func (s StackInstance) Child(name string, key addrs.InstanceKey) StackInstance {
	ret := make([]StackInstanceStep, len(s), len(s)+1)
	copy(ret, s)
	return append(ret, StackInstanceStep{
		Name: name,
		Key:  key,
	})
}

// Call returns the address of the embedded stack call that the receiever
// belongs to, or panics if the receiver is the root module since the root
// module is called only implicitly.
func (s StackInstance) Call() AbsStackCall {
	last := s[len(s)-1]
	si := s[: len(s)-1 : len(s)-1]
	return AbsStackCall{
		Stack: si,
		Item: StackCall{
			Name: last.Name,
		},
	}
}

// ConfigAddr returns the [Stack] corresponding to the receiving [StackInstance].
func (s StackInstance) ConfigAddr() Stack {
	if s.IsRoot() {
		return RootStack
	}
	ret := make(Stack, len(s))
	for i, step := range s {
		ret[i] = StackStep{Name: step.Name}
	}
	return ret
}
