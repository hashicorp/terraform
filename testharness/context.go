package testharness

import (
	"github.com/zclconf/go-cty/cty"
)

// Context describes the items that are selected by a particular "describe"
// call, which can thus be referenced via variables in check functions.
//
// A Context is immutable, but derived contexts can be created using the
// methods of this type.
type Context struct {
	resource cty.Value
	output   cty.Value
	module   cty.Value
	each     map[string]cty.Value
}

// HasResource returns true if there is a resource object associated with
// this context.
func (ctx *Context) HasResource() bool {
	return ctx.resource != cty.NilVal
}

// Resource returns the resource object associated with this context, or
// cty.NilVal if the context has no associated resource.
func (ctx *Context) Resource() cty.Value {
	return ctx.resource
}

// WithResource returns a new context which has the given resource object
// associated.
func (ctx *Context) WithResource(obj cty.Value) *Context {
	retVal := *ctx
	retVal.resource = obj
	return &retVal
}

// HasOutput returns true if there is an output object associated with
// this context.
func (ctx *Context) HasOutput() bool {
	return ctx.output != cty.NilVal
}

// Output returns the output object associated with this context, or
// cty.NilVal if the context has no associated output.
func (ctx *Context) Output() cty.Value {
	return ctx.output
}

// WithOutput returns a new context which has the given output object
// associated.
func (ctx *Context) WithOutput(obj cty.Value) *Context {
	retVal := *ctx
	retVal.output = obj
	return &retVal
}

// HasModule returns true if there is a module object associated with
// this context.
func (ctx *Context) HasModule() bool {
	return ctx.module != cty.NilVal
}

// Module returns the module object associated with this context, or
// cty.NilVal if the context has no associated module.
func (ctx *Context) Module() cty.Value {
	return ctx.module
}

// WithModule returns a new context which has the given module object
// associated.
func (ctx *Context) WithModule(obj cty.Value) *Context {
	retVal := *ctx
	retVal.module = obj
	return &retVal
}

// HasEach returns true if there is an "each" value of the given name associated
// with this context.
func (ctx *Context) HasEach(name string) bool {
	return ctx.each[name] != cty.NilVal
}

// Each returns the "each" value with the given name that is associated with
// this context, or cty.NilVal if the context has no such "each" value.
func (ctx *Context) Each(name string) cty.Value {
	return ctx.each[name]
}

// WithEach returns a new context which has the given each value
// associated with the given name.
func (ctx *Context) WithEach(name string, val cty.Value) *Context {
	retVal := *ctx
	newEach := make(map[string]cty.Value, len(ctx.each)+1)
	for k, v := range ctx.each {
		newEach[k] = v
	}
	newEach[name] = val
	retVal.each = newEach
	return &retVal
}

// EachObject returns a cty value of an object type representing all of the
// "each" values associated with this context, with their names as the
// object type attributes.
func (ctx *Context) EachObject() cty.Value {
	if len(ctx.each) == 0 {
		return cty.EmptyObjectVal
	}
	return cty.ObjectVal(ctx.each)
}
