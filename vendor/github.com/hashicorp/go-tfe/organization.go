package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ Organizations = (*organizations)(nil)

// Organizations describes all the organization related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/organizations.html
type Organizations interface {
	// List all the organizations visible to the current user.
	List(ctx context.Context, options OrganizationListOptions) (*OrganizationList, error)

	// Create a new organization with the given options.
	Create(ctx context.Context, options OrganizationCreateOptions) (*Organization, error)

	// Read an organization by its name.
	Read(ctx context.Context, organization string) (*Organization, error)

	// Update attributes of an existing organization.
	Update(ctx context.Context, organization string, options OrganizationUpdateOptions) (*Organization, error)

	// Delete an organization by its name.
	Delete(ctx context.Context, organization string) error

	// Capacity shows the current run capacity of an organization.
	Capacity(ctx context.Context, organization string) (*Capacity, error)

	// Entitlements shows the entitlements of an organization.
	Entitlements(ctx context.Context, organization string) (*Entitlements, error)

	// RunQueue shows the current run queue of an organization.
	RunQueue(ctx context.Context, organization string, options RunQueueOptions) (*RunQueue, error)
}

// organizations implements Organizations.
type organizations struct {
	client *Client
}

// AuthPolicyType represents an authentication policy type.
type AuthPolicyType string

// List of available authentication policies.
const (
	AuthPolicyPassword  AuthPolicyType = "password"
	AuthPolicyTwoFactor AuthPolicyType = "two_factor_mandatory"
)

// EnterprisePlanType represents an enterprise plan type.
type EnterprisePlanType string

// List of available enterprise plan types.
const (
	EnterprisePlanDisabled EnterprisePlanType = "disabled"
	EnterprisePlanPremium  EnterprisePlanType = "premium"
	EnterprisePlanPro      EnterprisePlanType = "pro"
	EnterprisePlanTrial    EnterprisePlanType = "trial"
)

// OrganizationList represents a list of organizations.
type OrganizationList struct {
	*Pagination
	Items []*Organization
}

// Organization represents a Terraform Enterprise organization.
type Organization struct {
	Name                   string                   `jsonapi:"primary,organizations"`
	CollaboratorAuthPolicy AuthPolicyType           `jsonapi:"attr,collaborator-auth-policy"`
	CreatedAt              time.Time                `jsonapi:"attr,created-at,iso8601"`
	Email                  string                   `jsonapi:"attr,email"`
	EnterprisePlan         EnterprisePlanType       `jsonapi:"attr,enterprise-plan"`
	OwnersTeamSamlRoleID   string                   `jsonapi:"attr,owners-team-saml-role-id"`
	Permissions            *OrganizationPermissions `jsonapi:"attr,permissions"`
	SAMLEnabled            bool                     `jsonapi:"attr,saml-enabled"`
	SessionRemember        int                      `jsonapi:"attr,session-remember"`
	SessionTimeout         int                      `jsonapi:"attr,session-timeout"`
	TrialExpiresAt         time.Time                `jsonapi:"attr,trial-expires-at,iso8601"`
	TwoFactorConformant    bool                     `jsonapi:"attr,two-factor-conformant"`
}

// Capacity represents the current run capacity of an organization.
type Capacity struct {
	Organization string `jsonapi:"primary,organization-capacity"`
	Pending      int    `jsonapi:"attr,pending"`
	Running      int    `jsonapi:"attr,running"`
}

// Entitlements represents the entitlements of an organization.
type Entitlements struct {
	ID                    string `jsonapi:"primary,entitlement-sets"`
	StateStorage          bool   `jsonapi:"attr,state-storage"`
	Operations            bool   `jsonapi:"attr,operations"`
	VCSIntegrations       bool   `jsonapi:"attr,vcs-integrations"`
	Sentinel              bool   `jsonapi:"attr,sentinel"`
	PrivateModuleRegistry bool   `jsonapi:"attr,private-module-registry"`
	Teams                 bool   `jsonapi:"attr,teams"`
}

// RunQueue represents the current run queue of an organization.
type RunQueue struct {
	*Pagination
	Items []*Run
}

// OrganizationPermissions represents the organization permissions.
type OrganizationPermissions struct {
	CanCreateTeam               bool `json:"can-create-team"`
	CanCreateWorkspace          bool `json:"can-create-workspace"`
	CanCreateWorkspaceMigration bool `json:"can-create-workspace-migration"`
	CanDestroy                  bool `json:"can-destroy"`
	CanTraverse                 bool `json:"can-traverse"`
	CanUpdate                   bool `json:"can-update"`
	CanUpdateAPIToken           bool `json:"can-update-api-token"`
	CanUpdateOAuth              bool `json:"can-update-oauth"`
	CanUpdateSentinel           bool `json:"can-update-sentinel"`
}

// OrganizationListOptions represents the options for listing organizations.
type OrganizationListOptions struct {
	ListOptions
}

// List all the organizations visible to the current user.
func (s *organizations) List(ctx context.Context, options OrganizationListOptions) (*OrganizationList, error) {
	req, err := s.client.newRequest("GET", "organizations", &options)
	if err != nil {
		return nil, err
	}

	orgl := &OrganizationList{}
	err = s.client.do(ctx, req, orgl)
	if err != nil {
		return nil, err
	}

	return orgl, nil
}

// OrganizationCreateOptions represents the options for creating an organization.
type OrganizationCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,organizations"`

	// Name of the organization.
	Name *string `jsonapi:"attr,name"`

	// Admin email address.
	Email *string `jsonapi:"attr,email"`
}

func (o OrganizationCreateOptions) valid() error {
	if !validString(o.Name) {
		return errors.New("name is required")
	}
	if !validStringID(o.Name) {
		return errors.New("invalid value for name")
	}
	if !validString(o.Email) {
		return errors.New("email is required")
	}
	return nil
}

// Create a new organization with the given options.
func (s *organizations) Create(ctx context.Context, options OrganizationCreateOptions) (*Organization, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("POST", "organizations", &options)
	if err != nil {
		return nil, err
	}

	org := &Organization{}
	err = s.client.do(ctx, req, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// Read an organization by its name.
func (s *organizations) Read(ctx context.Context, organization string) (*Organization, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	org := &Organization{}
	err = s.client.do(ctx, req, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// OrganizationUpdateOptions represents the options for updating an organization.
type OrganizationUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,organizations"`

	// New name for the organization.
	Name *string `jsonapi:"attr,name,omitempty"`

	// New admin email address.
	Email *string `jsonapi:"attr,email,omitempty"`

	// Session expiration (minutes).
	SessionRemember *int `jsonapi:"attr,session-remember,omitempty"`

	// Session timeout after inactivity (minutes).
	SessionTimeout *int `jsonapi:"attr,session-timeout,omitempty"`

	// Authentication policy.
	CollaboratorAuthPolicy *AuthPolicyType `jsonapi:"attr,collaborator-auth-policy,omitempty"`
}

// Update attributes of an existing organization.
func (s *organizations) Update(ctx context.Context, organization string, options OrganizationUpdateOptions) (*Organization, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s", url.QueryEscape(organization))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	org := &Organization{}
	err = s.client.do(ctx, req, org)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// Delete an organization by its name.
func (s *organizations) Delete(ctx context.Context, organization string) error {
	if !validStringID(&organization) {
		return errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s", url.QueryEscape(organization))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// Capacity shows the currently used capacity of an organization.
func (s *organizations) Capacity(ctx context.Context, organization string) (*Capacity, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/capacity", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	c := &Capacity{}
	err = s.client.do(ctx, req, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Entitlements shows the entitlements of an organization.
func (s *organizations) Entitlements(ctx context.Context, organization string) (*Entitlements, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/entitlement-set", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	e := &Entitlements{}
	err = s.client.do(ctx, req, e)
	if err != nil {
		return nil, err
	}

	return e, nil
}

// RunQueueOptions represents the options for showing the queue.
type RunQueueOptions struct {
	ListOptions
}

// RunQueue shows the current run queue of an organization.
func (s *organizations) RunQueue(ctx context.Context, organization string, options RunQueueOptions) (*RunQueue, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/runs/queue", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	rq := &RunQueue{}
	err = s.client.do(ctx, req, rq)
	if err != nil {
		return nil, err
	}

	return rq, nil
}
