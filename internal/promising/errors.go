// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package promising

import (
	"errors"
)

// ErrUnresolved is the error returned by a promise getter if the task
// responsible for resolving the promise returns before resolving the promise.
var ErrUnresolved error

func init() {
	ErrUnresolved = errors.New("promise unresolved")
}

// ErrSelfDependent is the error type returned by a promise getter if the
// requesting task is depending on itself for its own progress, by trying
// to read a promise that it is either directly or indirectly responsible
// for resolving.
//
// The built-in error message is generic but callers can type-assert to
// this type to obtain the chain of promises that lead from the task
// to itself, possibly via other tasks that are themselves awaiting the
// caller to resolve a different promise.
type ErrSelfDependent []PromiseID

func (err ErrSelfDependent) Error() string {
	return "task is self-dependent"
}
