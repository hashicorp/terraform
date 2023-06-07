package rpcapi

import (
	"sync"

	"github.com/hashicorp/go-slug/sourcebundle"
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

	mu sync.Mutex
}

func newHandleTable() *handleTable {
	return &handleTable{
		handleObjs: make(map[int64]any),
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

func (t *handleTable) CloseSourceBundle(hnd handle[*sourcebundle.Bundle]) bool {
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

func closeHandle[ObjT any](t *handleTable, hnd handle[ObjT]) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if existing, exists := t.handleObjs[int64(hnd)]; !exists {
		return false
	} else if _, ok := existing.(ObjT); !ok {
		return false
	}
	delete(t.handleObjs, int64(hnd))
	return true
}
