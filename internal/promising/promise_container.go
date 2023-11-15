// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package promising

// PromiseContainer is an interface implemented by types whose behavior
// is implemented in terms of at least one promise, which therefore allows
// the responsibility for resolving those promises to be moved from
// one task to another.
//
// All promises in a single container must have the same task responsible
// for resolving them.
type PromiseContainer interface {
	// AnnounceContainedPromises calls the given callback exactly once for
	// each promise that the receiver is implemented in terms of.
	AnnounceContainedPromises(func(AnyPromiseResolver))
}

var NoPromises PromiseContainer

type noPromises struct{}

func (noPromises) AnnounceContainedPromises(func(AnyPromiseResolver)) {
	// Nothing to announce.
}

func init() {
	NoPromises = noPromises{}
}

// PromiseResolverPair is a convenience [PromiseContainer] for passing a
// pair of promise resolvers of different result types to a child task without
// having to create a custom struct type to do it.
//
// This is a shortcut for simpler cases. If you need something more elaborate
// than this, write your own implementation of [PromiseContainer].
type PromiseResolverPair[AType, BType any] struct {
	A PromiseResolver[AType]
	B PromiseResolver[BType]
}

func (pair PromiseResolverPair[AType, BType]) AnnounceContainedPromises(cb func(AnyPromiseResolver)) {
	cb(pair.A)
	cb(pair.B)
}

// PromiseResolverList is a convenience [PromiseContainer] for passing an
// arbitrary number of promise resolvers of the same result type to a child
// task without having to create a custom struct type to do it.
//
// Go's type system does not support variadic generics so we cannot provide
// a single type that collects an arbitrary number of resolvers with different
// result types. If you need that then you must write your own struct type
// which implements [PromiseContainer], or alternatively use
// [PromiseResolverPair] if you happen to have exactly two resolvers to pass.
type PromiseResolverList[T any] []PromiseResolver[T]

func (l PromiseResolverList[T]) AnnounceContainedPromises(cb func(AnyPromiseResolver)) {
	for _, r := range l {
		cb(r)
	}
}
