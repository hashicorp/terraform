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
	"bytes"
	"fmt"
	"net/url"
	"time"
)

// ProjectSnippetsService handles communication with the project snippets
// related methods of the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/project_snippets.md
type ProjectSnippetsService struct {
	client *Client
}

// Snippet represents a GitLab project snippet.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/project_snippets.md
type Snippet struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	FileName string `json:"file_name"`
	Author   struct {
		ID        int        `json:"id"`
		Username  string     `json:"username"`
		Email     string     `json:"email"`
		Name      string     `json:"name"`
		State     string     `json:"state"`
		CreatedAt *time.Time `json:"created_at"`
	} `json:"author"`
	ExpiresAt *time.Time `json:"expires_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	CreatedAt *time.Time `json:"created_at"`
}

func (s Snippet) String() string {
	return Stringify(s)
}

// ListSnippetsOptions represents the available ListSnippets() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/project_snippets.md#list-snippets
type ListSnippetsOptions struct {
	ListOptions
}

// ListSnippets gets a list of project snippets.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/project_snippets.md#list-snippets
func (s *ProjectSnippetsService) ListSnippets(pid interface{}, opt *ListSnippetsOptions, options ...OptionFunc) ([]*Snippet, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/snippets", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var ps []*Snippet
	resp, err := s.client.Do(req, &ps)
	if err != nil {
		return nil, resp, err
	}

	return ps, resp, err
}

// GetSnippet gets a single project snippet
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/project_snippets.md#single-snippet
func (s *ProjectSnippetsService) GetSnippet(pid interface{}, snippet int, options ...OptionFunc) (*Snippet, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/snippets/%d", url.QueryEscape(project), snippet)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	ps := new(Snippet)
	resp, err := s.client.Do(req, ps)
	if err != nil {
		return nil, resp, err
	}

	return ps, resp, err
}

// CreateSnippetOptions represents the available CreateSnippet() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/project_snippets.md#create-new-snippet
type CreateSnippetOptions struct {
	Title           *string               `url:"title,omitempty" json:"title,omitempty"`
	FileName        *string               `url:"file_name,omitempty" json:"file_name,omitempty"`
	Code            *string               `url:"code,omitempty" json:"code,omitempty"`
	VisibilityLevel *VisibilityLevelValue `url:"visibility_level,omitempty" json:"visibility_level,omitempty"`
}

// CreateSnippet creates a new project snippet. The user must have permission
// to create new snippets.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/project_snippets.md#create-new-snippet
func (s *ProjectSnippetsService) CreateSnippet(pid interface{}, opt *CreateSnippetOptions, options ...OptionFunc) (*Snippet, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/snippets", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	ps := new(Snippet)
	resp, err := s.client.Do(req, ps)
	if err != nil {
		return nil, resp, err
	}

	return ps, resp, err
}

// UpdateSnippetOptions represents the available UpdateSnippet() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/project_snippets.md#update-snippet
type UpdateSnippetOptions struct {
	Title           *string               `url:"title,omitempty" json:"title,omitempty"`
	FileName        *string               `url:"file_name,omitempty" json:"file_name,omitempty"`
	Code            *string               `url:"code,omitempty" json:"code,omitempty"`
	VisibilityLevel *VisibilityLevelValue `url:"visibility_level,omitempty" json:"visibility_level,omitempty"`
}

// UpdateSnippet updates an existing project snippet. The user must have
// permission to change an existing snippet.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/project_snippets.md#update-snippet
func (s *ProjectSnippetsService) UpdateSnippet(pid interface{}, snippet int, opt *UpdateSnippetOptions, options ...OptionFunc) (*Snippet, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/snippets/%d", url.QueryEscape(project), snippet)

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	ps := new(Snippet)
	resp, err := s.client.Do(req, ps)
	if err != nil {
		return nil, resp, err
	}

	return ps, resp, err
}

// DeleteSnippet deletes an existing project snippet. This is an idempotent
// function and deleting a non-existent snippet still returns a 200 OK status
// code.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/project_snippets.md#delete-snippet
func (s *ProjectSnippetsService) DeleteSnippet(pid interface{}, snippet int, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/snippets/%d", url.QueryEscape(project), snippet)

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// SnippetContent returns the raw project snippet as plain text.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/project_snippets.md#snippet-content
func (s *ProjectSnippetsService) SnippetContent(pid interface{}, snippet int, options ...OptionFunc) ([]byte, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/snippets/%d/raw", url.QueryEscape(project), snippet)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var b bytes.Buffer
	resp, err := s.client.Do(req, &b)
	if err != nil {
		return nil, resp, err
	}

	return b.Bytes(), resp, err
}
