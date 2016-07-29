package terraform

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/config"
)

// EvalApply is an EvalNode implementation that writes the diff to
// the full diff.
type EvalApply struct {
	Info      *InstanceInfo
	State     **InstanceState
	Diff      **InstanceDiff
	Provider  *ResourceProvider
	Output    **InstanceState
	CreateNew *bool
	Error     *error
}

// TODO: test
func (n *EvalApply) Eval(ctx EvalContext) (interface{}, error) {
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
	for k, ad := range diff.CopyAttributes() {
		if ad.Type == DiffAttrOutput {
			diff.DelAttribute(k)
		}
	}

	// If the state is nil, make it non-nil
	if state == nil {
		state = new(InstanceState)
	}
	state.init()

	// Flag if we're creating a new instance
	if n.CreateNew != nil {
		*n.CreateNew = state.ID == "" && !diff.GetDestroy() || diff.RequiresNew()
	}

	{
		// Call pre-apply hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreApply(n.Info, state, diff)
		})
		if err != nil {
			return nil, err
		}
	}

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

	// If there are no errors, then we append it to our output error
	// if we have one, otherwise we just output it.
	if err != nil {
		if n.Error != nil {
			helpfulErr := fmt.Errorf("%s: %s", n.Info.Id, err.Error())
			*n.Error = multierror.Append(*n.Error, helpfulErr)
		} else {
			return nil, err
		}
	}

	return nil, nil
}

// EvalApplyPost is an EvalNode implementation that does the post-Apply work
type EvalApplyPost struct {
	Info  *InstanceInfo
	State **InstanceState
	Error *error
}

// TODO: test
func (n *EvalApplyPost) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State

	{
		// Call post-apply hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostApply(n.Info, state, *n.Error)
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, *n.Error
}

// EvalApplyProvisioners is an EvalNode implementation that executes
// the provisioners for a resource.
//
// TODO(mitchellh): This should probably be split up into a more fine-grained
// ApplyProvisioner (single) that is looped over.
type EvalApplyProvisioners struct {
	Info           *InstanceInfo
	State          **InstanceState
	Resource       *config.Resource
	InterpResource *Resource
	CreateNew      *bool
	Error          *error
}

// TODO: test
func (n *EvalApplyProvisioners) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State

	if !*n.CreateNew {
		// If we're not creating a new resource, then don't run provisioners
		return nil, nil
	}

	if len(n.Resource.Provisioners) == 0 {
		// We have no provisioners, so don't do anything
		return nil, nil
	}

	if n.Error != nil && *n.Error != nil {
		// We're already errored creating, so mark as tainted and continue
		state.Tainted = true

		// We're already tainted, so just return out
		return nil, nil
	}

	{
		// Call pre hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreProvisionResource(n.Info, state)
		})
		if err != nil {
			return nil, err
		}
	}

	// If there are no errors, then we append it to our output error
	// if we have one, otherwise we just output it.
	err := n.apply(ctx)
	if err != nil {
		// Provisioning failed, so mark the resource as tainted
		state.Tainted = true

		if n.Error != nil {
			*n.Error = multierror.Append(*n.Error, err)
		} else {
			return nil, err
		}
	}

	{
		// Call post hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostProvisionResource(n.Info, state)
		})
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (n *EvalApplyProvisioners) apply(ctx EvalContext) error {
	state := *n.State

	// Store the original connection info, restore later
	origConnInfo := state.Ephemeral.ConnInfo
	defer func() {
		state.Ephemeral.ConnInfo = origConnInfo
	}()

	for _, prov := range n.Resource.Provisioners {
		// Get the provisioner
		provisioner := ctx.Provisioner(prov.Type)

		// Interpolate the provisioner config
		provConfig, err := ctx.Interpolate(prov.RawConfig, n.InterpResource)
		if err != nil {
			return err
		}

		// Interpolate the conn info, since it may contain variables
		connInfo, err := ctx.Interpolate(prov.ConnInfo, n.InterpResource)
		if err != nil {
			return err
		}

		// Merge the connection information
		overlay := make(map[string]string)
		if origConnInfo != nil {
			for k, v := range origConnInfo {
				overlay[k] = v
			}
		}
		for k, v := range connInfo.Config {
			switch vt := v.(type) {
			case string:
				overlay[k] = vt
			case int64:
				overlay[k] = strconv.FormatInt(vt, 10)
			case int32:
				overlay[k] = strconv.FormatInt(int64(vt), 10)
			case int:
				overlay[k] = strconv.FormatInt(int64(vt), 10)
			case float32:
				overlay[k] = strconv.FormatFloat(float64(vt), 'f', 3, 32)
			case float64:
				overlay[k] = strconv.FormatFloat(vt, 'f', 3, 64)
			case bool:
				overlay[k] = strconv.FormatBool(vt)
			default:
				overlay[k] = fmt.Sprintf("%v", vt)
			}
		}
		state.Ephemeral.ConnInfo = overlay

		{
			// Call pre hook
			err := ctx.Hook(func(h Hook) (HookAction, error) {
				return h.PreProvision(n.Info, prov.Type)
			})
			if err != nil {
				return err
			}
		}

		// The output function
		outputFn := func(msg string) {
			ctx.Hook(func(h Hook) (HookAction, error) {
				h.ProvisionOutput(n.Info, prov.Type, msg)
				return HookActionContinue, nil
			})
		}

		// Invoke the Provisioner
		output := CallbackUIOutput{OutputFn: outputFn}
		if err := provisioner.Apply(&output, state, provConfig); err != nil {
			return err
		}

		{
			// Call post hook
			err := ctx.Hook(func(h Hook) (HookAction, error) {
				return h.PostProvision(n.Info, prov.Type)
			})
			if err != nil {
				return err
			}
		}
	}

	return nil

}
