package terraform

import (
	"io"
)

// newShadowContext creates a new context that will shadow the given context
// when walking the graph. The resulting context should be used _only once_
// for a graph walk.
//
// The returned io.Closer should be closed after the graph walk with the
// real context is complete. The result of the Close function will be any
// errors caught during the shadowing operation.
//
// Most importantly, any operations done on the shadow context (the returned
// context) will NEVER affect the real context. All structures are deep
// copied, no real providers or resources are used, etc.
func newShadowContext(c *Context) (*Context, *Context, io.Closer) {
	return c, nil, nil
}
