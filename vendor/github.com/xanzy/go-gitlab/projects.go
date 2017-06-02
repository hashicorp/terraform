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

// ProjectsService handles communication with the repositories related methods
// of the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md
type ProjectsService struct {
	client *Client
}

// Project represents a GitLab project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md
type Project struct {
	ID                                        int                  `json:"id"`
	Description                               string               `json:"description"`
	DefaultBranch                             string               `json:"default_branch"`
	Public                                    bool                 `json:"public"`
	VisibilityLevel                           VisibilityLevelValue `json:"visibility_level"`
	SSHURLToRepo                              string               `json:"ssh_url_to_repo"`
	HTTPURLToRepo                             string               `json:"http_url_to_repo"`
	WebURL                                    string               `json:"web_url"`
	TagList                                   []string             `json:"tag_list"`
	Owner                                     *User                `json:"owner"`
	Name                                      string               `json:"name"`
	NameWithNamespace                         string               `json:"name_with_namespace"`
	Path                                      string               `json:"path"`
	PathWithNamespace                         string               `json:"path_with_namespace"`
	IssuesEnabled                             bool                 `json:"issues_enabled"`
	OpenIssuesCount                           int                  `json:"open_issues_count"`
	MergeRequestsEnabled                      bool                 `json:"merge_requests_enabled"`
	ApprovalsBeforeMerge                      int                  `json:"approvals_before_merge"`
	BuildsEnabled                             bool                 `json:"builds_enabled"`
	WikiEnabled                               bool                 `json:"wiki_enabled"`
	SnippetsEnabled                           bool                 `json:"snippets_enabled"`
	ContainerRegistryEnabled                  bool                 `json:"container_registry_enabled"`
	CreatedAt                                 *time.Time           `json:"created_at,omitempty"`
	LastActivityAt                            *time.Time           `json:"last_activity_at,omitempty"`
	CreatorID                                 int                  `json:"creator_id"`
	Namespace                                 *ProjectNamespace    `json:"namespace"`
	Permissions                               *Permissions         `json:"permissions"`
	Archived                                  bool                 `json:"archived"`
	AvatarURL                                 string               `json:"avatar_url"`
	SharedRunnersEnabled                      bool                 `json:"shared_runners_enabled"`
	ForksCount                                int                  `json:"forks_count"`
	StarCount                                 int                  `json:"star_count"`
	RunnersToken                              string               `json:"runners_token"`
	PublicBuilds                              bool                 `json:"public_builds"`
	OnlyAllowMergeIfBuildSucceeds             bool                 `json:"only_allow_merge_if_build_succeeds"`
	OnlyAllowMergeIfAllDiscussionsAreResolved bool                 `json:"only_allow_merge_if_all_discussions_are_resolved"`
	LFSEnabled                                bool                 `json:"lfs_enabled"`
	RequestAccessEnabled                      bool                 `json:"request_access_enabled"`
	ForkedFromProject                         *ForkParent          `json:"forked_from_project"`
	SharedWithGroups                          []struct {
		GroupID          int    `json:"group_id"`
		GroupName        string `json:"group_name"`
		GroupAccessLevel int    `json:"group_access_level"`
	} `json:"shared_with_groups"`
	Statistics *ProjectStatistics `json:"statistics"`
}

// Repository represents a repository.
type Repository struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	WebURL            string `json:"web_url"`
	AvatarURL         string `json:"avatar_url"`
	GitSSHURL         string `json:"git_ssh_url"`
	GitHTTPURL        string `json:"git_http_url"`
	Namespace         string `json:"namespace"`
	VisibilityLevel   int    `json:"visibility_level"`
	PathWithNamespace string `json:"path_with_namespace"`
	DefaultBranch     string `json:"default_branch"`
	Homepage          string `json:"homepage"`
	URL               string `json:"url"`
	SSHURL            string `json:"ssh_url"`
	HTTPURL           string `json:"http_url"`
}

// ProjectNamespace represents a project namespace.
type ProjectNamespace struct {
	CreatedAt   *time.Time `json:"created_at"`
	Description string     `json:"description"`
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	OwnerID     int        `json:"owner_id"`
	Path        string     `json:"path"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// StorageStatistics represents a statistics record for a group or project.
type StorageStatistics struct {
	StorageSize        int64 `json:"storage_size"`
	RepositorySize     int64 `json:"repository_size"`
	LfsObjectsSize     int64 `json:"lfs_objects_size"`
	BuildArtifactsSize int64 `json:"build_artifacts_size"`
}

// ProjectStatistics represents a statistics record for a project.
type ProjectStatistics struct {
	StorageStatistics
	CommitCount int `json:"commit_count"`
}

// Permissions represents premissions.
type Permissions struct {
	ProjectAccess *ProjectAccess `json:"project_access"`
	GroupAccess   *GroupAccess   `json:"group_access"`
}

// ProjectAccess represents project access.
type ProjectAccess struct {
	AccessLevel       AccessLevelValue       `json:"access_level"`
	NotificationLevel NotificationLevelValue `json:"notification_level"`
}

// GroupAccess represents group access.
type GroupAccess struct {
	AccessLevel       AccessLevelValue       `json:"access_level"`
	NotificationLevel NotificationLevelValue `json:"notification_level"`
}

// ForkParent represents the parent project when this is a fork.
type ForkParent struct {
	HTTPURLToRepo     string `json:"http_url_to_repo"`
	ID                int    `json:"id"`
	Name              string `json:"name"`
	NameWithNamespace string `json:"name_with_namespace"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
}

func (s Project) String() string {
	return Stringify(s)
}

// ListProjectsOptions represents the available ListProjects() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#list-projects
type ListProjectsOptions struct {
	ListOptions
	Archived   *bool   `url:"archived,omitempty" json:"archived,omitempty"`
	OrderBy    *string `url:"order_by,omitempty" json:"order_by,omitempty"`
	Sort       *string `url:"sort,omitempty" json:"sort,omitempty"`
	Search     *string `url:"search,omitempty" json:"search,omitempty"`
	Simple     *bool   `url:"simple,omitempty" json:"simple,omitempty"`
	Visibility *string `url:"visibility,omitempty" json:"visibility,omitempty"`
	Statistics *bool   `url:"statistics,omitempty" json:"statistics,omitempty"`
}

// ListProjects gets a list of projects accessible by the authenticated user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#list-projects
func (s *ProjectsService) ListProjects(opt *ListProjectsOptions, options ...OptionFunc) ([]*Project, *Response, error) {
	req, err := s.client.NewRequest("GET", "projects", opt, options)
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

// ListOwnedProjects gets a list of projects which are owned by the
// authenticated user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#list-owned-projects
func (s *ProjectsService) ListOwnedProjects(opt *ListProjectsOptions, options ...OptionFunc) ([]*Project, *Response, error) {
	req, err := s.client.NewRequest("GET", "projects/owned", opt, options)
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

// ListStarredProjects gets a list of projects which are starred by the
// authenticated user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#list-starred-projects
func (s *ProjectsService) ListStarredProjects(opt *ListProjectsOptions, options ...OptionFunc) ([]*Project, *Response, error) {
	req, err := s.client.NewRequest("GET", "projects/starred", opt, options)
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

// ListAllProjects gets a list of all GitLab projects (admin only).
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#list-all-projects
func (s *ProjectsService) ListAllProjects(opt *ListProjectsOptions, options ...OptionFunc) ([]*Project, *Response, error) {
	req, err := s.client.NewRequest("GET", "projects/all", opt, options)
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

// GetProject gets a specific project, identified by project ID or
// NAMESPACE/PROJECT_NAME, which is owned by the authenticated user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#get-single-project
func (s *ProjectsService) GetProject(pid interface{}, options ...OptionFunc) (*Project, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	p := new(Project)
	resp, err := s.client.Do(req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}

// SearchProjectsOptions represents the available SearchProjects() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#search-for-projects-by-name
type SearchProjectsOptions struct {
	ListOptions
	OrderBy *string `url:"order_by,omitempty" json:"order_by,omitempty"`
	Sort    *string `url:"sort,omitempty" json:"sort,omitempty"`
}

// SearchProjects searches for projects by name which are accessible to the
// authenticated user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#search-for-projects-by-name
func (s *ProjectsService) SearchProjects(query string, opt *SearchProjectsOptions, options ...OptionFunc) ([]*Project, *Response, error) {
	u := fmt.Sprintf("projects/search/%s", query)

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

// ProjectEvent represents a GitLab project event.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#get-project-events
type ProjectEvent struct {
	Title          interface{} `json:"title"`
	ProjectID      int         `json:"project_id"`
	ActionName     string      `json:"action_name"`
	TargetID       interface{} `json:"target_id"`
	TargetType     interface{} `json:"target_type"`
	AuthorID       int         `json:"author_id"`
	AuthorUsername string      `json:"author_username"`
	Data           struct {
		Before            string      `json:"before"`
		After             string      `json:"after"`
		Ref               string      `json:"ref"`
		UserID            int         `json:"user_id"`
		UserName          string      `json:"user_name"`
		Repository        *Repository `json:"repository"`
		Commits           []*Commit   `json:"commits"`
		TotalCommitsCount int         `json:"total_commits_count"`
	} `json:"data"`
	TargetTitle interface{} `json:"target_title"`
}

func (s ProjectEvent) String() string {
	return Stringify(s)
}

// GetProjectEventsOptions represents the available GetProjectEvents() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#get-project-events
type GetProjectEventsOptions struct {
	ListOptions
}

// GetProjectEvents gets the events for the specified project. Sorted from
// newest to latest.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#get-project-events
func (s *ProjectsService) GetProjectEvents(pid interface{}, opt *GetProjectEventsOptions, options ...OptionFunc) ([]*ProjectEvent, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/events", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var p []*ProjectEvent
	resp, err := s.client.Do(req, &p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}

// CreateProjectOptions represents the available CreateProjects() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#create-project
type CreateProjectOptions struct {
	Name                                      *string               `url:"name,omitempty" json:"name,omitempty"`
	Path                                      *string               `url:"path,omitempty" json:"path,omitempty"`
	DefaultBranch                             *string               `url:"default_branch,omitempty" json:"default_branch,omitempty"` // note: this does not work
	NamespaceID                               *int                  `url:"namespace_id,omitempty" json:"namespace_id,omitempty"`
	Description                               *string               `url:"description,omitempty" json:"description,omitempty"`
	IssuesEnabled                             *bool                 `url:"issues_enabled,omitempty" json:"issues_enabled,omitempty"`
	MergeRequestsEnabled                      *bool                 `url:"merge_requests_enabled,omitempty" json:"merge_requests_enabled,omitempty"`
	BuildsEnabled                             *bool                 `url:"builds_enabled,omitempty" json:"builds_enabled,omitempty"`
	WikiEnabled                               *bool                 `url:"wiki_enabled,omitempty" json:"wiki_enabled,omitempty"`
	SnippetsEnabled                           *bool                 `url:"snippets_enabled,omitempty" json:"snippets_enabled,omitempty"`
	ContainerRegistryEnabled                  *bool                 `url:"container_registry_enabled,omitempty" json:"container_registry_enabled,omitempty"`
	SharedRunnersEnabled                      *bool                 `url:"shared_runners_enabled,omitempty" json:"shared_runners_enabled,omitempty"`
	Public                                    *bool                 `url:"public,omitempty" json:"public,omitempty"`
	VisibilityLevel                           *VisibilityLevelValue `url:"visibility_level,omitempty" json:"visibility_level,omitempty"`
	ImportURL                                 *string               `url:"import_url,omitempty" json:"import_url,omitempty"`
	PublicBuilds                              *bool                 `url:"public_builds,omitempty" json:"public_builds,omitempty"`
	OnlyAllowMergeIfBuildSucceeds             *bool                 `url:"only_allow_merge_if_build_succeeds,omitempty" json:"only_allow_merge_if_build_succeeds,omitempty"`
	OnlyAllowMergeIfAllDiscussionsAreResolved *bool                 `url:"only_allow_merge_if_all_discussions_are_resolved,omitempty" json:"only_allow_merge_if_all_discussions_are_resolved,omitempty"`
	LFSEnabled                                *bool                 `url:"lfs_enabled,omitempty" json:"lfs_enabled,omitempty"`
	RequestAccessEnabled                      *bool                 `url:"request_access_enabled,omitempty" json:"request_access_enabled,omitempty"`
}

// CreateProject creates a new project owned by the authenticated user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#create-project
func (s *ProjectsService) CreateProject(opt *CreateProjectOptions, options ...OptionFunc) (*Project, *Response, error) {
	req, err := s.client.NewRequest("POST", "projects", opt, options)
	if err != nil {
		return nil, nil, err
	}

	p := new(Project)
	resp, err := s.client.Do(req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}

// CreateProjectForUserOptions represents the available CreateProjectForUser()
// options; these are identical to those for an ordinary Project, since the
// required "user_id" parameter is provided in the request URL.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#create-project-for-user
type CreateProjectForUserOptions CreateProjectOptions

// CreateProjectForUser creates a new project owned by the specified user.
// Available only for admins.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#create-project-for-user
func (s *ProjectsService) CreateProjectForUser(user int, opt *CreateProjectForUserOptions, options ...OptionFunc) (*Project, *Response, error) {
	u := fmt.Sprintf("projects/user/%d", user)

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	p := new(Project)
	resp, err := s.client.Do(req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}

// EditProjectOptions represents the available EditProject() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#edit-project
type EditProjectOptions struct {
	Name                                      *string               `url:"name,omitempty" json:"name,omitempty"`
	Path                                      *string               `url:"path,omitempty" json:"path,omitempty"`
	Description                               *string               `url:"description,omitempty" json:"description,omitempty"`
	DefaultBranch                             *string               `url:"default_branch,omitempty" json:"default_branch,omitempty"`
	IssuesEnabled                             *bool                 `url:"issues_enabled,omitempty" json:"issues_enabled,omitempty"`
	MergeRequestsEnabled                      *bool                 `url:"merge_requests_enabled,omitempty" json:"merge_requests_enabled,omitempty"`
	ApprovalsBeforeMerge                      *int                  `url:"approvals_before_merge,omitempty" json:"approvals_before_merge,omitempty"`
	BuildsEnabled                             *bool                 `url:"builds_enabled,omitempty" json:"builds_enabled,omitempty"`
	WikiEnabled                               *bool                 `url:"wiki_enabled,omitempty" json:"wiki_enabled,omitempty"`
	SnippetsEnabled                           *bool                 `url:"snippets_enabled,omitempty" json:"snippets_enabled,omitempty"`
	ContainerRegistryEnabled                  *bool                 `url:"container_registry_enabled,omitempty" json:"container_registry_enabled,omitempty"`
	SharedRunnersEnabled                      *bool                 `url:"shared_runners_enabled,omitempty" json:"shared_runners_enabled,omitempty"`
	Public                                    *bool                 `url:"public,omitempty" json:"public,omitempty"`
	VisibilityLevel                           *VisibilityLevelValue `url:"visibility_level,omitempty" json:"visibility_level,omitempty"`
	ImportURL                                 *bool                 `url:"import_url,omitempty" json:"import_url,omitempty"`
	PublicBuilds                              *bool                 `url:"public_builds,omitempty" json:"public_builds,omitempty"`
	OnlyAllowMergeIfBuildSucceeds             *bool                 `url:"only_allow_merge_if_build_succeeds,omitempty" json:"only_allow_merge_if_build_succeeds,omitempty"`
	OnlyAllowMergeIfAllDiscussionsAreResolved *bool                 `url:"only_allow_merge_if_all_discussions_are_resolved,omitempty" json:"only_allow_merge_if_all_discussions_are_resolved,omitempty"`
	LFSEnabled                                *bool                 `url:"lfs_enabled,omitempty" json:"lfs_enabled,omitempty"`
	RequestAccessEnabled                      *bool                 `url:"request_access_enabled,omitempty" json:"request_access_enabled,omitempty"`
}

// EditProject updates an existing project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#edit-project
func (s *ProjectsService) EditProject(pid interface{}, opt *EditProjectOptions, options ...OptionFunc) (*Project, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s", url.QueryEscape(project))

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	p := new(Project)
	resp, err := s.client.Do(req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}

// ForkProject forks a project into the user namespace of the authenticated
// user.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#fork-project
func (s *ProjectsService) ForkProject(pid interface{}, options ...OptionFunc) (*Project, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/fork/%s", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	p := new(Project)
	resp, err := s.client.Do(req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}

// DeleteProject removes a project including all associated resources
// (issues, merge requests etc.)
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#remove-project
func (s *ProjectsService) DeleteProject(pid interface{}, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s", url.QueryEscape(project))

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// ProjectMember represents a project member.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#list-project-team-members
type ProjectMember struct {
	ID          int              `json:"id"`
	Username    string           `json:"username"`
	Email       string           `json:"email"`
	Name        string           `json:"name"`
	State       string           `json:"state"`
	CreatedAt   *time.Time       `json:"created_at"`
	AccessLevel AccessLevelValue `json:"access_level"`
}

// ListProjectMembersOptions represents the available ListProjectMembers()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#list-project-team-members
type ListProjectMembersOptions struct {
	ListOptions
	Query *string `url:"query,omitempty" json:"query,omitempty"`
}

// ListProjectMembers gets a list of a project's team members.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#list-project-team-members
func (s *ProjectsService) ListProjectMembers(pid interface{}, opt *ListProjectMembersOptions, options ...OptionFunc) ([]*ProjectMember, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/members", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var pm []*ProjectMember
	resp, err := s.client.Do(req, &pm)
	if err != nil {
		return nil, resp, err
	}

	return pm, resp, err
}

// GetProjectMember gets a project team member.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#get-project-team-member
func (s *ProjectsService) GetProjectMember(pid interface{}, user int, options ...OptionFunc) (*ProjectMember, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/members/%d", url.QueryEscape(project), user)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	pm := new(ProjectMember)
	resp, err := s.client.Do(req, pm)
	if err != nil {
		return nil, resp, err
	}

	return pm, resp, err
}

// AddProjectMemberOptions represents the available AddProjectMember() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#add-project-team-member
type AddProjectMemberOptions struct {
	UserID      *int              `url:"user_id,omitempty" json:"user_id,omitempty"`
	AccessLevel *AccessLevelValue `url:"access_level,omitempty" json:"access_level,omitempty"`
}

// AddProjectMember adds a user to a project team. This is an idempotent
// method and can be called multiple times with the same parameters. Adding
// team membership to a user that is already a member does not affect the
// existing membership.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#add-project-team-member
func (s *ProjectsService) AddProjectMember(pid interface{}, opt *AddProjectMemberOptions, options ...OptionFunc) (*ProjectMember, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/members", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	pm := new(ProjectMember)
	resp, err := s.client.Do(req, pm)
	if err != nil {
		return nil, resp, err
	}

	return pm, resp, err
}

// EditProjectMemberOptions represents the available EditProjectMember() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#edit-project-team-member
type EditProjectMemberOptions struct {
	AccessLevel *AccessLevelValue `url:"access_level,omitempty" json:"access_level,omitempty"`
}

// EditProjectMember updates a project team member to a specified access level..
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#edit-project-team-member
func (s *ProjectsService) EditProjectMember(pid interface{}, user int, opt *EditProjectMemberOptions, options ...OptionFunc) (*ProjectMember, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/members/%d", url.QueryEscape(project), user)

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	pm := new(ProjectMember)
	resp, err := s.client.Do(req, pm)
	if err != nil {
		return nil, resp, err
	}

	return pm, resp, err
}

// DeleteProjectMember removes a user from a project team.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#remove-project-team-member
func (s *ProjectsService) DeleteProjectMember(pid interface{}, user int, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/members/%d", url.QueryEscape(project), user)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// ProjectHook represents a project hook.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#list-project-hooks
type ProjectHook struct {
	ID                    int        `json:"id"`
	URL                   string     `json:"url"`
	ProjectID             int        `json:"project_id"`
	PushEvents            bool       `json:"push_events"`
	IssuesEvents          bool       `json:"issues_events"`
	MergeRequestsEvents   bool       `json:"merge_requests_events"`
	TagPushEvents         bool       `json:"tag_push_events"`
	NoteEvents            bool       `json:"note_events"`
	BuildEvents           bool       `json:"build_events"`
	PipelineEvents        bool       `json:"pipeline_events"`
	WikiPageEvents        bool       `json:"wiki_page_events"`
	EnableSSLVerification bool       `json:"enable_ssl_verification"`
	CreatedAt             *time.Time `json:"created_at"`
}

// ListProjectHooksOptions represents the available ListProjectHooks() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#list-project-hooks
type ListProjectHooksOptions struct {
	ListOptions
}

// ListProjectHooks gets a list of project hooks.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#list-project-hooks
func (s *ProjectsService) ListProjectHooks(pid interface{}, opt *ListProjectHooksOptions, options ...OptionFunc) ([]*ProjectHook, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/hooks", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var ph []*ProjectHook
	resp, err := s.client.Do(req, &ph)
	if err != nil {
		return nil, resp, err
	}

	return ph, resp, err
}

// GetProjectHook gets a specific hook for a project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#get-project-hook
func (s *ProjectsService) GetProjectHook(pid interface{}, hook int, options ...OptionFunc) (*ProjectHook, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/hooks/%d", url.QueryEscape(project), hook)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	ph := new(ProjectHook)
	resp, err := s.client.Do(req, ph)
	if err != nil {
		return nil, resp, err
	}

	return ph, resp, err
}

// AddProjectHookOptions represents the available AddProjectHook() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#add-project-hook
type AddProjectHookOptions struct {
	URL                   *string `url:"url,omitempty" json:"url,omitempty"`
	PushEvents            *bool   `url:"push_events,omitempty" json:"push_events,omitempty"`
	IssuesEvents          *bool   `url:"issues_events,omitempty" json:"issues_events,omitempty"`
	MergeRequestsEvents   *bool   `url:"merge_requests_events,omitempty" json:"merge_requests_events,omitempty"`
	TagPushEvents         *bool   `url:"tag_push_events,omitempty" json:"tag_push_events,omitempty"`
	NoteEvents            *bool   `url:"note_events,omitempty" json:"note_events,omitempty"`
	BuildEvents           *bool   `url:"build_events,omitempty" json:"build_events,omitempty"`
	PipelineEvents        *bool   `url:"pipeline_events,omitempty" json:"pipeline_events,omitempty"`
	WikiPageEvents        *bool   `url:"wiki_page_events,omitempty" json:"wiki_page_events,omitempty"`
	EnableSSLVerification *bool   `url:"enable_ssl_verification,omitempty" json:"enable_ssl_verification,omitempty"`
	Token                 *string `url:"token,omitempty" json:"token,omitempty"`
}

// AddProjectHook adds a hook to a specified project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#add-project-hook
func (s *ProjectsService) AddProjectHook(pid interface{}, opt *AddProjectHookOptions, options ...OptionFunc) (*ProjectHook, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/hooks", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	ph := new(ProjectHook)
	resp, err := s.client.Do(req, ph)
	if err != nil {
		return nil, resp, err
	}

	return ph, resp, err
}

// EditProjectHookOptions represents the available EditProjectHook() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#edit-project-hook
type EditProjectHookOptions struct {
	URL                   *string `url:"url,omitempty" json:"url,omitempty"`
	PushEvents            *bool   `url:"push_events,omitempty" json:"push_events,omitempty"`
	IssuesEvents          *bool   `url:"issues_events,omitempty" json:"issues_events,omitempty"`
	MergeRequestsEvents   *bool   `url:"merge_requests_events,omitempty" json:"merge_requests_events,omitempty"`
	TagPushEvents         *bool   `url:"tag_push_events,omitempty" json:"tag_push_events,omitempty"`
	NoteEvents            *bool   `url:"note_events,omitempty" json:"note_events,omitempty"`
	BuildEvents           *bool   `url:"build_events,omitempty" json:"build_events,omitempty"`
	PipelineEvents        *bool   `url:"pipeline_events,omitempty" json:"pipeline_events,omitempty"`
	WikiPageEvents        *bool   `url:"wiki_page_events,omitempty" json:"wiki_page_events,omitempty"`
	EnableSSLVerification *bool   `url:"enable_ssl_verification,omitempty" json:"enable_ssl_verification,omitempty"`
	Token                 *string `url:"token,omitempty" json:"token,omitempty"`
}

// EditProjectHook edits a hook for a specified project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#edit-project-hook
func (s *ProjectsService) EditProjectHook(pid interface{}, hook int, opt *EditProjectHookOptions, options ...OptionFunc) (*ProjectHook, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/hooks/%d", url.QueryEscape(project), hook)

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	ph := new(ProjectHook)
	resp, err := s.client.Do(req, ph)
	if err != nil {
		return nil, resp, err
	}

	return ph, resp, err
}

// DeleteProjectHook removes a hook from a project. This is an idempotent
// method and can be called multiple times. Either the hook is available or not.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#delete-project-hook
func (s *ProjectsService) DeleteProjectHook(pid interface{}, hook int, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/hooks/%d", url.QueryEscape(project), hook)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// BuildTrigger represents a project build trigger.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_triggers.md#build-triggers
type BuildTrigger struct {
	CreatedAt *time.Time `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at"`
	LastUsed  *time.Time `json:"last_used"`
	Token     string     `json:"token"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// ListBuildTriggersOptions represents the available ListBuildTriggers() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_triggers.md#list-project-triggers
type ListBuildTriggersOptions struct {
	ListOptions
}

// ListBuildTriggers gets a list of project triggers.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_triggers.md#list-project-triggers
func (s *ProjectsService) ListBuildTriggers(pid interface{}, opt *ListBuildTriggersOptions, options ...OptionFunc) ([]*BuildTrigger, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/triggers", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var bt []*BuildTrigger
	resp, err := s.client.Do(req, &bt)
	if err != nil {
		return nil, resp, err
	}

	return bt, resp, err
}

// GetBuildTrigger gets a specific build trigger for a project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_triggers.md#get-trigger-details
func (s *ProjectsService) GetBuildTrigger(pid interface{}, token string, options ...OptionFunc) (*BuildTrigger, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/triggers/%v", url.QueryEscape(project), token)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	bt := new(BuildTrigger)
	resp, err := s.client.Do(req, bt)
	if err != nil {
		return nil, resp, err
	}

	return bt, resp, err
}

// AddBuildTrigger adds a build trigger to a specified project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_triggers.md#create-a-project-trigger
func (s *ProjectsService) AddBuildTrigger(pid interface{}, options ...OptionFunc) (*BuildTrigger, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/triggers", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	bt := new(BuildTrigger)
	resp, err := s.client.Do(req, bt)
	if err != nil {
		return nil, resp, err
	}

	return bt, resp, err
}

// DeleteBuildTrigger removes a trigger from a project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/build_triggers.md#remove-a-project-trigger
func (s *ProjectsService) DeleteBuildTrigger(pid interface{}, token string, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/triggers/%s", url.QueryEscape(project), token)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// ProjectForkRelation represents a project fork relationship.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#admin-fork-relation
type ProjectForkRelation struct {
	ID                  int        `json:"id"`
	ForkedToProjectID   int        `json:"forked_to_project_id"`
	ForkedFromProjectID int        `json:"forked_from_project_id"`
	CreatedAt           *time.Time `json:"created_at"`
	UpdatedAt           *time.Time `json:"updated_at"`
}

// CreateProjectForkRelation creates a forked from/to relation between
// existing projects.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#create-a-forked-fromto-relation-between-existing-projects.
func (s *ProjectsService) CreateProjectForkRelation(pid int, fork int, options ...OptionFunc) (*ProjectForkRelation, *Response, error) {
	u := fmt.Sprintf("projects/%d/fork/%d", pid, fork)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	pfr := new(ProjectForkRelation)
	resp, err := s.client.Do(req, pfr)
	if err != nil {
		return nil, resp, err
	}

	return pfr, resp, err
}

// DeleteProjectForkRelation deletes an existing forked from relationship.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#delete-an-existing-forked-from-relationship
func (s *ProjectsService) DeleteProjectForkRelation(pid int, options ...OptionFunc) (*Response, error) {
	u := fmt.Sprintf("projects/%d/fork", pid)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// ArchiveProject archives the project if the user is either admin or the
// project owner of this project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#archive-a-project
func (s *ProjectsService) ArchiveProject(pid interface{}, options ...OptionFunc) (*Project, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/archive", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	p := new(Project)
	resp, err := s.client.Do(req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}

// UnarchiveProject unarchives the project if the user is either admin or
// the project owner of this project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/projects.md#unarchive-a-project
func (s *ProjectsService) UnarchiveProject(pid interface{}, options ...OptionFunc) (*Project, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/unarchive", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	p := new(Project)
	resp, err := s.client.Do(req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}
