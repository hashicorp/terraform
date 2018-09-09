package tfe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ Policies = (*policies)(nil)

// Policies describes all the policy related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/policies.html
type Policies interface {
	// List all the policies for a given organization
	List(ctx context.Context, organization string, options PolicyListOptions) (*PolicyList, error)

	// Create a policy and associate it with an organization.
	Create(ctx context.Context, organization string, options PolicyCreateOptions) (*Policy, error)

	// Read a policy by its ID.
	Read(ctx context.Context, policyID string) (*Policy, error)

	// Update an existing policy.
	Update(ctx context.Context, policyID string, options PolicyUpdateOptions) (*Policy, error)

	// Delete a policy by its ID.
	Delete(ctx context.Context, policyID string) error

	// Upload the policy content of the policy.
	Upload(ctx context.Context, policyID string, content []byte) error

	// Upload the policy content of the policy.
	Download(ctx context.Context, policyID string) ([]byte, error)
}

// policies implements Policies.
type policies struct {
	client *Client
}

// EnforcementLevel represents an enforcement level.
type EnforcementLevel string

// List the available enforcement types.
const (
	EnforcementAdvisory EnforcementLevel = "advisory"
	EnforcementHard     EnforcementLevel = "hard-mandatory"
	EnforcementSoft     EnforcementLevel = "soft-mandatory"
)

// PolicyList represents a list of policies..
type PolicyList struct {
	*Pagination
	Items []*Policy
}

// Policy represents a Terraform Enterprise policy.
type Policy struct {
	ID        string         `jsonapi:"primary,policies"`
	Name      string         `jsonapi:"attr,name"`
	Enforce   []*Enforcement `jsonapi:"attr,enforce"`
	UpdatedAt time.Time      `jsonapi:"attr,updated-at,iso8601"`
}

// Enforcement describes a enforcement.
type Enforcement struct {
	Path string           `json:"path"`
	Mode EnforcementLevel `json:"mode"`
}

// PolicyListOptions represents the options for listing policies.
type PolicyListOptions struct {
	ListOptions
}

// List all the policies for a given organization
func (s *policies) List(ctx context.Context, organization string, options PolicyListOptions) (*PolicyList, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/policies", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	pl := &PolicyList{}
	err = s.client.do(ctx, req, pl)
	if err != nil {
		return nil, err
	}

	return pl, nil
}

// PolicyCreateOptions represents the options for creating a new policy.
type PolicyCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,policies"`

	// The name of the policy.
	Name *string `jsonapi:"attr,name"`

	// The enforcements of the policy.
	Enforce []*EnforcementOptions `jsonapi:"attr,enforce"`
}

// EnforcementOptions represents the enforcement options of a policy.
type EnforcementOptions struct {
	Path *string           `json:"path,omitempty"`
	Mode *EnforcementLevel `json:"mode"`
}

func (o PolicyCreateOptions) valid() error {
	if !validString(o.Name) {
		return errors.New("Name is required")
	}
	if !validStringID(o.Name) {
		return errors.New("Invalid value for name")
	}
	if o.Enforce == nil {
		return errors.New("Enforce is required")
	}
	for _, e := range o.Enforce {
		if !validString(e.Path) {
			return errors.New("Enforcement path is required")
		}
		if e.Mode == nil {
			return errors.New("Enforcement mode is required")
		}
	}
	return nil
}

// Create a policy and associate it with an organization.
func (s *policies) Create(ctx context.Context, organization string, options PolicyCreateOptions) (*Policy, error) {
	if !validStringID(&organization) {
		return nil, errors.New("Invalid value for organization")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s/policies", url.QueryEscape(organization))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	p := &Policy{}
	err = s.client.do(ctx, req, p)
	if err != nil {
		return nil, err
	}

	return p, err
}

// Read a policy by its ID.
func (s *policies) Read(ctx context.Context, policyID string) (*Policy, error) {
	if !validStringID(&policyID) {
		return nil, errors.New("Invalid value for policy ID")
	}

	u := fmt.Sprintf("policies/%s", url.QueryEscape(policyID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	p := &Policy{}
	err = s.client.do(ctx, req, p)
	if err != nil {
		return nil, err
	}

	return p, err
}

// PolicyUpdateOptions represents the options for updating a policy.
type PolicyUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,policies"`

	// The enforcements of the policy.
	Enforce []*EnforcementOptions `jsonapi:"attr,enforce"`
}

func (o PolicyUpdateOptions) valid() error {
	if o.Enforce == nil {
		return errors.New("Enforce is required")
	}
	return nil
}

// Update an existing policy.
func (s *policies) Update(ctx context.Context, policyID string, options PolicyUpdateOptions) (*Policy, error) {
	if !validStringID(&policyID) {
		return nil, errors.New("Invalid value for policy ID")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("policies/%s", url.QueryEscape(policyID))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	p := &Policy{}
	err = s.client.do(ctx, req, p)
	if err != nil {
		return nil, err
	}

	return p, err
}

// Delete a policy by its ID.
func (s *policies) Delete(ctx context.Context, policyID string) error {
	if !validStringID(&policyID) {
		return errors.New("Invalid value for policy ID")
	}

	u := fmt.Sprintf("policies/%s", url.QueryEscape(policyID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// Upload the policy content of the policy.
func (s *policies) Upload(ctx context.Context, policyID string, content []byte) error {
	if !validStringID(&policyID) {
		return errors.New("Invalid value for policy ID")
	}

	u := fmt.Sprintf("policies/%s/upload", url.QueryEscape(policyID))
	req, err := s.client.newRequest("PUT", u, content)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// Download the policy content of the policy.
func (s *policies) Download(ctx context.Context, policyID string) ([]byte, error) {
	if !validStringID(&policyID) {
		return nil, errors.New("Invalid value for policy ID")
	}

	u := fmt.Sprintf("policies/%s/download", url.QueryEscape(policyID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = s.client.do(ctx, req, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
