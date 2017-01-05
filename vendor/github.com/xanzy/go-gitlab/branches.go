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

// BranchesService handles communication with the branch related methods
// of the GitLab API.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/branches.html
type BranchesService struct {
	client *Client
}

// Branch represents a GitLab branch.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/branches.html
type Branch struct {
	Commit    *Commit `json:"commit"`
	Name      string  `json:"name"`
	Protected bool    `json:"protected"`
}

func (b Branch) String() string {
	return Stringify(b)
}

// ListBranches gets a list of repository branches from a project, sorted by
// name alphabetically.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/branches.html#list-repository-branches
func (s *BranchesService) ListBranches(pid interface{}) ([]*Branch, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/branches", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var b []*Branch
	resp, err := s.client.Do(req, &b)
	if err != nil {
		return nil, resp, err
	}

	return b, resp, err
}

// GetBranch gets a single project repository branch.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/branches.html#get-single-repository-branch
func (s *BranchesService) GetBranch(pid interface{}, branch string) (*Branch, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/branches/%s", url.QueryEscape(project), branch)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	b := new(Branch)
	resp, err := s.client.Do(req, b)
	if err != nil {
		return nil, resp, err
	}

	return b, resp, err
}

// ProtectBranch protects a single project repository branch. This is an
// idempotent function, protecting an already protected repository branch
// still returns a 200 OK status code.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/branches.html#protect-repository-branch
func (s *BranchesService) ProtectBranch(pid interface{}, branch string) (*Branch, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/branches/%s/protect", url.QueryEscape(project), branch)

	req, err := s.client.NewRequest("PUT", u, nil)
	if err != nil {
		return nil, nil, err
	}

	b := new(Branch)
	resp, err := s.client.Do(req, b)
	if err != nil {
		return nil, resp, err
	}

	return b, resp, err
}

// UnprotectBranch unprotects a single project repository branch. This is an
// idempotent function, unprotecting an already unprotected repository branch
// still returns a 200 OK status code.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/branches.html#unprotect-repository-branch
func (s *BranchesService) UnprotectBranch(
	pid interface{},
	branch string) (*Branch, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/branches/%s/unprotect", url.QueryEscape(project), branch)

	req, err := s.client.NewRequest("PUT", u, nil)
	if err != nil {
		return nil, nil, err
	}

	b := new(Branch)
	resp, err := s.client.Do(req, b)
	if err != nil {
		return nil, resp, err
	}

	return b, resp, err
}

// CreateBranchOptions represents the available CreateBranch() options.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/branches.html#create-repository-branch
type CreateBranchOptions struct {
	BranchName *string `url:"branch_name,omitempty" json:"branch_name,omitempty"`
	Ref        *string `url:"ref,omitempty" json:"ref,omitempty"`
}

// CreateBranch creates branch from commit SHA or existing branch.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/branches.html#create-repository-branch
func (s *BranchesService) CreateBranch(
	pid interface{},
	opt *CreateBranchOptions) (*Branch, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/branches", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt)
	if err != nil {
		return nil, nil, err
	}

	b := new(Branch)
	resp, err := s.client.Do(req, b)
	if err != nil {
		return nil, resp, err
	}

	return b, resp, err
}

// DeleteBranch deletes an existing branch.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/branches.html#delete-repository-branch
func (s *BranchesService) DeleteBranch(pid interface{}, branch string) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/branches/%s", url.QueryEscape(project), branch)

	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}
