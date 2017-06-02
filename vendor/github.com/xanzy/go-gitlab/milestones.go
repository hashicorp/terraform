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

// MilestonesService handles communication with the milestone related methods
// of the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/milestones.md
type MilestonesService struct {
	client *Client
}

// Milestone represents a GitLab milestone.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/milestones.md
type Milestone struct {
	ID          int        `json:"id"`
	Iid         int        `json:"iid"`
	ProjectID   int        `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	StartDate   string     `json:"start_date"`
	DueDate     string     `json:"due_date"`
	State       string     `json:"state"`
	UpdatedAt   *time.Time `json:"updated_at"`
	CreatedAt   *time.Time `json:"created_at"`
}

func (m Milestone) String() string {
	return Stringify(m)
}

// ListMilestonesOptions represents the available ListMilestones() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/milestones.md#list-project-milestones
type ListMilestonesOptions struct {
	ListOptions
	IID *int `url:"iid,omitempty" json:"iid,omitempty"`
}

// ListMilestones returns a list of project milestones.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/milestones.md#list-project-milestones
func (s *MilestonesService) ListMilestones(pid interface{}, opt *ListMilestonesOptions, options ...OptionFunc) ([]*Milestone, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/milestones", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var m []*Milestone
	resp, err := s.client.Do(req, &m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// GetMilestone gets a single project milestone.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/milestones.md#get-single-milestone
func (s *MilestonesService) GetMilestone(pid interface{}, milestone int, options ...OptionFunc) (*Milestone, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/milestones/%d", url.QueryEscape(project), milestone)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	m := new(Milestone)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// CreateMilestoneOptions represents the available CreateMilestone() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/milestones.md#create-new-milestone
type CreateMilestoneOptions struct {
	Title       *string `url:"title,omitempty" json:"title,omitempty"`
	Description *string `url:"description,omitempty" json:"description,omitempty"`
	StartDate   *string `url:"start_date,omitempty" json:"start_date,omitempty"`
	DueDate     *string `url:"due_date,omitempty" json:"due_date,omitempty"`
}

// CreateMilestone creates a new project milestone.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/milestones.md#create-new-milestone
func (s *MilestonesService) CreateMilestone(pid interface{}, opt *CreateMilestoneOptions, options ...OptionFunc) (*Milestone, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/milestones", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	m := new(Milestone)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// UpdateMilestoneOptions represents the available UpdateMilestone() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/milestones.md#edit-milestone
type UpdateMilestoneOptions struct {
	Title       *string `url:"title,omitempty" json:"title,omitempty"`
	Description *string `url:"description,omitempty" json:"description,omitempty"`
	StartDate   *string `url:"start_date,omitempty" json:"start_date,omitempty"`
	DueDate     *string `url:"due_date,omitempty" json:"due_date,omitempty"`
	StateEvent  *string `url:"state_event,omitempty" json:"state_event,omitempty"`
}

// UpdateMilestone updates an existing project milestone.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/milestones.md#edit-milestone
func (s *MilestonesService) UpdateMilestone(pid interface{}, milestone int, opt *UpdateMilestoneOptions, options ...OptionFunc) (*Milestone, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/milestones/%d", url.QueryEscape(project), milestone)

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	m := new(Milestone)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// GetMilestoneIssuesOptions represents the available GetMilestoneIssues() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/milestones.md#get-all-issues-assigned-to-a-single-milestone
type GetMilestoneIssuesOptions struct {
	ListOptions
}

// GetMilestoneIssues gets all issues assigned to a single project milestone.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/milestones.md#get-all-issues-assigned-to-a-single-milestone
func (s *MilestonesService) GetMilestoneIssues(pid interface{}, milestone int, opt *GetMilestoneIssuesOptions, options ...OptionFunc) ([]*Issue, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/milestones/%d/issues", url.QueryEscape(project), milestone)

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var i []*Issue
	resp, err := s.client.Do(req, &i)
	if err != nil {
		return nil, resp, err
	}

	return i, resp, err
}
