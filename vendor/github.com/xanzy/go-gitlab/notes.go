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

// NotesService handles communication with the notes related methods
// of the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md
type NotesService struct {
	client *Client
}

// Note represents a GitLab note.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md
type Note struct {
	ID         int    `json:"id"`
	Body       string `json:"body"`
	Attachment string `json:"attachment"`
	Title      string `json:"title"`
	FileName   string `json:"file_name"`
	Author     struct {
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

func (n Note) String() string {
	return Stringify(n)
}

// ListIssueNotesOptions represents the available ListIssueNotes() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#list-project-issue-notes
type ListIssueNotesOptions struct {
	ListOptions
}

// ListIssueNotes gets a list of all notes for a single issue.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#list-project-issue-notes
func (s *NotesService) ListIssueNotes(pid interface{}, issue int, opt *ListIssueNotesOptions, options ...OptionFunc) ([]*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/issues/%d/notes", url.QueryEscape(project), issue)

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	var n []*Note
	resp, err := s.client.Do(req, &n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}

// GetIssueNote returns a single note for a specific project issue.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#get-single-issue-note
func (s *NotesService) GetIssueNote(pid interface{}, issue int, note int, options ...OptionFunc) (*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/issues/%d/notes/%d", url.QueryEscape(project), issue, note)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	n := new(Note)
	resp, err := s.client.Do(req, n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}

// CreateIssueNoteOptions represents the available CreateIssueNote()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#create-new-issue-note
type CreateIssueNoteOptions struct {
	Body *string `url:"body,omitempty" json:"body,omitempty"`
}

// CreateIssueNote creates a new note to a single project issue.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#create-new-issue-note
func (s *NotesService) CreateIssueNote(pid interface{}, issue int, opt *CreateIssueNoteOptions, options ...OptionFunc) (*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/issues/%d/notes", url.QueryEscape(project), issue)

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	n := new(Note)
	resp, err := s.client.Do(req, n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}

// UpdateIssueNoteOptions represents the available UpdateIssueNote()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#modify-existing-issue-note
type UpdateIssueNoteOptions struct {
	Body *string `url:"body,omitempty" json:"body,omitempty"`
}

// UpdateIssueNote modifies existing note of an issue.
//
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#modify-existing-issue-note
func (s *NotesService) UpdateIssueNote(pid interface{}, issue int, note int, opt *UpdateIssueNoteOptions, options ...OptionFunc) (*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/issues/%d/notes/%d", url.QueryEscape(project), issue, note)

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	n := new(Note)
	resp, err := s.client.Do(req, n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}

// ListSnippetNotes gets a list of all notes for a single snippet. Snippet
// notes are comments users can post to a snippet.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#list-all-snippet-notes
func (s *NotesService) ListSnippetNotes(pid interface{}, snippet int, options ...OptionFunc) ([]*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/snippets/%d/notes", url.QueryEscape(project), snippet)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var n []*Note
	resp, err := s.client.Do(req, &n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}

// GetSnippetNote returns a single note for a given snippet.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#get-single-snippet-note
func (s *NotesService) GetSnippetNote(pid interface{}, snippet int, note int, options ...OptionFunc) (*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/snippets/%d/notes/%d", url.QueryEscape(project), snippet, note)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	n := new(Note)
	resp, err := s.client.Do(req, n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}

// CreateSnippetNoteOptions represents the available CreateSnippetNote()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#create-new-snippet-note
type CreateSnippetNoteOptions struct {
	Body *string `url:"body,omitempty" json:"body,omitempty"`
}

// CreateSnippetNote creates a new note for a single snippet. Snippet notes are
// comments users can post to a snippet.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#create-new-snippet-note
func (s *NotesService) CreateSnippetNote(pid interface{}, snippet int, opt *CreateSnippetNoteOptions, options ...OptionFunc) (*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/snippets/%d/notes", url.QueryEscape(project), snippet)

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	n := new(Note)
	resp, err := s.client.Do(req, n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}

// UpdateSnippetNoteOptions represents the available UpdateSnippetNote()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#modify-existing-snippet-note
type UpdateSnippetNoteOptions struct {
	Body *string `url:"body,omitempty" json:"body,omitempty"`
}

// UpdateSnippetNote modifies existing note of a snippet.
//
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#modify-existing-snippet-note
func (s *NotesService) UpdateSnippetNote(pid interface{}, snippet int, note int, opt *UpdateSnippetNoteOptions, options ...OptionFunc) (*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/snippets/%d/notes/%d", url.QueryEscape(project), snippet, note)

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	n := new(Note)
	resp, err := s.client.Do(req, n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}

// ListMergeRequestNotes gets a list of all notes for a single merge request.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#list-all-merge-request-notes
func (s *NotesService) ListMergeRequestNotes(pid interface{}, mergeRequest int, options ...OptionFunc) ([]*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/notes", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var n []*Note
	resp, err := s.client.Do(req, &n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}

// GetMergeRequestNote returns a single note for a given merge request.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#get-single-merge-request-note
func (s *NotesService) GetMergeRequestNote(pid interface{}, mergeRequest int, note int, options ...OptionFunc) (*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/notes/%d", url.QueryEscape(project), mergeRequest, note)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	n := new(Note)
	resp, err := s.client.Do(req, n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}

// CreateMergeRequestNoteOptions represents the available
// CreateMergeRequestNote() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#create-new-merge-request-note
type CreateMergeRequestNoteOptions struct {
	Body *string `url:"body,omitempty" json:"body,omitempty"`
}

// CreateMergeRequestNote creates a new note for a single merge request.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#create-new-merge-request-note
func (s *NotesService) CreateMergeRequestNote(pid interface{}, mergeRequest int, opt *CreateMergeRequestNoteOptions, options ...OptionFunc) (*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/merge_requests/%d/notes", url.QueryEscape(project), mergeRequest)

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	n := new(Note)
	resp, err := s.client.Do(req, n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}

// UpdateMergeRequestNoteOptions represents the available
// UpdateMergeRequestNote() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#modify-existing-merge-request-note
type UpdateMergeRequestNoteOptions struct {
	Body *string `url:"body,omitempty" json:"body,omitempty"`
}

// UpdateMergeRequestNote modifies existing note of a merge request.
//
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/notes.md#modify-existing-merge-request-note
func (s *NotesService) UpdateMergeRequestNote(pid interface{}, mergeRequest int, note int, opt *UpdateMergeRequestNoteOptions, options ...OptionFunc) (*Note, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf(
		"projects/%s/merge_requests/%d/notes/%d", url.QueryEscape(project), mergeRequest, note)
	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	n := new(Note)
	resp, err := s.client.Do(req, n)
	if err != nil {
		return nil, resp, err
	}

	return n, resp, err
}
