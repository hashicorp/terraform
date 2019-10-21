package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// Compile-time proof of interface implementation.
var _ Teams = (*teams)(nil)

// Teams describes all the team related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/teams.html
type Teams interface {
	// List all the teams of the given organization.
	List(ctx context.Context, organization string, options TeamListOptions) (*TeamList, error)

	// Create a new team with the given options.
	Create(ctx context.Context, organization string, options TeamCreateOptions) (*Team, error)

	// Read a team by its ID.
	Read(ctx context.Context, teamID string) (*Team, error)

	// Update a team by its ID.
	Update(ctx context.Context, teamID string, options TeamUpdateOptions) (*Team, error)

	// Delete a team by its ID.
	Delete(ctx context.Context, teamID string) error
}

// teams implements Teams.
type teams struct {
	client *Client
}

// TeamList represents a list of teams.
type TeamList struct {
	*Pagination
	Items []*Team
}

// Team represents a Terraform Enterprise team.
type Team struct {
	ID                 string              `jsonapi:"primary,teams"`
	Name               string              `jsonapi:"attr,name"`
	OrganizationAccess *OrganizationAccess `jsonapi:"attr,organization-access"`
	Permissions        *TeamPermissions    `jsonapi:"attr,permissions"`
	UserCount          int                 `jsonapi:"attr,users-count"`

	// Relations
	Users []*User `jsonapi:"relation,users"`
}

// OrganizationAccess represents the team's permissions on its organization
type OrganizationAccess struct {
	ManagePolicies    bool `json:"manage-policies"`
	ManageWorkspaces  bool `json:"manage-workspaces"`
	ManageVCSSettings bool `json:"manage-vcs-settings"`
}

// TeamPermissions represents the current user's permissions on the team.
type TeamPermissions struct {
	CanDestroy          bool `json:"can-destroy"`
	CanUpdateMembership bool `json:"can-update-membership"`
}

// TeamListOptions represents the options for listing teams.
type TeamListOptions struct {
	ListOptions
}

// List all the teams of the given organization.
func (s *teams) List(ctx context.Context, organization string, options TeamListOptions) (*TeamList, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/teams", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	tl := &TeamList{}
	err = s.client.do(ctx, req, tl)
	if err != nil {
		return nil, err
	}

	return tl, nil
}

// TeamCreateOptions represents the options for creating a team.
type TeamCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,teams"`

	// Name of the team.
	Name *string `jsonapi:"attr,name"`

	// The team's organization access
	OrganizationAccess *OrganizationAccessOptions `jsonapi:"attr,organization-access,omitempty"`
}

// OrganizationAccessOptions represents the organization access options of a team.
type OrganizationAccessOptions struct {
	ManagePolicies    *bool `json:"manage-policies,omitempty"`
	ManageWorkspaces  *bool `json:"manage-workspaces,omitempty"`
	ManageVCSSettings *bool `json:"manage-vcs-settings,omitempty"`
}

func (o TeamCreateOptions) valid() error {
	if !validString(o.Name) {
		return errors.New("name is required")
	}
	return nil
}

// Create a new team with the given options.
func (s *teams) Create(ctx context.Context, organization string, options TeamCreateOptions) (*Team, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s/teams", url.QueryEscape(organization))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	t := &Team{}
	err = s.client.do(ctx, req, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// Read a single team by its ID.
func (s *teams) Read(ctx context.Context, teamID string) (*Team, error) {
	if !validStringID(&teamID) {
		return nil, errors.New("invalid value for team ID")
	}

	u := fmt.Sprintf("teams/%s", url.QueryEscape(teamID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	t := &Team{}
	err = s.client.do(ctx, req, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// TeamUpdateOptions represents the options for updating a team.
type TeamUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,teams"`

	// New name for the team
	Name *string `jsonapi:"attr,name,omitempty"`

	// The team's organization access
	OrganizationAccess *OrganizationAccessOptions `jsonapi:"attr,organization-access,omitempty"`
}

// Update a team by its ID.
func (s *teams) Update(ctx context.Context, teamID string, options TeamUpdateOptions) (*Team, error) {
	if !validStringID(&teamID) {
		return nil, errors.New("invalid value for team ID")
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("teams/%s", url.QueryEscape(teamID))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	t := &Team{}
	err = s.client.do(ctx, req, t)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// Delete a team by its ID.
func (s *teams) Delete(ctx context.Context, teamID string) error {
	if !validStringID(&teamID) {
		return errors.New("invalid value for team ID")
	}

	u := fmt.Sprintf("teams/%s", url.QueryEscape(teamID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
