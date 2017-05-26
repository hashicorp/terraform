package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// EvalValidateResourceSelfRef is an EvalNode implementation that validates that
// a configuration doesn't contain a reference to the resource itself.
//
// This must be done prior to interpolating configuration in order to avoid
// any infinite loop scenarios.
type EvalValidateResourceSelfRef struct {
	Addr   **ResourceAddress
	Config **config.RawConfig
}

func (n *EvalValidateResourceSelfRef) Eval(ctx EvalContext) (interface{}, error) {
	addr := *n.Addr
	conf := *n.Config

	// Go through the variables and find self references
	var errs []error
	for k, raw := range conf.Variables {
		rv, ok := raw.(*config.ResourceVariable)
		if !ok {
			continue
		}

		// Build an address from the variable
		varAddr := &ResourceAddress{
			Path:         addr.Path,
			Mode:         rv.Mode,
			Type:         rv.Type,
			Name:         rv.Name,
			Index:        rv.Index,
			InstanceType: TypePrimary,
		}

		// If the variable access is a multi-access (*), then we just
		// match the index so that we'll match our own addr if everything
		// else matches.
		if rv.Multi && rv.Index == -1 {
			varAddr.Index = addr.Index
		}

		// This is a weird thing where ResourceAddres has index "-1" when
		// index isn't set at all. This means index "0" for resource access.
		// So, if we have this scenario, just set our varAddr to -1 so it
		// matches.
		if addr.Index == -1 && varAddr.Index == 0 {
			varAddr.Index = -1
		}

		// If the addresses match, then this is a self reference
		if varAddr.Equals(addr) && varAddr.Index == addr.Index {
			errs = append(errs, fmt.Errorf(
				"%s: self reference not allowed: %q",
				addr, k))
		}
	}

	// If no errors, no errors!
	if len(errs) == 0 {
		return nil, nil
	}

	// Wrap the errors in the proper wrapper so we can handle validation
	// formatting properly upstream.
	return nil, &EvalValidateError{
		Errors: errs,
	}
}
