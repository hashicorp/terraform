package testharness

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"
)

// Context describes the items that are selected by a particular "describe"
// call, which can thus be referenced via variables in check functions.
//
// A Context is immutable, but derived contexts can be created using the
// methods of this type.
type Context struct {
	name     string
	resource cty.Value
	output   cty.Value
	module   cty.Value
	each     map[string]cty.Value
}

var RootContext *Context

func init() {
	RootContext = &Context{
		name: "",
		each: map[string]cty.Value{},
	}
}

// Name returns the full name of the object associated with this context.
// Some objects are _only_ represented by name, and so the name may be
// more specific than the other specific object methods would imply.
func (ctx *Context) Name() string {
	return ctx.name
}

// WithName returns a a new context that has the given string as its name.
//
// This completely replaces any existing name. Usually it's preferable to add
// a suffix to the name, preserving the context seen so far; to do this,
// use WithNameSuffix.
func (ctx *Context) WithName(name string) *Context {
	retVal := *ctx
	retVal.name = name
	return &retVal
}

// WithNameSuffix returns a new context that has the given string appended to
// the name of the receiving context.
func (ctx *Context) WithNameSuffix(suffix string) *Context {
	if ctx.name == "" {
		return ctx.WithName(suffix)
	}
	retVal := *ctx
	retVal.name = fmt.Sprintf("%s %s", ctx.name, suffix)
	return &retVal
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
