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
)

// LabelsService handles communication with the label related methods
// of the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/labels.md
type LabelsService struct {
	client *Client
}

// Label represents a GitLab label.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/labels.md
type Label struct {
	Name                   string `json:"name"`
	Color                  string `json:"color"`
	Description            string `json:"description"`
	OpenIssuesCount        int    `json:"open_issues_count"`
	ClosedIssuesCount      int    `json:"closed_issues_count"`
	OpenMergeRequestsCount int    `json:"open_merge_requests_count"`
}

func (l Label) String() string {
	return Stringify(l)
}

// ListLabels gets all labels for given project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/labels.md#list-labels
func (s *LabelsService) ListLabels(pid interface{}, options ...OptionFunc) ([]*Label, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/labels", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var l []*Label
	resp, err := s.client.Do(req, &l)
	if err != nil {
		return nil, resp, err
	}

	return l, resp, err
}

// CreateLabelOptions represents the available CreateLabel() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/labels.md#create-a-new-label
type CreateLabelOptions struct {
	Name        *string `url:"name,omitempty" json:"name,omitempty"`
	Color       *string `url:"color,omitempty" json:"color,omitempty"`
	Description *string `url:"description,omitempty" json:"description,omitempty"`
}

// CreateLabel creates a new label for given repository with given name and
// color.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/labels.md#create-a-new-label
func (s *LabelsService) CreateLabel(pid interface{}, opt *CreateLabelOptions, options ...OptionFunc) (*Label, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/labels", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	l := new(Label)
	resp, err := s.client.Do(req, l)
	if err != nil {
		return nil, resp, err
	}

	return l, resp, err
}

// DeleteLabelOptions represents the available DeleteLabel() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/labels.md#delete-a-label
type DeleteLabelOptions struct {
	Name *string `url:"name,omitempty" json:"name,omitempty"`
}

// DeleteLabel deletes a label given by its name.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/labels.md#delete-a-label
func (s *LabelsService) DeleteLabel(pid interface{}, opt *DeleteLabelOptions, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/labels", url.QueryEscape(project))

	req, err := s.client.NewRequest("DELETE", u, opt, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// UpdateLabelOptions represents the available UpdateLabel() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/labels.md#delete-a-label
type UpdateLabelOptions struct {
	Name        *string `url:"name,omitempty" json:"name,omitempty"`
	NewName     *string `url:"new_name,omitempty" json:"new_name,omitempty"`
	Color       *string `url:"color,omitempty" json:"color,omitempty"`
	Description *string `url:"description,omitempty" json:"description,omitempty"`
}

// UpdateLabel updates an existing label with new name or now color. At least
// one parameter is required, to update the label.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/labels.md#edit-an-existing-label
func (s *LabelsService) UpdateLabel(pid interface{}, opt *UpdateLabelOptions, options ...OptionFunc) (*Label, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/labels", url.QueryEscape(project))

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	l := new(Label)
	resp, err := s.client.Do(req, l)
	if err != nil {
		return nil, resp, err
	}

	return l, resp, err
}
