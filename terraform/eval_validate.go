package terraform

// EvalValidateError is the error structure returned if there were
// validation errors.
type EvalValidateError struct {
	Warnings []string
	Errors   []error
}

func (e *EvalValidateError) Error() string {
	return ""
}

// EvalValidateProvider is an EvalNode implementation that validates
// the configuration of a resource.
type EvalValidateProvider struct {
	Provider EvalNode
	Config   EvalNode
}

func (n *EvalValidateProvider) Args() ([]EvalNode, []EvalType) {
	return []EvalNode{n.Provider, n.Config},
		[]EvalType{EvalTypeResourceProvider, EvalTypeConfig}
}

func (n *EvalValidateProvider) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	provider := args[0].(ResourceProvider)
	config := args[1].(*ResourceConfig)
	warns, errs := provider.Validate(config)
	if len(warns) == 0 && len(errs) == 0 {
		return nil, nil
	}

	return nil, &EvalValidateError{
		Warnings: warns,
		Errors:   errs,
	}
}

func (n *EvalValidateProvider) Type() EvalType {
	return EvalTypeNull
}

// EvalValidateResource is an EvalNode implementation that validates
// the configuration of a resource.
type EvalValidateResource struct {
	Provider     EvalNode
	Config       EvalNode
	ProviderType string
}

func (n *EvalValidateResource) Args() ([]EvalNode, []EvalType) {
	return []EvalNode{n.Provider, n.Config},
		[]EvalType{EvalTypeResourceProvider, EvalTypeConfig}
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
