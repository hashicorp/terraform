package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// EvalValidateError is the error structure returned if there were
// validation errors.
type EvalValidateError struct {
	Warnings []string
	Errors   []error
}

func (e *EvalValidateError) Error() string {
	return ""
}

// EvalValidateResource is an EvalNode implementation that validates
// the configuration of a resource.
type EvalValidateResource struct {
	Provider     EvalNode
	Config       *config.RawConfig
	ProviderType string
}

func (n *EvalValidateResource) Args() ([]EvalNode, []EvalType) {
	return []EvalNode{n.Provider},
		[]EvalType{EvalTypeResourceProvider}
}

func (n *EvalValidateResource) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	// TODO: test

	//provider := args[0].(ResourceProvider)
	return nil, nil
}

func (n *EvalValidateResource) Type() EvalType {
	return EvalTypeNull
}
