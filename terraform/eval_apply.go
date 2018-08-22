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

// EvalApplyPre is an EvalNode implementation that does the pre-Apply work
type EvalApplyPre struct {
	Info  *InstanceInfo
	State **InstanceState
	Diff  **InstanceDiff
}

// TODO: test
func (n *EvalApplyPre) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State
	diff := *n.Diff

	// If the state is nil, make it non-nil
	if state == nil {
		state = new(InstanceState)
	}
	state.init()

	if resourceHasUserVisibleApply(n.Info) {
		// Call post-apply hook
		err := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PreApply(n.Info, state, diff)
		})
		if err != nil {
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

	if resourceHasUserVisibleApply(n.Info) {
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

// resourceHasUserVisibleApply returns true if the given resource is one where
// apply actions should be exposed to the user.
//
// Certain resources do apply actions only as an implementation detail, so
// these should not be advertised to code outside of this package.
func resourceHasUserVisibleApply(info *InstanceInfo) bool {
	addr := info.ResourceAddress()

	// Only managed resources have user-visible apply actions.
	// In particular, this excludes data resources since we "apply" these
	// only as an implementation detail of removing them from state when
	// they are destroyed. (When reading, they don't get here at all because
	// we present them as "Refresh" actions.)
	return addr.Mode == config.ManagedResourceMode
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

	// When is the type of provisioner to run at this point
	When config.ProvisionerWhen
}

// TODO: test
func (n *EvalApplyProvisioners) Eval(ctx EvalContext) (interface{}, error) {
	state := *n.State

	if n.CreateNew != nil && !*n.CreateNew {
		// If we're not creating a new resource, then don't run provisioners
		return nil, nil
	}

	provs := n.filterProvisioners()
	if len(provs) == 0 {
		// We have no provisioners, so don't do anything
		return nil, nil
	}

	// taint tells us whether to enable tainting.
	taint := n.When == config.ProvisionerWhenCreate

	if n.Error != nil && *n.Error != nil {
		if taint {
			state.Tainted = true
		}

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
	err := n.apply(ctx, provs)
	if err != nil {
		if taint {
			state.Tainted = true
		}

		*n.Error = multierror.Append(*n.Error, err)
		return nil, err
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

// filterProvisioners filters the provisioners on the resource to only
// the provisioners specified by the "when" option.
func (n *EvalApplyProvisioners) filterProvisioners() []*config.Provisioner {
	// Fast path the zero case
	if n.Resource == nil {
		return nil
	}

	if len(n.Resource.Provisioners) == 0 {
		return nil
	}

	result := make([]*config.Provisioner, 0, len(n.Resource.Provisioners))
	for _, p := range n.Resource.Provisioners {
		if p.When == n.When {
			result = append(result, p)
		}
	}

	return result
}

func (n *EvalApplyProvisioners) apply(ctx EvalContext, provs []*config.Provisioner) error {
	state := *n.State

	// Store the original connection info, restore later
	origConnInfo := state.Ephemeral.ConnInfo
	defer func() {
		state.Ephemeral.ConnInfo = origConnInfo
	}()

	for _, prov := range provs {
		// Get the provisioner
		provisioner := ctx.Provisioner(prov.Type)

		// Interpolate the provisioner config
		provConfig, err := ctx.Interpolate(prov.RawConfig.Copy(), n.InterpResource)
		if err != nil {
			return err
		}

		// Interpolate the conn info, since it may contain variables
		connInfo, err := ctx.Interpolate(prov.ConnInfo.Copy(), n.InterpResource)
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
		applyErr := provisioner.Apply(&output, state, provConfig)

		// Call post hook
		hookErr := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostProvision(n.Info, prov.Type, applyErr)
		})

		// Handle the error before we deal with the hook
		if applyErr != nil {
			// Determine failure behavior
			switch prov.OnFailure {
			case config.ProvisionerOnFailureContinue:
				log.Printf(
					"[INFO] apply: %s [%s]: error during provision, continue requested",
					n.Info.Id, prov.Type)

			case config.ProvisionerOnFailureFail:
				return applyErr
			}
		}

		// Deal with the hook
		if hookErr != nil {
			return hookErr
		}
	}

	return nil

}
