//
// Copyright 2017, Igor Varavko
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

// PipelinesService handles communication with the repositories related
// methods of the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/pipelines.md
type PipelinesService struct {
	client *Client
}

// Pipeline represents a GitLab pipeline.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/pipelines.md
type Pipeline struct {
	ID         int    `json:"id"`
	Status     string `json:"status"`
	Ref        string `json:"ref"`
	Sha        string `json:"sha"`
	BeforeSha  string `json:"before_sha"`
	Tag        bool   `json:"tag"`
	YamlErrors string `json:"yaml_errors"`
	User       struct {
		Name      string `json:"name"`
		Username  string `json:"username"`
		ID        int    `json:"id"`
		State     string `json:"state"`
		AvatarURL string `json:"avatar_url"`
		WebURL    string `json:"web_url"`
	}
	UpdatedAt   *time.Time `json:"updated_at"`
	CreatedAt   *time.Time `json:"created_at"`
	StartedAt   *time.Time `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at"`
	CommittedAt *time.Time `json:"committed_at"`
	Duration    int        `json:"duration"`
	Coverage    string     `json:"coverage"`
}

func (i Pipeline) String() string {
	return Stringify(i)
}

// ListProjectPipelines gets a list of project piplines.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/pipelines.md#list-project-pipelines
func (s *PipelinesService) ListProjectPipelines(pid interface{}, options ...OptionFunc) ([]*Pipeline, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/pipelines", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	var p []*Pipeline
	resp, err := s.client.Do(req, &p)
	if err != nil {
		return nil, resp, err
	}
	return p, resp, err
}

// GetPipeline gets a single project pipeline.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/pipelines.md#get-a-single-pipeline
func (s *PipelinesService) GetPipeline(pid interface{}, pipeline int, options ...OptionFunc) (*Pipeline, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/pipelines/%d", url.QueryEscape(project), pipeline)

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	p := new(Pipeline)
	resp, err := s.client.Do(req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}

// CreatePipelineOptions represents the available CreatePipeline() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/pipelines.md#create-a-new-pipeline
type CreatePipelineOptions struct {
	Ref *string `url:"ref,omitempty" json:"ref"`
}

// CreatePipeline creates a new project pipeline.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/pipelines.md#create-a-new-pipeline
func (s *PipelinesService) CreatePipeline(pid interface{}, opt *CreatePipelineOptions, options ...OptionFunc) (*Pipeline, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/pipeline", url.QueryEscape(project))

	req, err := s.client.NewRequest("POST", u, opt, options)
	if err != nil {
		return nil, nil, err
	}

	p := new(Pipeline)
	resp, err := s.client.Do(req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}

// RetryPipelineBuild retries failed builds in a pipeline
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/pipelines.md#retry-failed-builds-in-a-pipeline
func (s *PipelinesService) RetryPipelineBuild(pid interface{}, pipelineID int, options ...OptionFunc) (*Pipeline, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/pipelines/%d/retry", project, pipelineID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	p := new(Pipeline)
	resp, err := s.client.Do(req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}

// CancelPipelineBuild cancels a pipeline builds
//
// GitLab API docs:
//https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/pipelines.md#cancel-a-pipelines-builds
func (s *PipelinesService) CancelPipelineBuild(pid interface{}, pipelineID int, options ...OptionFunc) (*Pipeline, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/pipelines/%d/cancel", project, pipelineID)

	req, err := s.client.NewRequest("POST", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	p := new(Pipeline)
	resp, err := s.client.Do(req, p)
	if err != nil {
		return nil, resp, err
	}

	return p, resp, err
}
