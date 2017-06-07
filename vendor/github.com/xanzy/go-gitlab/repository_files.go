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

// RepositoryFilesService handles communication with the repository files
// related methods of the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/repository_files.md
type RepositoryFilesService struct {
	client *Client
}

// File represents a GitLab repository file.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/repository_files.md
type File struct {
	FileName string `json:"file_name"`
	FilePath string `json:"file_path"`
	Size     int    `json:"size"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
	Ref      string `json:"ref"`
	BlobID   string `json:"blob_id"`
	CommitID string `json:"commit_id"`
}

func (r File) String() string {
	return Stringify(r)
}

// GetFileOptions represents the available GetFile() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/repository_files.md#get-file-from-respository
type GetFileOptions struct {
	FilePath *string `url:"file_path,omitempty" json:"file_path,omitempty"`
	Ref      *string `url:"ref,omitempty" json:"ref,omitempty"`
}

// GetFile allows you to receive information about a file in repository like
// name, size, content. Note that file content is Base64 encoded.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/repository_files.md#get-file-from-respository
func (s *RepositoryFilesService) GetFile(pid interface{}, opt *GetFileOptions, options ...OptionFunc) (*File, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/files", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	f := new(File)
	resp, err := s.client.Do(req, f)
	if err != nil {
		return nil, resp, err
	}

	return f, resp, err
}

// FileInfo represents file details of a GitLab repository file.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/repository_files.md
type FileInfo struct {
	FilePath   string `json:"file_path"`
	BranchName string `json:"branch_name"`
}

func (r FileInfo) String() string {
	return Stringify(r)
}

// CreateFileOptions represents the available CreateFile() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/repository_files.md#create-new-file-in-repository
type CreateFileOptions struct {
	FilePath      *string `url:"file_path,omitempty" json:"file_path,omitempty"`
	BranchName    *string `url:"branch_name,omitempty" json:"branch_name,omitempty"`
	Encoding      *string `url:"encoding,omitempty" json:"encoding,omitempty"`
	AuthorEmail   *string `url:"author_email,omitempty" json:"author_email,omitempty"`
	AuthorName    *string `url:"author_name,omitempty" json:"author_name,omitempty"`
	Content       *string `url:"content,omitempty" json:"content,omitempty"`
	CommitMessage *string `url:"commit_message,omitempty" json:"commit_message,omitempty"`
}

// CreateFile creates a new file in a repository.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/repository_files.md#create-new-file-in-repository
func (s *RepositoryFilesService) CreateFile(pid interface{}, opt *CreateFileOptions, options ...OptionFunc) (*FileInfo, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/files", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	f := new(FileInfo)
	resp, err := s.client.Do(req, f)
	if err != nil {
		return nil, resp, err
	}

	return f, resp, err
}

// UpdateFileOptions represents the available UpdateFile() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/repository_files.md#update-existing-file-in-repository
type UpdateFileOptions struct {
	FilePath      *string `url:"file_path,omitempty" json:"file_path,omitempty"`
	BranchName    *string `url:"branch_name,omitempty" json:"branch_name,omitempty"`
	Encoding      *string `url:"encoding,omitempty" json:"encoding,omitempty"`
	AuthorEmail   *string `url:"author_email,omitempty" json:"author_email,omitempty"`
	AuthorName    *string `url:"author_name,omitempty" json:"author_name,omitempty"`
	Content       *string `url:"content,omitempty" json:"content,omitempty"`
	CommitMessage *string `url:"commit_message,omitempty" json:"commit_message,omitempty"`
}

// UpdateFile updates an existing file in a repository
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/repository_files.md#update-existing-file-in-repository
func (s *RepositoryFilesService) UpdateFile(pid interface{}, opt *UpdateFileOptions, options ...OptionFunc) (*FileInfo, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/files", url.QueryEscape(project))

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	f := new(FileInfo)
	resp, err := s.client.Do(req, f)
	if err != nil {
		return nil, resp, err
	}

	return f, resp, err
}

// DeleteFileOptions represents the available DeleteFile() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/repository_files.md#delete-existing-file-in-repository
type DeleteFileOptions struct {
	FilePath      *string `url:"file_path,omitempty" json:"file_path,omitempty"`
	BranchName    *string `url:"branch_name,omitempty" json:"branch_name,omitempty"`
	AuthorEmail   *string `url:"author_email,omitempty" json:"author_email,omitempty"`
	AuthorName    *string `url:"author_name,omitempty" json:"author_name,omitempty"`
	CommitMessage *string `url:"commit_message,omitempty" json:"commit_message,omitempty"`
}

// DeleteFile deletes an existing file in a repository
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/repository_files.md#delete-existing-file-in-repository
func (s *RepositoryFilesService) DeleteFile(pid interface{}, opt *DeleteFileOptions, options ...OptionFunc) (*FileInfo, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/repository/files", url.QueryEscape(project))

	req, err := s.client.NewRequest("DELETE", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	f := new(FileInfo)
	resp, err := s.client.Do(req, f)
	if err != nil {
		return nil, resp, err
	}

	return f, resp, err
}
