// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package stackeval contains all of the internal logic of the stacks language
// runtime.
//
// This package may be imported only by the "stackruntime" package. It's a
// separate package only so we can draw a distinction between symbols that
// are exported to stackruntime vs. symbols that are private to this package,
// since there are lots of symbols in here.
//
// All functions in this package which take a [context.Context] value require
// that context to represent a task started by the package "promising", and
// may use the given task context to create promises and then wait for and/or
// resolve them. Calling with a non-task context will typically panic.
package stackeval
