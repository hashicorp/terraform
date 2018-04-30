package terraform

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalPreventDestroy is an EvalNode implementation that returns an
// error if a resource has PreventDestroy configured and the diff
// would destroy the resource.
type EvalCheckPreventDestroy struct {
	Addr   addrs.ResourceInstance
	Config *configs.Resource
	Diff   **InstanceDiff
}

func (n *EvalCheckPreventDestroy) Eval(ctx EvalContext) (interface{}, error) {
	if n.Diff == nil || *n.Diff == nil || n.Config == nil || n.Config.Managed == nil {
		return nil, nil
	}

	diff := *n.Diff
	preventDestroy := n.Config.Managed.PreventDestroy

	if diff.GetDestroy() && preventDestroy {
		var diags tfdiags.Diagnostics
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Instance cannot be destroyed",
			Detail: fmt.Sprintf(
				"Resource %s has lifecycle.prevent_destroy set, but the plan calls for this resource to be destroyed. To avoid this error and continue with the plan, either disable lifecycle.prevent_destroy or reduce the scope of the plan using the -target flag.",
				n.Addr.Absolute(ctx.Path()).String(),
			),
			Subject: &n.Config.DeclRange,
		})
		return nil, diags.Err()
	}

	return nil, nil
}

const preventDestroyErrStr = `%s: the plan would destroy this resource, but it currently has lifecycle.prevent_destroy set to true. To avoid this error and continue with the plan, either disable lifecycle.prevent_destroy or adjust the scope of the plan using the -target flag.`
