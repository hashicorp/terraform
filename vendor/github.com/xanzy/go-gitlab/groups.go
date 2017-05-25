//
// Copyright 2015, Sander van Harmelen
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package gitlab

import (
	"fmt"
	"net/url"
	"time"
)

// GroupsService handles communication with the group related methods of
// the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md
type GroupsService struct {
	client *Client
}

// Group represents a GitLab group.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md
type Group struct {
	ID                   int                  `json:"id"`
	Name                 string               `json:"name"`
	Path                 string               `json:"path"`
	Description          string               `json:"description"`
	AvatarURL            string               `json:"avatar_url"`
	LFSEnabled           bool                 `json:"lfs_enabled"`
	Projects             []*Project           `json:"projects"`
	Statistics           *StorageStatistics   `json:"statistics"`
	RequestAccessEnabled bool                 `json:"request_access_enabled"`
	VisibilityLevel      VisibilityLevelValue `json:"visibility_level"`
	WebURL               string               `json:"web_url"`
}

// ListGroupsOptions represents the available ListGroups() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#list-project-groups
type ListGroupsOptions struct {
	ListOptions
	Search     *string `url:"search,omitempty" json:"search,omitempty"`
	Statistics *bool   `url:"statistics,omitempty" json:"statistics,omitempty"`
}

// ListGroups gets a list of groups. (As user: my groups, as admin: all groups)
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#list-project-groups
func (s *GroupsService) ListGroups(opt *ListGroupsOptions, options ...OptionFunc) ([]*Group, *Response, error) {
	req, err := s.client.NewRequest("GET", "groups", opt, options)
	if err != nil {
		return nil, nil, err
	}

	var g []*Group
	resp, err := s.client.Do(req, &g)
	if err != nil {
		return nil, resp, err
	}

	return g, resp, err
}

// GetGroup gets all details of a group.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#details-of-a-group
func (s *GroupsService) GetGroup(gid interface{}, options ...OptionFunc) (*Group, *Response, error) {
	group, err := parseID(gid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("groups/%s", url.QueryEscape(group))

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	g := new(Group)
	resp, err := s.client.Do(req, g)
	if err != nil {
		return nil, resp, err
	}

	return g, resp, err
}

// CreateGroupOptions represents the available CreateGroup() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#new-group
type CreateGroupOptions struct {
	Name                 *string               `url:"name,omitempty" json:"name,omitempty"`
	Path                 *string               `url:"path,omitempty" json:"path,omitempty"`
	Description          *string               `url:"description,omitempty" json:"description,omitempty"`
	LFSEnabled           *bool                 `url:"lfs_enabled,omitempty" json:"lfs_enabled,omitempty"`
	RequestAccessEnabled *bool                 `url:"request_access_enabled,omitempty" json:"request_access_enabled,omitempty"`
	VisibilityLevel      *VisibilityLevelValue `url:"visibility_level,omitempty" json:"visibility_level,omitempty"`
}

// CreateGroup creates a new project group. Available only for users who can
// create groups.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#new-group
func (s *GroupsService) CreateGroup(opt *CreateGroupOptions, options ...OptionFunc) (*Group, *Response, error) {
	req, err := s.client.NewRequest("POST", "groups", opt, options)
	if err != nil {
		return nil, nil, err
	}

	g := new(Group)
	resp, err := s.client.Do(req, g)
	if err != nil {
		return nil, resp, err
	}

	return g, resp, err
}

// UpdateGroupOptions represents the set of available options to update a Group;
// as of today these are exactly the same available when creating a new Group.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#update-group
type UpdateGroupOptions CreateGroupOptions

// UpdateGroup updates an existing group; only available to group owners and
// administrators.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#update-group
func (s *GroupsService) UpdateGroup(gid interface{}, opt *UpdateGroupOptions, options ...OptionFunc) (*Group, *Response, error) {
	group, err := parseID(gid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("groups/%s", url.QueryEscape(group))

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	g := new(Group)
	resp, err := s.client.Do(req, g)
	if err != nil {
		return nil, resp, err
	}

	return g, resp, err
}

// TransferGroup transfers a project to the Group namespace. Available only
// for admin.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#transfer-project-to-group
func (s *GroupsService) TransferGroup(gid interface{}, project int, options ...OptionFunc) (*Group, *Response, error) {
	group, err := parseID(gid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("groups/%s/projects/%d", url.QueryEscape(group), project)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	g := new(Group)
	resp, err := s.client.Do(req, g)
	if err != nil {
		return nil, resp, err
	}

	return g, resp, err
}

// DeleteGroup removes group with all projects inside.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#remove-group
func (s *GroupsService) DeleteGroup(gid interface{}, options ...OptionFunc) (*Response, error) {
	group, err := parseID(gid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("groups/%s", url.QueryEscape(group))

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// SearchGroup get all groups that match your string in their name or path.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#search-for-group
func (s *GroupsService) SearchGroup(query string, options ...OptionFunc) ([]*Group, *Response, error) {
	var q struct {
		Search string `url:"search,omitempty" json:"search,omitempty"`
	}
	q.Search = query

	req, err := s.client.NewRequest("GET", "groups", &q, options)
	if err != nil {
		return nil, nil, err
	}

	var g []*Group
	resp, err := s.client.Do(req, &g)
	if err != nil {
		return nil, resp, err
	}

	return g, resp, err
}

// GroupMember represents a GitLab group member.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md
type GroupMember struct {
	ID          int              `json:"id"`
	Username    string           `json:"username"`
	Email       string           `json:"email"`
	Name        string           `json:"name"`
	State       string           `json:"state"`
	CreatedAt   *time.Time       `json:"created_at"`
	AccessLevel AccessLevelValue `json:"access_level"`
}

// ListGroupMembersOptions represents the available ListGroupMembers()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#list-group-members
type ListGroupMembersOptions struct {
	ListOptions
}

// ListGroupMembers get a list of group members viewable by the authenticated
// user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#list-group-members
func (s *GroupsService) ListGroupMembers(gid interface{}, opt *ListGroupMembersOptions, options ...OptionFunc) ([]*GroupMember, *Response, error) {
	group, err := parseID(gid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("groups/%s/members", url.QueryEscape(group))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var g []*GroupMember
	resp, err := s.client.Do(req, &g)
	if err != nil {
		return nil, resp, err
	}

	return g, resp, err
}

// ListGroupProjectsOptions represents the available ListGroupProjects()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#list-a-group-s-projects
type ListGroupProjectsOptions struct {
	ListOptions
}

// ListGroupProjects get a list of group projects
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#list-a-group-s-projects
func (s *GroupsService) ListGroupProjects(gid interface{}, opt *ListGroupProjectsOptions, options ...OptionFunc) ([]*Project, *Response, error) {
	group, err := parseID(gid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("groups/%s/projects", url.QueryEscape(group))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var p []*Project
	resp, err := s.client.Do(req, &p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}

// AddGroupMemberOptions represents the available AddGroupMember() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#add-group-member
type AddGroupMemberOptions struct {
	UserID      *int              `url:"user_id,omitempty" json:"user_id,omitempty"`
	AccessLevel *AccessLevelValue `url:"access_level,omitempty" json:"access_level,omitempty"`
}

// AddGroupMember adds a user to the list of group members.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#list-group-members
func (s *GroupsService) AddGroupMember(gid interface{}, opt *AddGroupMemberOptions, options ...OptionFunc) (*GroupMember, *Response, error) {
	group, err := parseID(gid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("groups/%s/members", url.QueryEscape(group))

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	g := new(GroupMember)
	resp, err := s.client.Do(req, g)
	if err != nil {
		return nil, resp, err
	}

	return g, resp, err
}

// UpdateGroupMemberOptions represents the available UpdateGroupMember()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#edit-group-team-member
type UpdateGroupMemberOptions struct {
	AccessLevel *AccessLevelValue `url:"access_level,omitempty" json:"access_level,omitempty"`
}

// UpdateGroupMember updates a group team member to a specified access level.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#list-group-members
func (s *GroupsService) UpdateGroupMember(gid interface{}, user int, opt *UpdateGroupMemberOptions, options ...OptionFunc) (*GroupMember, *Response, error) {
	group, err := parseID(gid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("groups/%s/members/%d", url.QueryEscape(group), user)

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	g := new(GroupMember)
	resp, err := s.client.Do(req, g)
	if err != nil {
		return nil, resp, err
	}

	return g, resp, err
}

// RemoveGroupMember removes user from user team.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/groups.md#remove-user-from-user-team
func (s *GroupsService) RemoveGroupMember(gid interface{}, user int, options ...OptionFunc) (*Response, error) {
	group, err := parseID(gid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("groups/%s/members/%d", url.QueryEscape(group), user)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}
