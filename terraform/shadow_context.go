package terraform

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/copystructure"
)

// newShadowContext creates a new context that will shadow the given context
// when walking the graph. The resulting context should be used _only once_
// for a graph walk.
//
// The returned Shadow should be closed after the graph walk with the
// real context is complete. Errors from the shadow can be retrieved there.
//
// Most importantly, any operations done on the shadow context (the returned
// context) will NEVER affect the real context. All structures are deep
// copied, no real providers or resources are used, etc.
func newShadowContext(c *Context) (*Context, *Context, Shadow) {
	// Copy the targets
	targetRaw, err := copystructure.Copy(c.targets)
	if err != nil {
		panic(err)
	}

	// Copy the variables
	varRaw, err := copystructure.Copy(c.variables)
	if err != nil {
		panic(err)
	}

	// The factories
	componentsReal, componentsShadow := newShadowComponentFactory(c.components)

	// Create the shadow
	shadow := &Context{
		components: componentsShadow,
		destroy:    c.destroy,
		diff:       c.diff.DeepCopy(),
		hooks:      nil, // TODO: do we need to copy? stop hook?
		module:     c.module,
		state:      c.state.DeepCopy(),
		targets:    targetRaw.([]string),
		uiInput:    nil, // TODO
		variables:  varRaw.(map[string]interface{}),

		// Hardcoded to 4 since parallelism in the shadow doesn't matter
		// a ton since we're doing far less compared to the real side
		// and our operations are MUCH faster.
		parallelSem:         NewSemaphore(4),
		providerInputConfig: make(map[string]map[string]interface{}),
	}

	// Create the real context. This is effectively just a copy of
	// the context given except we need to modify some of the values
	// to point to the real side of a shadow so the shadow can compare values.
	real := *c
	real.components = componentsReal

	return &real, shadow, &shadowContextCloser{
		Components: componentsShadow,
	}
}

// shadowContextVerify takes the real and shadow context and verifies they
// have equal diffs and states.
func shadowContextVerify(real, shadow *Context) error {
	var result error

	// Compare the states
	if !real.state.Equal(shadow.state) {
		result = multierror.Append(result, fmt.Errorf(
			"Real and shadow states do not match! "+
				"Real state:\n\n%s\n\n"+
				"Shadow state:\n\n%s\n\n",
			real.state, shadow.state))
	}

	// Compare the diffs
	if !reflect.DeepEqual(real.diff, shadow.diff) {
		result = multierror.Append(result, fmt.Errorf(
			"Real and shadow diffs do not match! "+
				"Real diff:\n\n%s\n\n"+
				"Shadow diff:\n\n%s\n\n",
			real.diff, shadow.diff))
	}

	return result
}

// shadowContextCloser is the io.Closer returned by newShadowContext that
// closes all the shadows and returns the results.
type shadowContextCloser struct {
	Components *shadowComponentFactory
}

// Close closes the shadow context.
func (c *shadowContextCloser) CloseShadow() error {
	return c.Components.CloseShadow()
}

func (c *shadowContextCloser) ShadowError() error {
	return c.Components.ShadowError()
}
