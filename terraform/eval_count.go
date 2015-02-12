package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// EvalCountFixZeroOneBoundary is an EvalNode that fixes up the state
// when there is a resource count with zero/one boundary, i.e. fixing
// a resource named "aws_instance.foo" to "aws_instance.foo.0" and vice-versa.
type EvalCountFixZeroOneBoundary struct {
	Resource *config.Resource
}

func (n *EvalCountFixZeroOneBoundary) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

// TODO: test
func (n *EvalCountFixZeroOneBoundary) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	// Get the count, important for knowing whether we're supposed to
	// be adding the zero, or trimming it.
	count, err := n.Resource.Count()
	if err != nil {
		return nil, err
	}

	// Figure what to look for and what to replace it with
	hunt := n.Resource.Id()
	replace := hunt + ".0"
	if count < 2 {
		hunt, replace = replace, hunt
	}

	state, lock := ctx.State()

	// Get a lock so we can access this instance and potentially make
	// changes to it.
	lock.Lock()
	defer lock.Unlock()

	// Look for the module state. If we don't have one, then it doesn't matter.
	mod := state.ModuleByPath(ctx.Path())
	if mod == nil {
		return nil, nil
	}

	// Look for the resource state. If we don't have one, then it is okay.
	if rs, ok := mod.Resources[hunt]; ok {
		mod.Resources[replace] = rs
		delete(mod.Resources, hunt)
	}

	return nil, nil
}

func (n *EvalCountFixZeroOneBoundary) Type() EvalType {
	return EvalTypeNull
}
