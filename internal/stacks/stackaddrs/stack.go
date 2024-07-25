// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackaddrs

import (
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
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

func (s Stack) String() string {
	if s.IsRoot() {
		// Callers should typically not ask for the string representation of
		// the main root stack, but we'll return a reasonable placeholder
		// for situations like e.g. internal logs where we just fmt %s in an
		// arbitrary stack address that is sometimes the main stack.
		return "<main>"
	}
	var buf strings.Builder
	for i, step := range s {
		if i != 0 {
			buf.WriteByte('.')
		}
		buf.WriteString("stack.")
		buf.WriteString(step.Name)
	}
	return buf.String()
}

func (s Stack) UniqueKey() collections.UniqueKey[Stack] {
	return stackUniqueKey(s.String())
}

type stackUniqueKey string

// IsUniqueKey implements collections.UniqueKey.
func (stackUniqueKey) IsUniqueKey(Stack) {}

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

func (s StackInstance) String() string {
	if s.IsRoot() {
		// Callers should typically not ask for the string representation of
		// the main root stack, but we'll return a reasonable placeholder
		// for situations like e.g. internal logs where we just fmt %s in an
		// arbitrary stack address that is sometimes the main stack.
		return "<main>"
	}
	var buf strings.Builder
	for i, step := range s {
		if i != 0 {
			buf.WriteByte('.')
		}
		buf.WriteString("stack.")
		buf.WriteString(step.Name)
		if step.Key != nil {
			buf.WriteString(step.Key.String())
		}
	}
	return buf.String()
}

func (s StackInstance) UniqueKey() collections.UniqueKey[StackInstance] {
	return stackInstanceUniqueKey(s.String())
}

type stackInstanceUniqueKey string

// IsUniqueKey implements collections.UniqueKey.
func (stackInstanceUniqueKey) IsUniqueKey(StackInstance) {}
