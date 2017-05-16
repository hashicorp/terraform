package triton

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/errwrap"
)

type RolesClient struct {
	*Client
}

// Roles returns a c used for accessing functions pertaining
// to Role functionality in the Triton API.
func (c *Client) Roles() *RolesClient {
	return &RolesClient{c}
}

type Role struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Policies       []string `json:"policies"`
	Members        []string `json:"policies"`
	DefaultMembers []string `json:"default_members"`
}

type ListRolesInput struct{}

func (client *RolesClient) ListRoles(ctx context.Context, _ *ListRolesInput) ([]*Role, error) {
	path := fmt.Sprintf("/%s/roles", client.accountName)
	respReader, err := client.executeRequest(ctx, http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing ListRoles request: {{err}}", err)
	}

	var result []*Role
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding ListRoles response: {{err}}", err)
	}

	return result, nil
}

type GetRoleInput struct {
	RoleID string
}

func (client *RolesClient) GetRole(ctx context.Context, input *GetRoleInput) (*Role, error) {
	path := fmt.Sprintf("/%s/roles/%s", client.accountName, input.RoleID)
	respReader, err := client.executeRequest(ctx, http.MethodGet, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing GetRole request: {{err}}", err)
	}

	var result *Role
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding GetRole response: {{err}}", err)
	}

	return result, nil
}

// CreateRoleInput represents the options that can be specified
// when creating a new role.
type CreateRoleInput struct {
	// Name of the role. Required.
	Name string `json:"name"`

	// This account's policies to be given to this role. Optional.
	Policies []string `json:"policies,omitempty"`

	// This account's user logins to be added to this role. Optional.
	Members []string `json:"members,omitempty"`

	// This account's user logins to be added to this role and have
	// it enabled by default. Optional.
	DefaultMembers []string `json:"default_members,omitempty"`
}

func (client *RolesClient) CreateRole(ctx context.Context, input *CreateRoleInput) (*Role, error) {
	path := fmt.Sprintf("/%s/roles", client.accountName)
	respReader, err := client.executeRequest(ctx, http.MethodPost, path, input)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing CreateRole request: {{err}}", err)
	}

	var result *Role
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding CreateRole response: {{err}}", err)
	}

	return result, nil
}

// UpdateRoleInput represents the options that can be specified
// when updating a role. Anything but ID can be modified.
type UpdateRoleInput struct {
	// ID of the role to modify. Required.
	RoleID string `json:"id"`

	// Name of the role. Required.
	Name string `json:"name"`

	// This account's policies to be given to this role. Optional.
	Policies []string `json:"policies,omitempty"`

	// This account's user logins to be added to this role. Optional.
	Members []string `json:"members,omitempty"`

	// This account's user logins to be added to this role and have
	// it enabled by default. Optional.
	DefaultMembers []string `json:"default_members,omitempty"`
}

func (client *RolesClient) UpdateRole(ctx context.Context, input *UpdateRoleInput) (*Role, error) {
	path := fmt.Sprintf("/%s/roles/%s", client.accountName, input.RoleID)
	respReader, err := client.executeRequest(ctx, http.MethodPost, path, input)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return nil, errwrap.Wrapf("Error executing UpdateRole request: {{err}}", err)
	}

	var result *Role
	decoder := json.NewDecoder(respReader)
	if err = decoder.Decode(&result); err != nil {
		return nil, errwrap.Wrapf("Error decoding UpdateRole response: {{err}}", err)
	}

	return result, nil
}

type DeleteRoleInput struct {
	RoleID string
}

func (client *RolesClient) DeleteRoles(ctx context.Context, input *DeleteRoleInput) error {
	path := fmt.Sprintf("/%s/roles/%s", client.accountName, input.RoleID)
	respReader, err := client.executeRequest(ctx, http.MethodDelete, path, nil)
	if respReader != nil {
		defer respReader.Close()
	}
	if err != nil {
		return errwrap.Wrapf("Error executing DeleteRole request: {{err}}", err)
	}

	return nil
}
