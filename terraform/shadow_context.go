package terraform

import (
	"io"

	"github.com/mitchellh/copystructure"
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
	// Copy the targets
	targetRaw, err := copystructure.Config{Lock: true}.Copy(c.targets)
	if err != nil {
		panic(err)
	}

	// Copy the variables
	varRaw, err := copystructure.Config{Lock: true}.Copy(c.variables)
	if err != nil {
		panic(err)
	}

	// The factories
	providerFactory := &shadowResourceProviderFactory{Original: c.providers}

	// Create the shadow
	shadow := &Context{
		destroy:      c.destroy,
		diff:         c.diff.DeepCopy(),
		hooks:        nil, // TODO: do we need to copy? stop hook?
		module:       c.module,
		providers:    providerFactory.ShadowMap(),
		provisioners: nil, //TODO
		state:        c.state.DeepCopy(),
		targets:      targetRaw.([]string),
		uiInput:      nil, // TODO
		variables:    varRaw.(map[string]interface{}),
	}

	// Create the real context. This is effectively just a copy of
	// the context given except we need to modify some of the values
	// to point to the real side of a shadow so the shadow can compare values.
	real := *c
	real.providers = providerFactory.RealMap()

	return &real, shadow, nil
}
