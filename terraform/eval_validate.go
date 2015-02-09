package terraform

import (
	"fmt"

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

// EvalValidateCount is an EvalNode implementation that validates
// the count of a resource.
type EvalValidateCount struct {
	Resource *config.Resource
}

func (n *EvalValidateCount) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

// TODO: test
func (n *EvalValidateCount) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	var count int
	var errs []error
	var err error
	if _, err := ctx.Interpolate(n.Resource.RawCount, nil); err != nil {
		errs = append(errs, fmt.Errorf(
			"Failed to interpolate count: %s", err))
		goto RETURN
	}

	count, err = n.Resource.Count()
	if err != nil {
		errs = append(errs)
		goto RETURN
	}

	if count < 0 {
		errs = append(errs, fmt.Errorf(
			"Count is less than zero: %d", count))
	}

RETURN:
	return nil, &EvalValidateError{
		Errors: errs,
	}
}

func (n *EvalValidateCount) Type() EvalType {
	return EvalTypeNull
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

// EvalValidateProvisioner is an EvalNode implementation that validates
// the configuration of a resource.
type EvalValidateProvisioner struct {
	Provisioner EvalNode
	Config      EvalNode
}

func (n *EvalValidateProvisioner) Args() ([]EvalNode, []EvalType) {
	return []EvalNode{n.Provisioner, n.Config},
		[]EvalType{EvalTypeResourceProvisioner, EvalTypeConfig}
}

func (n *EvalValidateProvisioner) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	provider := args[0].(ResourceProvisioner)
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

func (n *EvalValidateProvisioner) Type() EvalType {
	return EvalTypeNull
}

// EvalValidateResource is an EvalNode implementation that validates
// the configuration of a resource.
type EvalValidateResource struct {
	Provider     EvalNode
	Config       EvalNode
	ResourceName string
	ResourceType string
}

func (n *EvalValidateResource) Args() ([]EvalNode, []EvalType) {
	return []EvalNode{n.Provider, n.Config},
		[]EvalType{EvalTypeResourceProvider, EvalTypeConfig}
}

func (n *EvalValidateResource) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	// TODO: test

	provider := args[0].(ResourceProvider)
	cfg := args[1].(*ResourceConfig)
	warns, errs := provider.ValidateResource(n.ResourceType, cfg)

	// If the resouce name doesn't match the name regular
	// expression, show a warning.
	if !config.NameRegexp.Match([]byte(n.ResourceName)) {
		warns = append(warns, fmt.Sprintf(
			"%s: resource name can only contain letters, numbers, "+
				"dashes, and underscores.\n"+
				"This will be an error in Terraform 0.4",
			n.ResourceName))
	}

	if len(warns) == 0 && len(errs) == 0 {
		return nil, nil
	}

	return nil, &EvalValidateError{
		Warnings: warns,
		Errors:   errs,
	}
}

func (n *EvalValidateResource) Type() EvalType {
	return EvalTypeNull
}
