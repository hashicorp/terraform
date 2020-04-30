package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ OrganizationMemberships = (*organizationMemberships)(nil)

// OrganizationMemberships describes all the organization membership related methods that
// the Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/cloud/api/organization-memberships.html
type OrganizationMemberships interface {
	// List all the organization memberships of the given organization.
	List(ctx context.Context, organization string, options OrganizationMembershipListOptions) (*OrganizationMembershipList, error)

	// Create a new organization membership with the given options.
	Create(ctx context.Context, organization string, options OrganizationMembershipCreateOptions) (*OrganizationMembership, error)

	// Read an organization membership by ID
	Read(ctx context.Context, organizationMembershipID string) (*OrganizationMembership, error)

	// Read an organization membership by ID with options
	ReadWithOptions(ctx context.Context, organizationMembershipID string, options OrganizationMembershipReadOptions) (*OrganizationMembership, error)

	// Delete an organization membership by its ID.
	Delete(ctx context.Context, organizationMembershipID string) error
}

// organizationMemberships implements OrganizationMemberships.
type organizationMemberships struct {
	client *Client
}

// OrganizationMembershipStatus represents an organization membership status.
type OrganizationMembershipStatus string

// List all available organization membership statuses.
const (
	OrganizationMembershipActive  = "active"
	OrganizationMembershipInvited = "invited"
)

// OrganizationMembershipList represents a list of organization memberships.
type OrganizationMembershipList struct {
	*Pagination
	Items []*OrganizationMembership
}

// OrganizationMembership represents a Terraform Enterprise organization membership.
type OrganizationMembership struct {
	ID     string                       `jsonapi:"primary,organization-memberships"`
	Status OrganizationMembershipStatus `jsonapi:"attr,status"`
	Email  string                       `jsonapi:"attr,email"`

	// Relations
	Organization *Organization `jsonapi:"relation,organization"`
	User         *User         `jsonapi:"relation,user"`
	Teams        []*Team       `jsonapi:"relation,teams"`
}

// OrganizationMembershipListOptions represents the options for listing organization memberships.
type OrganizationMembershipListOptions struct {
	ListOptions

	Include string `url:"include"`
}

// List all the organization memberships of the given organization.
func (s *organizationMemberships) List(ctx context.Context, organization string, options OrganizationMembershipListOptions) (*OrganizationMembershipList, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/organization-memberships", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	ml := &OrganizationMembershipList{}
	err = s.client.do(ctx, req, ml)
	if err != nil {
		return nil, err
	}

	return ml, nil
}

// OrganizationMembershipCreateOptions represents the options for creating an organization membership.
type OrganizationMembershipCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,organization-memberships"`

	// User's email address.
	Email *string `jsonapi:"attr,email"`
}

func (o OrganizationMembershipCreateOptions) valid() error {
	if o.Email == nil {
		return errors.New("email is required")
	}
	return nil
}

// Create an organization membership with the given options.
func (s *organizationMemberships) Create(ctx context.Context, organization string, options OrganizationMembershipCreateOptions) (*OrganizationMembership, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	options.ID = ""

	u := fmt.Sprintf("organizations/%s/organization-memberships", url.QueryEscape(organization))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	m := &OrganizationMembership{}
	err = s.client.do(ctx, req, m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Read an organization membership by its ID.
func (s *organizationMemberships) Read(ctx context.Context, organizationMembershipID string) (*OrganizationMembership, error) {
	return s.ReadWithOptions(ctx, organizationMembershipID, OrganizationMembershipReadOptions{})
}

// OrganizationMembershipReadOptions represents the options for reading organization memberships.
type OrganizationMembershipReadOptions struct {
	Include string `url:"include"`
}

// Read an organization membership by ID with options
func (s *organizationMemberships) ReadWithOptions(ctx context.Context, organizationMembershipID string, options OrganizationMembershipReadOptions) (*OrganizationMembership, error) {
	if !validStringID(&organizationMembershipID) {
		return nil, errors.New("invalid value for membership")
	}

	u := fmt.Sprintf("organization-memberships/%s", url.QueryEscape(organizationMembershipID))
	req, err := s.client.newRequest("GET", u, &options)

	mem := &OrganizationMembership{}
	err = s.client.do(ctx, req, mem)
	if err != nil {
		return nil, err
	}

	return mem, nil
}

// Delete an organization membership by its ID.
func (s *organizationMemberships) Delete(ctx context.Context, organizationMembershipID string) error {
	if !validStringID(&organizationMembershipID) {
		return errors.New("invalid value for membership")
	}

	u := fmt.Sprintf("organization-memberships/%s", url.QueryEscape(organizationMembershipID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
