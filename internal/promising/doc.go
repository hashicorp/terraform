// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package promising is a utility package providing a model for concurrent
// data fetching and preparation which can detect and report deadlocks
// and failure to resolve.
//
// This is based on the structure and algorithms introduced by Caleb Voss and
// Vivek Sarkar of Georgia Institute of Technology in arXiv:2101.01312v1
// "An Ownership Policy and Deadlock Detector for Promises".
//
// The model includes both promises and tasks, where tasks can wait for and
// resolve promises and each promise has a single task that is responsbile
// for resolving it. Only explicit tasks can interact with promises, and the
// system uses that rule to detect incorrect situations such as:
//
//   - Mutual dependency, where one task blocks on a promise owned by another
//     and vice-versa.
//   - Failure to resolve, where the task responsible for resolving a promise
//     completes before it does so.
//
// Mutual dependency is assumed to be the result of invalid user input where
// two objects rely on each others results, and so that situation is reported
// in a way that can allow describing the problem to an end-user.
//
// Failure to resolve is always an implementation error: a task should always
// either resolve all promises it owns or pass ownership to some other task
// before it completes.
//
// This system cannot detect situations not directly related to promise and task
// relationships. For example, if a particular task blocks forever for a
// non-promise-related reason then that can still cause an effective deadlock
// of the overall system. Callers should design their usage of tasks carefully
// so that e.g. tasks also respond to context cancellation/deadlines.
//
// Package promising uses [context.Context] values to represent dynamic task
// scope, so callers must take care to use the contexts provided to task
// functions by this package (or children of those contexts) when performing
// any task-related or promise-related actions. This implicit behavior is not
// ideal but is a pragmatic tradeoff to help keep task identity aligned with
// other cross-cutting concerns that can travel in contexts, such as loggers
// and distributed tracing clients.
//
// Internally the task-related and promise-related operations implicitly
// construct a directed bipartite graph. Between tasks and promises the
// edges represent "awaiting", and between promises and tasks the edges
// represent which task is currently responsible for resolving each promise.
// Self-dependency is therefore detected by noticing when a call to a
// [PromiseGet] would form a cycle in the graph, and immediately returning an
// error in that case to avoid deadlocking the system.
package promising

import (
	_ "context"
)
