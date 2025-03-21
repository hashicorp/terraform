// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"fmt"
	"sync"

	"github.com/hashicorp/go-slug/sourcebundle"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/stacks/stackconfig"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
)

// handle represents an identifier shared between client and server to identify
// a particular object.
//
// From the server's perspective, these are always issued by and tracked within
// a [handleTable]. The conventional variable name for a handle in this codebase
// is "hnd", or a name with "Hnd" as a suffix such as "planHnd".
type handle[T any] int64

// ForProtobuf returns the handle as a naked int64 for use in a protocol buffers
// message. This erases the information about what type of object this handle
// was for, so should be used only when preparing a protobuf response and not
// for anything internal to this package.
func (hnd handle[T]) ForProtobuf() int64 {
	return int64(hnd)
}

// IsNil returns true if the reciever is the "nil handle", which is also the
// zero value of any handle type and represents the absense of a handle.
func (hnd handle[T]) IsNil() bool {
	return int64(hnd) == 0
}

// handleTable is our shared table of "handles", which are really just integers
// that clients can use to refer to an object they've previously opened.
//
// In our public API contract each different object has a separate numberspace
// of handles, but as an implementation detail we share a single numberspace
// for everything just because that means that if a client gets their handles
// mixed up then it'll fail with an error rather than potentially doing
// something unexpected to an unrelated object, since these all appear as the
// same type in the protobuf-generated API.
//
// The handleTable API intentionally requires callers to be explicit about
// what kind of handle they are intending to work with. There is no function
// to query which kind of object is associated with a handle, because if
// a particular operation can accept handles of multiple types it should
// separate those into separate request fields and decide how to interpret
// the handles based on which fields are populated by the caller.
type handleTable struct {
	handleObjs map[int64]any
	nextHandle int64

	// handleDeps tracks dependencies between handles to disallow closing
	// a handle that has another open handle depending on it. For example,
	// each stack configuration handle depends on a source bundle handle
	// because closing the source bundle and deleting its underlying directory
	// would cause unwanted misbehavior for future operations against that
	// stack configuration.
	//
	// The first level of map is the object being depended on, and the second
	// level are the objects doing the depending.
	handleDeps map[int64]map[int64]struct{}

	// TODO: Consider also tracking when a particular handle is being actively
	// used by a running RPC operation, so that we can return an error if
	// a caller tries to close a handle concurrently with an active operation.
	// That would be a weird thing to do though and always a bug in the caller,
	// so for now we're just letting it cause unspecified behavior for
	// simplicity's sake.

	mu sync.Mutex
}

func newHandleTable() *handleTable {
	return &handleTable{
		handleObjs: make(map[int64]any),
		handleDeps: make(map[int64]map[int64]struct{}),
		nextHandle: 1,
	}
}

func (t *handleTable) NewSourceBundle(sources *sourcebundle.Bundle) handle[*sourcebundle.Bundle] {
	return newHandle(t, sources)
}

func (t *handleTable) SourceBundle(hnd handle[*sourcebundle.Bundle]) *sourcebundle.Bundle {
	ret, _ := readHandle(t, hnd) // non-existent or invalid returns nil
	return ret
}

func (t *handleTable) CloseSourceBundle(hnd handle[*sourcebundle.Bundle]) error {
	return closeHandle(t, hnd)
}

func (t *handleTable) NewStackConfig(cfg *stackconfig.Config, owningSourceBundle handle[*sourcebundle.Bundle]) (handle[*stackconfig.Config], error) {
	return newHandleWithDependency(t, cfg, owningSourceBundle)
}

func (t *handleTable) StackConfig(hnd handle[*stackconfig.Config]) *stackconfig.Config {
	ret, _ := readHandle(t, hnd) // non-existent or invalid returns nil
	return ret
}

func (t *handleTable) CloseStackConfig(hnd handle[*stackconfig.Config]) error {
	return closeHandle(t, hnd)
}

func (t *handleTable) NewStackState(state *stackstate.State) handle[*stackstate.State] {
	return newHandle(t, state)
}

func (t *handleTable) StackState(hnd handle[*stackstate.State]) *stackstate.State {
	ret, _ := readHandle(t, hnd) // non-existent or invalid returns nil
	return ret
}

func (t *handleTable) CloseStackState(hnd handle[*stackstate.State]) error {
	return closeHandle(t, hnd)
}

func (t *handleTable) NewStackPlan(state *stackplan.Plan) handle[*stackplan.Plan] {
	return newHandle(t, state)
}

func (t *handleTable) StackPlan(hnd handle[*stackplan.Plan]) *stackplan.Plan {
	ret, _ := readHandle(t, hnd) // non-existent or invalid returns nil
	return ret
}

func (t *handleTable) CloseStackPlan(hnd handle[*stackplan.Plan]) error {
	return closeHandle(t, hnd)
}

func (t *handleTable) NewDependencyLocks(locks *depsfile.Locks) handle[*depsfile.Locks] {
	// NOTE: We intentionally don't track a dependency on a source bundle
	// here for two reasons:
	// - Not all lock objects necessarily original from lock files. For example,
	//   we could be creating a new empty locks that will be mutated and then
	//   saved to disk for the first time afterwards.
	// - The locks object in memory is not dependent on the lock file it was
	//   loaded from once the load is complete. Closing the source bundle and
	//   deleting its directory would not affect the validity of the locks
	//   object.
	return newHandle(t, locks)
}

func (t *handleTable) DependencyLocks(hnd handle[*depsfile.Locks]) *depsfile.Locks {
	ret, _ := readHandle(t, hnd) // non-existent or invalid returns nil
	return ret
}

func (t *handleTable) CloseDependencyLocks(hnd handle[*depsfile.Locks]) error {
	return closeHandle(t, hnd)
}

func (t *handleTable) NewProviderPluginCache(dir *providercache.Dir) handle[*providercache.Dir] {
	return newHandle(t, dir)
}

func (t *handleTable) ProviderPluginCache(hnd handle[*providercache.Dir]) *providercache.Dir {
	ret, _ := readHandle(t, hnd) // non-existent or invalid returns nil
	return ret
}

func (t *handleTable) CloseProviderPluginCache(hnd handle[*providercache.Dir]) error {
	return closeHandle(t, hnd)
}

func (t *handleTable) NewStackInspector(dir *stacksInspector) handle[*stacksInspector] {
	return newHandle(t, dir)
}

func (t *handleTable) StackInspector(hnd handle[*stacksInspector]) *stacksInspector {
	ret, _ := readHandle(t, hnd) // non-existent or invalid returns nil
	return ret
}

func (t *handleTable) CloseStackInspector(hnd handle[*stacksInspector]) error {
	return closeHandle(t, hnd)
}

func newHandle[ObjT any](t *handleTable, obj ObjT) handle[ObjT] {
	t.mu.Lock()
	hnd := t.nextHandle
	t.nextHandle++ // NOTE: We're assuming int64 is big enough for overflow to be impractical
	t.handleObjs[hnd] = obj
	t.mu.Unlock()
	return handle[ObjT](hnd)
}

// newHandleWithDependency is a variant of newHandle which also records a
// dependency on some other handle.
//
// Unlike newHandle, this is fallible because creating the new handle might
// race with closing the dependency handle, causing that handle to no longer
// be available by the time this function is running. In that case, the
// returned error will be [newHandleErrorNoParent].
func newHandleWithDependency[ObjT, DepT any](t *handleTable, obj ObjT, dep handle[DepT]) (handle[ObjT], error) {
	t.mu.Lock()
	if depObjectI, exists := t.handleObjs[int64(dep)]; !exists {
		return handle[ObjT](0), newHandleErrorNoParent
	} else if depObject, ok := depObjectI.(DepT); !ok {
		// It's caller's responsibility to ensure that it's passing in valid handles.
		// (This will typically be ensured by our type-safe wrapper methods)
		panic(fmt.Sprintf("dependency handle %d is %T, not %T", int64(dep), depObjectI, depObject))
	}
	hnd := t.nextHandle
	t.nextHandle++ // NOTE: We're assuming int64 is big enough for overflow to be impractical
	t.handleObjs[hnd] = obj
	if _, exists := t.handleDeps[int64(dep)]; !exists {
		t.handleDeps[int64(dep)] = make(map[int64]struct{})
	}
	t.handleDeps[int64(dep)][hnd] = struct{}{}
	t.mu.Unlock()
	return handle[ObjT](hnd), nil
}

func readHandle[ObjT any](t *handleTable, hnd handle[ObjT]) (ObjT, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	var zero ObjT
	if existing, exists := t.handleObjs[int64(hnd)]; !exists {
		return zero, false
	} else if existing, ok := existing.(ObjT); !ok {
		return zero, false
	} else {
		return existing, true
	}
}

func closeHandle[ObjT any](t *handleTable, hnd handle[ObjT]) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if existing, exists := t.handleObjs[int64(hnd)]; !exists {
		return closeHandleErrorUnknown
	} else if _, ok := existing.(ObjT); !ok {
		return closeHandleErrorUnknown
	}
	if len(t.handleDeps[int64(hnd)]) > 0 {
		return closeHandleErrorBlocked
	}
	delete(t.handleObjs, int64(hnd))
	// We'll also revoke this object's dependencies so that they
	// can potentially be closed after we return. Our dependency-tracking
	// data structure is not optimized for deleting because that's rare
	// in comparison to adding and checking, but we expect the handle
	// table to typically be small enough for this full scan not to hurt.
	for _, m := range t.handleDeps {
		delete(m, int64(hnd)) // no-op if not present
	}
	return nil
}

type newHandleError rune

const (
	newHandleErrorNoParent newHandleError = '^'
)

func (err newHandleError) Error() string {
	switch err {
	case newHandleErrorNoParent:
		return "parent handle does not exist"
	default:
		return "unknown error creating handle"
	}
}

type closeHandleError rune

const (
	closeHandleErrorUnknown closeHandleError = '?'
	closeHandleErrorBlocked closeHandleError = 'B'
)

func (err closeHandleError) Error() string {
	switch err {
	case closeHandleErrorUnknown:
		return "unknown handle"
	case closeHandleErrorBlocked:
		return "handle is in use by another open handle"
	default:
		return "unknown error closing handle"
	}
}
