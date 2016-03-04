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
	return fmt.Sprintf("Warnings: %s. Errors: %s", e.Warnings, e.Errors)
}

// EvalValidateCount is an EvalNode implementation that validates
// the count of a resource.
type EvalValidateCount struct {
	Resource *config.Resource
}

// TODO: test
func (n *EvalValidateCount) Eval(ctx EvalContext) (interface{}, error) {
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
		// If we can't get the count during validation, then
		// just replace it with the number 1.
		c := n.Resource.RawCount.Config()
		c[n.Resource.RawCount.Key] = "1"
		count = 1
	}

	if count < 0 {
		errs = append(errs, fmt.Errorf(
			"Count is less than zero: %d", count))
	}

RETURN:
	if len(errs) != 0 {
		err = &EvalValidateError{
			Errors: errs,
		}
	}
	return nil, err
}

// EvalValidateProvider is an EvalNode implementation that validates
// the configuration of a resource.
type EvalValidateProvider struct {
	Provider *ResourceProvider
	Config   **ResourceConfig
}

func (n *EvalValidateProvider) Eval(ctx EvalContext) (interface{}, error) {
	provider := *n.Provider
	config := *n.Config

	warns, errs := provider.Validate(config)
	if len(warns) == 0 && len(errs) == 0 {
		return nil, nil
	}

	return nil, &EvalValidateError{
		Warnings: warns,
		Errors:   errs,
	}
}

// EvalValidateProvisioner is an EvalNode implementation that validates
// the configuration of a resource.
type EvalValidateProvisioner struct {
	Provisioner *ResourceProvisioner
	Config      **ResourceConfig
}

func (n *EvalValidateProvisioner) Eval(ctx EvalContext) (interface{}, error) {
	provisioner := *n.Provisioner
	config := *n.Config
	warns, errs := provisioner.Validate(config)
	if len(warns) == 0 && len(errs) == 0 {
		return nil, nil
	}

	return nil, &EvalValidateError{
		Warnings: warns,
		Errors:   errs,
	}
}

// EvalValidateResource is an EvalNode implementation that validates
// the configuration of a resource.
type EvalValidateResource struct {
	Provider     *ResourceProvider
	Config       **ResourceConfig
	ResourceName string
	ResourceType string
}

func (n *EvalValidateResource) Eval(ctx EvalContext) (interface{}, error) {
	// TODO: test

	provider := *n.Provider
	cfg := *n.Config
	warns, errs := provider.ValidateResource(n.ResourceType, cfg)

	// If the resouce name doesn't match the name regular
	// expression, show a warning.
	if !config.NameRegexp.Match([]byte(n.ResourceName)) {
		errs = append(errs, fmt.Errorf(
			"%s: resource name can only contain letters, numbers, "+
				"dashes, and underscores."+
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
