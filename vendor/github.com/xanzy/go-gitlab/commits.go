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

// CommitsService handles communication with the commit related methods
// of the GitLab API.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html
type CommitsService struct {
	client *Client
}

// Commit represents a GitLab commit.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html
type Commit struct {
	ID            string     `json:"id"`
	ShortID       string     `json:"short_id"`
	Title         string     `json:"title"`
	AuthorName    string     `json:"author_name"`
	AuthorEmail   string     `json:"author_email"`
	AuthoredDate  *time.Time `json:"authored_date"`
	CommittedDate *time.Time `json:"committed_date"`
	CreatedAt     *time.Time `json:"created_at"`
	Message       string     `json:"message"`
	ParentsIds    []string   `json:"parents_ids"`
}

func (c Commit) String() string {
	return Stringify(c)
}

// ListCommitsOptions represents the available ListCommits() options.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html#list-commits
type ListCommitsOptions struct {
	ListOptions
	RefName *string `url:"ref_name,omitempty" json:"ref_name,omitempty"`
}

// ListCommits gets a list of repository commits in a project.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html#list-commits
func (s *CommitsService) ListCommits(
	pid interface{},
	opt *ListCommitsOptions) ([]*Commit, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/commits", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt)
	if err != nil {
		return nil, nil, err
	}

	var c []*Commit
	resp, err := s.client.Do(req, &c)
	if err != nil {
		return nil, resp, err
	}

	return c, resp, err
}

// GetCommit gets a specific commit identified by the commit hash or name of a
// branch or tag.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html#get-a-single-commit
func (s *CommitsService) GetCommit(
	pid interface{},
	sha string) (*Commit, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/commits/%s", url.QueryEscape(project), sha)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	c := new(Commit)
	resp, err := s.client.Do(req, c)
	if err != nil {
		return nil, resp, err
	}

	return c, resp, err
}

// Diff represents a GitLab diff.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html
type Diff struct {
	Diff        string `json:"diff"`
	NewPath     string `json:"new_path"`
	OldPath     string `json:"old_path"`
	AMode       string `json:"a_mode"`
	BMode       string `json:"b_mode"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
}

func (d Diff) String() string {
	return Stringify(d)
}

// GetCommitDiff gets the diff of a commit in a project..
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/commits.html#get-the-diff-of-a-commit
func (s *CommitsService) GetCommitDiff(
	pid interface{},
	sha string) ([]*Diff, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/commits/%s/diff", url.QueryEscape(project), sha)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var d []*Diff
	resp, err := s.client.Do(req, &d)
	if err != nil {
		return nil, resp, err
	}

	return d, resp, err
}

// CommitComment represents a GitLab commit comment.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html
type CommitComment struct {
	Note     string `json:"note"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	LineType string `json:"line_type"`
	Author   Author `json:"author"`
}

type Author struct {
	ID        int        `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	State     string     `json:"state"`
	Blocked   bool       `json:"blocked"`
	CreatedAt *time.Time `json:"created_at"`
}

func (c CommitComment) String() string {
	return Stringify(c)
}

// GetCommitComments gets the comments of a commit in a project.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/commits.html#get-the-comments-of-a-commit
func (s *CommitsService) GetCommitComments(
	pid interface{},
	sha string) ([]*CommitComment, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/commits/%s/comments", url.QueryEscape(project), sha)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	var c []*CommitComment
	resp, err := s.client.Do(req, &c)
	if err != nil {
		return nil, resp, err
	}

	return c, resp, err
}

// PostCommitCommentOptions represents the available PostCommitComment()
// options.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/commits.html#post-comment-to-commit
type PostCommitCommentOptions struct {
	Note     *string `url:"note,omitempty" json:"note,omitempty"`
	Path     *string `url:"path" json:"path"`
	Line     *int    `url:"line" json:"line"`
	LineType *string `url:"line_type" json:"line_type"`
}

// PostCommitComment adds a comment to a commit. Optionally you can post
// comments on a specific line of a commit. Therefor both path, line_new and
// line_old are required.
//
// GitLab API docs:
// http://doc.gitlab.com/ce/api/commits.html#post-comment-to-commit
func (s *CommitsService) PostCommitComment(
	pid interface{},
	sha string,
	opt *PostCommitCommentOptions) (*CommitComment, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/commits/%s/comments", url.QueryEscape(project), sha)

	req, err := s.client.NewRequest("POST", u, opt)
	if err != nil {
		return nil, nil, err
	}

	c := new(CommitComment)
	resp, err := s.client.Do(req, c)
	if err != nil {
		return nil, resp, err
	}

	return c, resp, err
}

// GetCommitStatusesOptions represents the available GetCommitStatuses() options.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html#get-the-status-of-a-commit
type GetCommitStatusesOptions struct {
	Ref   *string `url:"ref,omitempty" json:"ref,omitempty"`
	Stage *string `url:"stage,omitempty" json:"stage,omitempty"`
	Name  *string `url:"name,omitempty" json:"name,omitempty"`
	All   *bool   `url:"all,omitempty" json:"all,omitempty"`
}

// CommitStatus represents a GitLab commit status.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html#get-the-status-of-a-commit
type CommitStatus struct {
	ID          int        `json:"id"`
	SHA         string     `json:"sha"`
	Ref         string     `json:"ref"`
	Status      string     `json:"status"`
	Name        string     `json:"name"`
	TargetUrl   string     `json:"target_url"`
	Description string     `json:"description"`
	CreatedAt   *time.Time `json:"created_at"`
	StartedAt   *time.Time `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
	Author      Author     `json:"author"`
}

// GetCommitStatuses gets the statuses of a commit in a project.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html#get-the-status-of-a-commit
func (s *CommitsService) GetCommitStatuses(
	pid interface{},
	sha string,
	opt *GetCommitStatusesOptions) ([]*CommitStatus, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/commits/%s/statuses", url.QueryEscape(project), sha)

	req, err := s.client.NewRequest("GET", u, opt)
	if err != nil {
		return nil, nil, err
	}

	var cs []*CommitStatus
	resp, err := s.client.Do(req, &cs)
	if err != nil {
		return nil, resp, err
	}

	return cs, resp, err
}

// SetCommitStatusOptions represents the available SetCommitStatus() options.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html#post-the-status-to-commit
type SetCommitStatusOptions struct {
	State       BuildState `url:"state" json:"state"`
	Ref         *string    `url:"ref,omitempty" json:"ref,omitempty"`
	Name        *string    `url:"name,omitempty" json:"name,omitempty"`
	Context     *string    `url:"context,omitempty" json:"context,omitempty"`
	TargetUrl   *string    `url:"target_url,omitempty" json:"target_url,omitempty"`
	Description *string    `url:"description,omitempty" json:"description,omitempty"`
}

type BuildState string

const (
	Pending  BuildState = "pending"
	Running  BuildState = "running"
	Success  BuildState = "success"
	Failed   BuildState = "failed"
	Canceled BuildState = "canceled"
)

// SetCommitStatus sets the status of a commit in a project.
//
// GitLab API docs: http://doc.gitlab.com/ce/api/commits.html#post-the-status-to-commit
func (s *CommitsService) SetCommitStatus(
	pid interface{},
	sha string,
	opt *SetCommitStatusOptions) (*CommitStatus, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/statuses/%s", url.QueryEscape(project), sha)

	req, err := s.client.NewRequest("POST", u, opt)
	if err != nil {
		return nil, nil, err
	}

	var cs *CommitStatus
	resp, err := s.client.Do(req, &cs)
	if err != nil {
		return nil, resp, err
	}

	return cs, resp, err
}
