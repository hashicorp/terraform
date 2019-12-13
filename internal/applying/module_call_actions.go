package applying

import (
	"context"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

// moduleCallActions collects together all of the actions for a particular
// module call.
type moduleCallActions struct {
	// ContainingAddr is the address of the module instance that contains the
	// call, while CallAddr is the local address of the call itself, unique
	// only within the module instance identified by ContainingAddr.
	ContainingAddr addrs.ModuleInstance
	CallAddr       addrs.ModuleCall

	// Expand is the action responsible for deciding the expansion of the
	// module based on its count or for_each argument.
	Expand *moduleCallExpandAction

	// Dependencies describes the values required to evaluate the for_each
	// or count expressions, if present, and anything specified explicitly
	// in depends_on. It does _not_ include dependencies of any individual
	// variables; they each track their own separately.
	Dependencies []addrs.Referenceable
}

type moduleCallExpandAction struct {
	ContainingAddr addrs.ModuleInstance
	CallAddr       addrs.ModuleCall

	ForEach hcl.Expression
	Count   hcl.Expression
}

func (a *moduleCallExpandAction) Name() string {
	if a.ContainingAddr.IsRoot() {
		return "Expand " + a.CallAddr.String()
	}
	return "Expand " + a.ContainingAddr.String() + "." + a.CallAddr.String()
}

func (a *moduleCallExpandAction) Execute(ctx context.Context, data *actionData) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Module call expansion is not yet implemented",
		"The prototype apply codepath cannot expand module calls",
	))

	return diags
}
