package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

// Compile-time proof of interface implementation.
var _ TeamMembers = (*teamMembers)(nil)

// TeamMembers describes all the team member related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/team-members.html
type TeamMembers interface {
	// List returns all Users of a team calling ListUsers
	// See ListOrganizationMemberships for fetching memberships
	List(ctx context.Context, teamID string) ([]*User, error)

	// ListUsers returns the Users of this team.
	ListUsers(ctx context.Context, teamID string) ([]*User, error)

	// ListOrganizationMemberships returns the OrganizationMemberships of this team.
	ListOrganizationMemberships(ctx context.Context, teamID string) ([]*OrganizationMembership, error)

	// Add multiple users to a team.
	Add(ctx context.Context, teamID string, options TeamMemberAddOptions) error

	// Remove multiple users from a team.
	Remove(ctx context.Context, teamID string, options TeamMemberRemoveOptions) error
}

// teamMembers implements TeamMembers.
type teamMembers struct {
	client *Client
}

type teamMemberUser struct {
	Username string `jsonapi:"primary,users"`
}

type teamMemberOrgMembership struct {
	ID string `jsonapi:"primary,organization-memberships"`
}

// List returns all Users of a team calling ListUsers
// See ListOrganizationMemberships for fetching memberships
func (s *teamMembers) List(ctx context.Context, teamID string) ([]*User, error) {
	return s.ListUsers(ctx, teamID)
}

// ListUsers returns the Users of this team.
func (s *teamMembers) ListUsers(ctx context.Context, teamID string) ([]*User, error) {
	if !validStringID(&teamID) {
		return nil, errors.New("invalid value for team ID")
	}

	options := struct {
		Include string `url:"include"`
	}{
		Include: "users",
	}

	u := fmt.Sprintf("teams/%s", url.QueryEscape(teamID))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	t := &Team{}
	err = s.client.do(ctx, req, t)
	if err != nil {
		return nil, err
	}

	return t.Users, nil
}

// ListOrganizationMemberships returns the OrganizationMemberships of this team.
func (s *teamMembers) ListOrganizationMemberships(ctx context.Context, teamID string) ([]*OrganizationMembership, error) {
	if !validStringID(&teamID) {
		return nil, errors.New("invalid value for team ID")
	}

	options := struct {
		Include string `url:"include"`
	}{
		Include: "organization-memberships",
	}

	u := fmt.Sprintf("teams/%s", url.QueryEscape(teamID))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	t := &Team{}
	err = s.client.do(ctx, req, t)
	if err != nil {
		return nil, err
	}

	return t.OrganizationMemberships, nil
}

// TeamMemberAddOptions represents the options for
// adding or removing team members.
type TeamMemberAddOptions struct {
	Usernames                 []string
	OrganizationMembershipIDs []string
}

func (o *TeamMemberAddOptions) valid() error {
	if o.Usernames == nil && o.OrganizationMembershipIDs == nil {
		return errors.New("usernames or organization membership ids are required")
	}
	if o.Usernames != nil && o.OrganizationMembershipIDs != nil {
		return errors.New("only one of usernames or organization membership ids can be provided")
	}
	if o.Usernames != nil && len(o.Usernames) == 0 {
		return errors.New("invalid value for usernames")
	}
	if o.OrganizationMembershipIDs != nil && len(o.OrganizationMembershipIDs) == 0 {
		return errors.New("invalid value for organization membership ids")
	}
	return nil
}

// kind returns "users" or "organization-memberships"
// depending on which is defined
func (o *TeamMemberAddOptions) kind() string {
	if o.Usernames != nil && len(o.Usernames) != 0 {
		return "users"
	}
	return "organization-memberships"
}

// Add multiple users to a team.
func (s *teamMembers) Add(ctx context.Context, teamID string, options TeamMemberAddOptions) error {
	if !validStringID(&teamID) {
		return errors.New("invalid value for team ID")
	}
	if err := options.valid(); err != nil {
		return err
	}

	usersOrMemberships := options.kind()
	URL := fmt.Sprintf("teams/%s/relationships/%s", url.QueryEscape(teamID), usersOrMemberships)

	var req *retryablehttp.Request

	if usersOrMemberships == "users" {
		var err error
		var members []*teamMemberUser
		for _, name := range options.Usernames {
			members = append(members, &teamMemberUser{Username: name})
		}
		req, err = s.client.newRequest("POST", URL, members)
		if err != nil {
			return err
		}
	} else {
		var err error
		var members []*teamMemberOrgMembership
		for _, ID := range options.OrganizationMembershipIDs {
			members = append(members, &teamMemberOrgMembership{ID: ID})
		}
		req, err = s.client.newRequest("POST", URL, members)
		if err != nil {
			return err
		}
	}

	return s.client.do(ctx, req, nil)
}

// TeamMemberRemoveOptions represents the options for
// adding or removing team members.
type TeamMemberRemoveOptions struct {
	Usernames                 []string
	OrganizationMembershipIDs []string
}

func (o *TeamMemberRemoveOptions) valid() error {
	if o.Usernames == nil && o.OrganizationMembershipIDs == nil {
		return errors.New("usernames or organization membership ids are required")
	}
	if o.Usernames != nil && o.OrganizationMembershipIDs != nil {
		return errors.New("only one of usernames or organization membership ids can be provided")
	}
	if o.Usernames != nil && len(o.Usernames) == 0 {
		return errors.New("invalid value for usernames")
	}
	if o.OrganizationMembershipIDs != nil && len(o.OrganizationMembershipIDs) == 0 {
		return errors.New("invalid value for organization membership ids")
	}
	return nil
}

// kind returns "users" or "organization-memberships"
// depending on which is defined
func (o *TeamMemberRemoveOptions) kind() string {
	if o.Usernames != nil && len(o.Usernames) != 0 {
		return "users"
	}
	return "organization-memberships"
}

// Remove multiple users from a team.
func (s *teamMembers) Remove(ctx context.Context, teamID string, options TeamMemberRemoveOptions) error {
	if !validStringID(&teamID) {
		return errors.New("invalid value for team ID")
	}
	if err := options.valid(); err != nil {
		return err
	}

	usersOrMemberships := options.kind()
	URL := fmt.Sprintf("teams/%s/relationships/%s", url.QueryEscape(teamID), usersOrMemberships)

	var req *retryablehttp.Request

	if usersOrMemberships == "users" {
		var err error
		var members []*teamMemberUser
		for _, name := range options.Usernames {
			members = append(members, &teamMemberUser{Username: name})
		}
		req, err = s.client.newRequest("DELETE", URL, members)
		if err != nil {
			return err
		}
	} else {
		var err error
		var members []*teamMemberOrgMembership
		for _, ID := range options.OrganizationMembershipIDs {
			members = append(members, &teamMemberOrgMembership{ID: ID})
		}
		req, err = s.client.newRequest("DELETE", URL, members)
		if err != nil {
			return err
		}
	}

	return s.client.do(ctx, req, nil)
}
