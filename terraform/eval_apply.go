package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/config"
)

// EvalApply is an EvalNode implementation that writes the diff to
// the full diff.
type EvalApply struct {
	Info     *InstanceInfo
	State    **InstanceState
	Diff     **InstanceDiff
	Provider *ResourceProvider
	Output   **InstanceState
}

func (n *EvalApply) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

// TODO: test
func (n *EvalApply) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	diff := *n.Diff
	provider := *n.Provider
	state := *n.State

	// If we have no diff, we have nothing to do!
	if diff.Empty() {
		log.Printf(
			"[DEBUG] apply: %s: diff is empty, doing nothing.", n.Info.Id)
		return nil, nil
	}

	// Remove any output values from the diff
	for k, ad := range diff.Attributes {
		if ad.Type == DiffAttrOutput {
			delete(diff.Attributes, k)
		}
	}

	/*
		// Call pre-apply hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreApply(n.Info, state, diff)
		})
		if err != nil {
			return nil, err
		}
	*/

	// With the completed diff, apply!
	log.Printf("[DEBUG] apply: %s: executing Apply", n.Info.Id)
	state, err := provider.Apply(n.Info, state, diff)
	if state == nil {
		state = new(InstanceState)
	}
	state.init()

	// Force the "id" attribute to be our ID
	if state.ID != "" {
		state.Attributes["id"] = state.ID
	}

	// If the value is the unknown variable value, then it is an error.
	// In this case we record the error and remove it from the state
	for ak, av := range state.Attributes {
		if av == config.UnknownVariableValue {
			err = multierror.Append(err, fmt.Errorf(
				"Attribute with unknown value: %s", ak))
			delete(state.Attributes, ak)
		}
	}

	// Write the final state
	if n.Output != nil {
		*n.Output = state
	}

	return nil, nil
}

func (n *EvalApply) Type() EvalType {
	return EvalTypeNull
}
