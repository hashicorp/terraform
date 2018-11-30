package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ Workspaces = (*workspaces)(nil)

// Workspaces describes all the workspace related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/workspaces.html
type Workspaces interface {
	// List all the workspaces within an organization.
	List(ctx context.Context, organization string, options WorkspaceListOptions) (*WorkspaceList, error)

	// Create is used to create a new workspace.
	Create(ctx context.Context, organization string, options WorkspaceCreateOptions) (*Workspace, error)

	// Read a workspace by its name.
	Read(ctx context.Context, organization string, workspace string) (*Workspace, error)

	// Update settings of an existing workspace.
	Update(ctx context.Context, organization string, workspace string, options WorkspaceUpdateOptions) (*Workspace, error)

	// Delete a workspace by its name.
	Delete(ctx context.Context, organization string, workspace string) error

	// Lock a workspace by its ID.
	Lock(ctx context.Context, workspaceID string, options WorkspaceLockOptions) (*Workspace, error)

	// Unlock a workspace by its ID.
	Unlock(ctx context.Context, workspaceID string) (*Workspace, error)

	// ForceUnlock a workspace by its ID.
	ForceUnlock(ctx context.Context, workspaceID string) (*Workspace, error)

	// AssignSSHKey to a workspace.
	AssignSSHKey(ctx context.Context, workspaceID string, options WorkspaceAssignSSHKeyOptions) (*Workspace, error)

	// UnassignSSHKey from a workspace.
	UnassignSSHKey(ctx context.Context, workspaceID string) (*Workspace, error)
}

// workspaces implements Workspaces.
type workspaces struct {
	client *Client
}

// WorkspaceList represents a list of workspaces.
type WorkspaceList struct {
	*Pagination
	Items []*Workspace
}

// Workspace represents a Terraform Enterprise workspace.
type Workspace struct {
	ID                   string                `jsonapi:"primary,workspaces"`
	Actions              *WorkspaceActions     `jsonapi:"attr,actions"`
	AutoApply            bool                  `jsonapi:"attr,auto-apply"`
	CanQueueDestroyPlan  bool                  `jsonapi:"attr,can-queue-destroy-plan"`
	CreatedAt            time.Time             `jsonapi:"attr,created-at,iso8601"`
	Environment          string                `jsonapi:"attr,environment"`
	Locked               bool                  `jsonapi:"attr,locked"`
	MigrationEnvironment string                `jsonapi:"attr,migration-environment"`
	Name                 string                `jsonapi:"attr,name"`
	Operations           bool                  `jsonapi:"attr,operations"`
	Permissions          *WorkspacePermissions `jsonapi:"attr,permissions"`
	TerraformVersion     string                `jsonapi:"attr,terraform-version"`
	VCSRepo              *VCSRepo              `jsonapi:"attr,vcs-repo"`
	WorkingDirectory     string                `jsonapi:"attr,working-directory"`

	// Relations
	CurrentRun   *Run          `jsonapi:"relation,current-run"`
	Organization *Organization `jsonapi:"relation,organization"`
	SSHKey       *SSHKey       `jsonapi:"relation,ssh-key"`
}

// VCSRepo contains the configuration of a VCS integration.
type VCSRepo struct {
	Branch            string `json:"branch"`
	Identifier        string `json:"identifier"`
	IngressSubmodules bool   `json:"ingress-submodules"`
	OAuthTokenID      string `json:"oauth-token-id"`
}

// WorkspaceActions represents the workspace actions.
type WorkspaceActions struct {
	IsDestroyable bool `json:"is-destroyable"`
}

// WorkspacePermissions represents the workspace permissions.
type WorkspacePermissions struct {
	CanDestroy        bool `json:"can-destroy"`
	CanLock           bool `json:"can-lock"`
	CanQueueDestroy   bool `json:"can-queue-destroy"`
	CanQueueRun       bool `json:"can-queue-run"`
	CanReadSettings   bool `json:"can-read-settings"`
	CanUpdate         bool `json:"can-update"`
	CanUpdateVariable bool `json:"can-update-variable"`
}

// WorkspaceListOptions represents the options for listing workspaces.
type WorkspaceListOptions struct {
	ListOptions

	// A search string (partial workspace name) used to filter the results.
	Search *string `url:"search[name],omitempty"`
}

// List all the workspaces within an organization.
func (s *workspaces) List(ctx context.Context, organization string, options WorkspaceListOptions) (*WorkspaceList, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}

	u := fmt.Sprintf("organizations/%s/workspaces", url.QueryEscape(organization))
	req, err := s.client.newRequest("GET", u, &options)
	if err != nil {
		return nil, err
	}

	wl := &WorkspaceList{}
	err = s.client.do(ctx, req, wl)
	if err != nil {
		return nil, err
	}

	return wl, nil
}

// WorkspaceCreateOptions represents the options for creating a new workspace.
type WorkspaceCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,workspaces"`

	// Whether to automatically apply changes when a Terraform plan is successful.
	AutoApply *bool `jsonapi:"attr,auto-apply,omitempty"`

	// The legacy TFE environment to use as the source of the migration, in the
	// form organization/environment. Omit this unless you are migrating a legacy
	// environment.
	MigrationEnvironment *string `jsonapi:"attr,migration-environment,omitempty"`

	// The name of the workspace, which can only include letters, numbers, -,
	// and _. This will be used as an identifier and must be unique in the
	// organization.
	Name *string `jsonapi:"attr,name"`

	// The version of Terraform to use for this workspace. Upon creating a
	// workspace, the latest version is selected unless otherwise specified.
	TerraformVersion *string `jsonapi:"attr,terraform-version,omitempty"`

	// Settings for the workspace's VCS repository. If omitted, the workspace is
	// created without a VCS repo. If included, you must specify at least the
	// oauth-token-id and identifier keys below.
	VCSRepo *VCSRepoOptions `jsonapi:"attr,vcs-repo,omitempty"`

	// A relative path that Terraform will execute within. This defaults to the
	// root of your repository and is typically set to a subdirectory matching the
	// environment when multiple environments exist within the same repository.
	WorkingDirectory *string `jsonapi:"attr,working-directory,omitempty"`
}

// VCSRepoOptions represents the configuration options of a VCS integration.
type VCSRepoOptions struct {
	Branch            *string `json:"branch,omitempty"`
	Identifier        *string `json:"identifier,omitempty"`
	IngressSubmodules *bool   `json:"ingress-submodules,omitempty"`
	OAuthTokenID      *string `json:"oauth-token-id,omitempty"`
}

func (o WorkspaceCreateOptions) valid() error {
	if !validString(o.Name) {
		return errors.New("name is required")
	}
	if !validStringID(o.Name) {
		return errors.New("invalid value for name")
	}
	return nil
}

// Create is used to create a new workspace.
func (s *workspaces) Create(ctx context.Context, organization string, options WorkspaceCreateOptions) (*Workspace, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("organizations/%s/workspaces", url.QueryEscape(organization))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = s.client.do(ctx, req, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// Read a workspace by its name.
func (s *workspaces) Read(ctx context.Context, organization, workspace string) (*Workspace, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}
	if !validStringID(&workspace) {
		return nil, errors.New("invalid value for workspace")
	}

	u := fmt.Sprintf(
		"organizations/%s/workspaces/%s",
		url.QueryEscape(organization),
		url.QueryEscape(workspace),
	)
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = s.client.do(ctx, req, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// WorkspaceUpdateOptions represents the options for updating a workspace.
type WorkspaceUpdateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,workspaces"`

	// Whether to automatically apply changes when a Terraform plan is successful.
	AutoApply *bool `jsonapi:"attr,auto-apply,omitempty"`

	// A new name for the workspace, which can only include letters, numbers, -,
	// and _. This will be used as an identifier and must be unique in the
	// organization. Warning: Changing a workspace's name changes its URL in the
	// API and UI.
	Name *string `jsonapi:"attr,name,omitempty"`

	// The version of Terraform to use for this workspace.
	TerraformVersion *string `jsonapi:"attr,terraform-version,omitempty"`

	// To delete a workspace's existing VCS repo, specify null instead of an
	// object. To modify a workspace's existing VCS repo, include whichever of
	// the keys below you wish to modify. To add a new VCS repo to a workspace
	// that didn't previously have one, include at least the oauth-token-id and
	// identifier keys.  VCSRepo *VCSRepo `jsonapi:"relation,vcs-repo,om-tempty"`
	VCSRepo *VCSRepoOptions `jsonapi:"attr,vcs-repo,omitempty"`

	// A relative path that Terraform will execute within. This defaults to the
	// root of your repository and is typically set to a subdirectory matching
	// the environment when multiple environments exist within the same
	// repository.
	WorkingDirectory *string `jsonapi:"attr,working-directory,omitempty"`
}

// Update settings of an existing workspace.
func (s *workspaces) Update(ctx context.Context, organization, workspace string, options WorkspaceUpdateOptions) (*Workspace, error) {
	if !validStringID(&organization) {
		return nil, errors.New("invalid value for organization")
	}
	if !validStringID(&workspace) {
		return nil, errors.New("invalid value for workspace")
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf(
		"organizations/%s/workspaces/%s",
		url.QueryEscape(organization),
		url.QueryEscape(workspace),
	)
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = s.client.do(ctx, req, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// Delete a workspace by its name.
func (s *workspaces) Delete(ctx context.Context, organization, workspace string) error {
	if !validStringID(&organization) {
		return errors.New("invalid value for organization")
	}
	if !validStringID(&workspace) {
		return errors.New("invalid value for workspace")
	}

	u := fmt.Sprintf(
		"organizations/%s/workspaces/%s",
		url.QueryEscape(organization),
		url.QueryEscape(workspace),
	)
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// WorkspaceLockOptions represents the options for locking a workspace.
type WorkspaceLockOptions struct {
	// Specifies the reason for locking the workspace.
	Reason *string `json:"reason,omitempty"`
}

// Lock a workspace by its ID.
func (s *workspaces) Lock(ctx context.Context, workspaceID string, options WorkspaceLockOptions) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("invalid value for workspace ID")
	}

	u := fmt.Sprintf("workspaces/%s/actions/lock", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = s.client.do(ctx, req, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// Unlock a workspace by its ID.
func (s *workspaces) Unlock(ctx context.Context, workspaceID string) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("invalid value for workspace ID")
	}

	u := fmt.Sprintf("workspaces/%s/actions/unlock", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = s.client.do(ctx, req, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// ForceUnlock a workspace by its ID.
func (s *workspaces) ForceUnlock(ctx context.Context, workspaceID string) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("invalid value for workspace ID")
	}

	u := fmt.Sprintf("workspaces/%s/actions/force-unlock", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = s.client.do(ctx, req, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// WorkspaceAssignSSHKeyOptions represents the options to assign an SSH key to
// a workspace.
type WorkspaceAssignSSHKeyOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,workspaces"`

	// The SSH key ID to assign.
	SSHKeyID *string `jsonapi:"attr,id"`
}

func (o WorkspaceAssignSSHKeyOptions) valid() error {
	if !validString(o.SSHKeyID) {
		return errors.New("SSH key ID is required")
	}
	if !validStringID(o.SSHKeyID) {
		return errors.New("invalid value for SSH key ID")
	}
	return nil
}

// AssignSSHKey to a workspace.
func (s *workspaces) AssignSSHKey(ctx context.Context, workspaceID string, options WorkspaceAssignSSHKeyOptions) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("invalid value for workspace ID")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("workspaces/%s/relationships/ssh-key", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("PATCH", u, &options)
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = s.client.do(ctx, req, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}

// workspaceUnassignSSHKeyOptions represents the options to unassign an SSH key
// to a workspace.
type workspaceUnassignSSHKeyOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,workspaces"`

	// Must be nil to unset the currently assigned SSH key.
	SSHKeyID *string `jsonapi:"attr,id"`
}

// UnassignSSHKey from a workspace.
func (s *workspaces) UnassignSSHKey(ctx context.Context, workspaceID string) (*Workspace, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("invalid value for workspace ID")
	}

	u := fmt.Sprintf("workspaces/%s/relationships/ssh-key", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("PATCH", u, &workspaceUnassignSSHKeyOptions{})
	if err != nil {
		return nil, err
	}

	w := &Workspace{}
	err = s.client.do(ctx, req, w)
	if err != nil {
		return nil, err
	}

	return w, nil
}
