package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ PolicySetParameters = (*policySetParameters)(nil)

// PolicySetParameters describes all the parameter related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/policy-set-params.html
type PolicySetParameters interface {
	// List all the parameters associated with the given policy-set.
	List(ctx context.Context, policySetID string, options PolicySetParameterListOptions) (*PolicySetParameterList, error)

	// Create is used to create a new parameter.
	Create(ctx context.Context, policySetID string, options PolicySetParameterCreateOptions) (*PolicySetParameter, error)

	// Read a parameter by its ID.
	Read(ctx context.Context, policySetID string, parameterID string) (*PolicySetParameter, error)

	// Update values of an existing parameter.
	Update(ctx context.Context, policySetID string, parameterID string, options PolicySetParameterUpdateOptions) (*PolicySetParameter, error)

	// Delete a parameter by its ID.
	Delete(ctx context.Context, policySetID string, parameterID string) error
}

// policySetParameters implements Parameters.
type policySetParameters struct {
	client *Client
}

// PolicySetParameterList represents a list of parameters.
type PolicySetParameterList struct {
	*Pagination
	Items []*PolicySetParameter
}

// PolicySetParameter represents a Policy Set parameter
type PolicySetParameter struct {
	ID        string       `jsonapi:"primary,vars"`
	Key       string       `jsonapi:"attr,key"`
	Value     string       `jsonapi:"attr,value"`
	Category  CategoryType `jsonapi:"attr,category"`
	Sensitive bool         `jsonapi:"attr,sensitive"`

	// Relations
	PolicySet *PolicySet `jsonapi:"relation,configurable"`
}

// PolicySetParameterListOptions represents the options for listing parameters.
type PolicySetParameterListOptions struct {
	ListOptions
}

func (o PolicySetParameterListOptions) valid() error {
	return nil
}

// List all the parameters associated with the given policy-set.
func (s *policySetParameters) List(ctx context.Context, policySetID string, options PolicySetParameterListOptions) (*PolicySetParameterList, error) {
	if !validStringID(&policySetID) {
		return nil, errors.New("invalid value for policy set ID")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("policy-sets/%s/parameters", policySetID)
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	vl := &PolicySetParameterList{}
	err = s.client.do(ctx, req, vl)
	if err != nil {
		return nil, err
	}

	return vl, nil
}

// PolicySetParameterCreateOptions represents the options for creating a new parameter.
type PolicySetParameterCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,vars"`

	// The name of the parameter.
	Key *string `jsonapi:"attr,key"`

	// The value of the parameter.
	Value *string `jsonapi:"attr,value,omitempty"`

	// The Category of the parameter, should always be "policy-set"
	Category *CategoryType `jsonapi:"attr,category"`

	// Whether the value is sensitive.
	Sensitive *bool `jsonapi:"attr,sensitive,omitempty"`
}

func (o PolicySetParameterCreateOptions) valid() error {
	if !validString(o.Key) {
		return errors.New("key is required")
	}
	if o.Category == nil {
		return errors.New("category is required")
	}
	if *o.Category != CategoryPolicySet {
		return errors.New("category must be policy-set")
	}
	return nil
}

// Create is used to create a new parameter.
func (s *policySetParameters) Create(ctx context.Context, policySetID string, options PolicySetParameterCreateOptions) (*PolicySetParameter, error) {
	if !validStringID(&policySetID) {
		return nil, errors.New("invalid value for policy set ID")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("policy-sets/%s/parameters", url.QueryEscape(policySetID))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	p := &PolicySetParameter{}
	err = s.client.do(ctx, req, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Read a parameter by its ID.
func (s *policySetParameters) Read(ctx context.Context, policySetID string, parameterID string) (*PolicySetParameter, error) {
	if !validStringID(&policySetID) {
		return nil, errors.New("invalid value for policy set ID")
	}
	if !validStringID(&parameterID) {
		return nil, errors.New("invalid value for parameter ID")
	}

	u := fmt.Sprintf("policy-sets/%s/parameters/%s", url.QueryEscape(policySetID), url.QueryEscape(parameterID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	p := &PolicySetParameter{}
	err = s.client.do(ctx, req, p)
	if err != nil {
		return nil, err
	}

	return p, err
}

// PolicySetParameterUpdateOptions represents the options for updating a parameter.
type PolicySetParameterUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,vars"`

	// The name of the parameter.
	Key *string `jsonapi:"attr,key,omitempty"`

	// The value of the parameter.
	Value *string `jsonapi:"attr,value,omitempty"`

	// Whether the value is sensitive.
	Sensitive *bool `jsonapi:"attr,sensitive,omitempty"`
}

// Update values of an existing parameter.
func (s *policySetParameters) Update(ctx context.Context, policySetID string, parameterID string, options PolicySetParameterUpdateOptions) (*PolicySetParameter, error) {
	if !validStringID(&policySetID) {
		return nil, errors.New("invalid value for policy set ID")
	}
	if !validStringID(&parameterID) {
		return nil, errors.New("invalid value for parameter ID")
	}

	// Make sure we don't send a user provided ID.
	options.ID = parameterID

	u := fmt.Sprintf("policy-sets/%s/parameters/%s", url.QueryEscape(policySetID), url.QueryEscape(parameterID))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	p := &PolicySetParameter{}
	err = s.client.do(ctx, req, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Delete a parameter by its ID.
func (s *policySetParameters) Delete(ctx context.Context, policySetID string, parameterID string) error {
	if !validStringID(&policySetID) {
		return errors.New("invalid value for policy set ID")
	}
	if !validStringID(&parameterID) {
		return errors.New("invalid value for parameter ID")
	}

	u := fmt.Sprintf("policy-sets/%s/parameters/%s", url.QueryEscape(policySetID), url.QueryEscape(parameterID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
