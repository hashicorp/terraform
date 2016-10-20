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

// MergeRequestsService handles communication with the merge requests related
// methods of the GitLab API.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/merge_requests.html
type MergeRequestsService struct {
	client *Client
}

// MergeRequest represents a GitLab merge request.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/merge_requests.html
type MergeRequest struct {
	ID             int        `json:"id"`
	IID            int        `json:"iid"`
	ProjectID      int        `json:"project_id"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	WorkInProgress bool       `json:"work_in_progress"`
	State          string     `json:"state"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
	TargetBranch   string     `json:"target_branch"`
	SourceBranch   string     `json:"source_branch"`
	Upvotes        int        `json:"upvotes"`
	Downvotes      int        `json:"downvotes"`
	Author         struct {
		Name      string `json:"name"`
		Username  string `json:"username"`
		ID        int    `json:"id"`
		State     string `json:"state"`
		AvatarURL string `json:"avatar_url"`
	} `json:"author"`
	Assignee struct {
		Name      string `json:"name"`
		Username  string `json:"username"`
		ID        int    `json:"id"`
		State     string `json:"state"`
		AvatarURL string `json:"avatar_url"`
	} `json:"assignee"`
	SourceProjectID int      `json:"source_project_id"`
	TargetProjectID int      `json:"target_project_id"`
	Labels          []string `json:"labels"`
	Milestone       struct {
		ID          int        `json:"id"`
		Iid         int        `json:"iid"`
		ProjectID   int        `json:"project_id"`
		Title       string     `json:"title"`
		Description string     `json:"description"`
		State       string     `json:"state"`
		CreatedAt   *time.Time `json:"created_at"`
		UpdatedAt   *time.Time `json:"updated_at"`
		DueDate     string     `json:"due_date"`
	} `json:"milestone"`
	MergeWhenBuildSucceeds  bool   `json:"merge_when_build_succeeds"`
	MergeStatus             string `json:"merge_status"`
	Subscribed              bool   `json:"subscribed"`
	UserNotesCount          int    `json:"user_notes_count"`
	SouldRemoveSourceBranch bool   `json:"should_remove_source_branch"`
	ForceRemoveSourceBranch bool   `json:"force_remove_source_branch"`
	Changes                 []struct {
		OldPath     string `json:"old_path"`
		NewPath     string `json:"new_path"`
		AMode       string `json:"a_mode"`
		BMode       string `json:"b_mode"`
		Diff        string `json:"diff"`
		NewFile     bool   `json:"new_file"`
		RenamedFile bool   `json:"renamed_file"`
		DeletedFile bool   `json:"deleted_file"`
	} `json:"changes"`
}

func (m MergeRequest) String() string {
	return Stringify(m)
}

// ListMergeRequestsOptions represents the available ListMergeRequests()
// options.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/merge_requests.html#list-merge-requests
type ListMergeRequestsOptions struct {
	ListOptions
	IID     *int    `url:"iid,omitempty" json:"iid,omitempty"`
	State   *string `url:"state,omitempty" json:"state,omitempty"`
	OrderBy *string `url:"order_by,omitempty" json:"order_by,omitempty"`
	Sort    *string `url:"sort,omitempty" json:"sort,omitempty"`
}

// ListMergeRequests gets all merge requests for this project. The state
// parameter can be used to get only merge requests with a given state (opened,
// closed, or merged) or all of them (all). The pagination parameters page and
// per_page can be used to restrict the list of merge requests.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/merge_requests.html#list-merge-requests
func (s *MergeRequestsService) ListMergeRequests(
	pid interface{},
	opt *ListMergeRequestsOptions) ([]*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt)
	if err != nil {
		return nil, nil, err
	}

	var m []*MergeRequest
	resp, err := s.client.Do(req, &m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// GetMergeRequest shows information about a single merge request.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/merge_requests.html#get-single-mr
func (s *MergeRequestsService) GetMergeRequest(
	pid interface{},
	mergeRequest int) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_request/%d", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// GetMergeRequestChanges shows information about the merge request including
// its files and changes.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/merge_requests.html#get-single-mr-changes
func (s *MergeRequestsService) GetMergeRequestChanges(
	pid interface{},
	mergeRequest int) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_request/%d/changes", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// CreateMergeRequestOptions represents the available CreateMergeRequest()
// options.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/merge_requests.html#create-mr
type CreateMergeRequestOptions struct {
	Title           *string `url:"title,omitempty" json:"title,omitempty"`
	Description     *string `url:"description,omitempty" json:"description,omitempty"`
	SourceBranch    *string `url:"source_branch,omitemtpy" json:"source_branch,omitemtpy"`
	TargetBranch    *string `url:"target_branch,omitemtpy" json:"target_branch,omitemtpy"`
	AssigneeID      *int    `url:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	TargetProjectID *int    `url:"target_project_id,omitempty" json:"target_project_id,omitempty"`
}

// CreateMergeRequest creates a new merge request.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/merge_requests.html#create-mr
func (s *MergeRequestsService) CreateMergeRequest(
	pid interface{},
	opt *CreateMergeRequestOptions) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// UpdateMergeRequestOptions represents the available UpdateMergeRequest()
// options.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/merge_requests.html#update-mr
type UpdateMergeRequestOptions struct {
	Title        *string `url:"title,omitempty" json:"title,omitempty"`
	Description  *string `url:"description,omitempty" json:"description,omitempty"`
	TargetBranch *string `url:"target_branch,omitemtpy" json:"target_branch,omitemtpy"`
	AssigneeID   *int    `url:"assignee_id,omitempty" json:"assignee_id,omitempty"`
	StateEvent   *string `url:"state_event,omitempty" json:"state_event,omitempty"`
}

// UpdateMergeRequest updates an existing project milestone.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/merge_requests.html#update-mr
func (s *MergeRequestsService) UpdateMergeRequest(
	pid interface{},
	mergeRequest int,
	opt *UpdateMergeRequestOptions) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_request/%d", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("PUT", u, opt)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// AcceptMergeRequest merges changes submitted with MR using this API. If merge
// success you get 200 OK. If it has some conflicts and can not be merged - you
// get 405 and error message 'Branch cannot be merged'. If merge request is
// already merged or closed - you get 405 and error message 'Method Not Allowed'
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/merge_requests.html#accept-mr
func (s *MergeRequestsService) AcceptMergeRequest(
	pid interface{},
	mergeRequest int) (*MergeRequest, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_request/%d/merge", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("PUT", u, nil)
	if err != nil {
		return nil, nil, err
	}

	m := new(MergeRequest)
	resp, err := s.client.Do(req, m)
	if err != nil {
		return nil, resp, err
	}

	return m, resp, err
}

// MergeRequestComment represents a GitLab merge request comment.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/merge_requests.html
type MergeRequestComment struct {
	Note   string `json:"note"`
	Author struct {
		ID        int        `json:"id"`
		Username  string     `json:"username"`
		Email     string     `json:"email"`
		Name      string     `json:"name"`
		State     string     `json:"state"`
		CreatedAt *time.Time `json:"created_at"`
	} `json:"author"`
}

func (m MergeRequestComment) String() string {
	return Stringify(m)
}

// GetMergeRequestCommentsOptions represents the available GetMergeRequestComments()
// options.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/merge_requests.html#get-the-comments-on-a-mr
type GetMergeRequestCommentsOptions struct {
	ListOptions
}

// GetMergeRequestComments gets all the comments associated with a merge
// request.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/merge_requests.html#get-the-comments-on-a-mr
func (s *MergeRequestsService) GetMergeRequestComments(
	pid interface{},
	mergeRequest int,
	opt *GetMergeRequestCommentsOptions) ([]*MergeRequestComment, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_request/%d/comments", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("GET", u, opt)
	if err != nil {
		return nil, nil, err
	}

	var c []*MergeRequestComment
	resp, err := s.client.Do(req, &c)
	if err != nil {
		return nil, resp, err
	}

	return c, resp, err
}

// PostMergeRequestCommentOptions represents the available
// PostMergeRequestComment() options.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/commits.html#post-comment-to-mr
type PostMergeRequestCommentOptions struct {
	Note *string `url:"note,omitempty" json:"note,omitempty"`
}

// PostMergeRequestComment dds a comment to a merge request.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/commits.html#post-comment-to-mr
func (s *MergeRequestsService) PostMergeRequestComment(
	pid interface{},
	mergeRequest int,
	opt *PostMergeRequestCommentOptions) (*MergeRequestComment, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_request/%d/comments", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("POST", u, opt)
	if err != nil {
		return nil, nil, err
	}

	c := new(MergeRequestComment)
	resp, err := s.client.Do(req, c)
	if err != nil {
		return nil, resp, err
	}

	return c, resp, err
}
