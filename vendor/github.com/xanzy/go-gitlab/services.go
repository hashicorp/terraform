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

// ServicesService handles communication with the services related methods of
// the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md
type ServicesService struct {
	client *Client
}

// Service represents a GitLab service.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md
type Service struct {
	ID                  *int       `json:"id"`
	Title               *string    `json:"title"`
	CreatedAt           *time.Time `json:"created_at"`
	UpdatedAt           *time.Time `json:"created_at"`
	Active              *bool      `json:"active"`
	PushEvents          *bool      `json:"push_events"`
	IssuesEvents        *bool      `json:"issues_events"`
	MergeRequestsEvents *bool      `json:"merge_requests_events"`
	TagPushEvents       *bool      `json:"tag_push_events"`
	NoteEvents          *bool      `json:"note_events"`
}

// SetGitLabCIServiceOptions represents the available SetGitLabCIService()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#edit-gitlab-ci-service
type SetGitLabCIServiceOptions struct {
	Token      *string `url:"token,omitempty" json:"token,omitempty"`
	ProjectURL *string `url:"project_url,omitempty" json:"project_url,omitempty"`
}

// SetGitLabCIService sets GitLab CI service for a project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#edit-gitlab-ci-service
func (s *ServicesService) SetGitLabCIService(pid interface{}, opt *SetGitLabCIServiceOptions, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/services/gitlab-ci", url.QueryEscape(project))

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// DeleteGitLabCIService deletes GitLab CI service settings for a project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#delete-gitlab-ci-service
func (s *ServicesService) DeleteGitLabCIService(pid interface{}, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/services/gitlab-ci", url.QueryEscape(project))

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// SetHipChatServiceOptions represents the available SetHipChatService()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#edit-hipchat-service
type SetHipChatServiceOptions struct {
	Token *string `url:"token,omitempty" json:"token,omitempty" `
	Room  *string `url:"room,omitempty" json:"room,omitempty"`
}

// SetHipChatService sets HipChat service for a project
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#edit-hipchat-service
func (s *ServicesService) SetHipChatService(pid interface{}, opt *SetHipChatServiceOptions, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/services/hipchat", url.QueryEscape(project))

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// DeleteHipChatService deletes HipChat service for project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#delete-hipchat-service
func (s *ServicesService) DeleteHipChatService(pid interface{}, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/services/hipchat", url.QueryEscape(project))

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// SetDroneCIServiceOptions represents the available SetDroneCIService()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#createedit-drone-ci-service
type SetDroneCIServiceOptions struct {
	Token                 *string `url:"token" json:"token" `
	DroneURL              *string `url:"drone_url" json:"drone_url"`
	EnableSSLVerification *string `url:"enable_ssl_verification,omitempty" json:"enable_ssl_verification,omitempty"`
}

// SetDroneCIService sets Drone CI service for a project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#createedit-drone-ci-service
func (s *ServicesService) SetDroneCIService(pid interface{}, opt *SetDroneCIServiceOptions, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/services/drone-ci", url.QueryEscape(project))

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// DeleteDroneCIService deletes Drone CI service settings for a project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#delete-drone-ci-service
func (s *ServicesService) DeleteDroneCIService(pid interface{}, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/services/drone-ci", url.QueryEscape(project))

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// DroneCIServiceProperties represents Drone CI specific properties.
type DroneCIServiceProperties struct {
	Token                 *string `url:"token" json:"token"`
	DroneURL              *string `url:"drone_url" json:"drone_url"`
	EnableSSLVerification *string `url:"enable_ssl_verification" json:"enable_ssl_verification"`
}

// DroneCIService represents Drone CI service settings.
type DroneCIService struct {
	Service
	Properties *DroneCIServiceProperties `json:"properties"`
}

// GetDroneCIService gets Drone CI service settings for a project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#get-drone-ci-service-settings
func (s *ServicesService) GetDroneCIService(pid interface{}, options ...OptionFunc) (*DroneCIService, *Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, nil, err
	}
	u := fmt.Sprintf("projects/%s/services/drone-ci", url.QueryEscape(project))

	req, err := s.client.NewRequest("GET", u, nil, options)
	if err != nil {
		return nil, nil, err
	}

	opt := new(DroneCIService)
	resp, err := s.client.Do(req, opt)
	if err != nil {
		return nil, resp, err
	}

	return opt, resp, err
}

// SetSlackServiceOptions represents the available SetSlackService()
// options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#edit-slack-service
type SetSlackServiceOptions struct {
	WebHook  *string `url:"webhook,omitempty" json:"webhook,omitempty" `
	Username *string `url:"username,omitempty" json:"username,omitempty" `
	Channel  *string `url:"channel,omitempty" json:"channel,omitempty"`
}

// SetSlackService sets Slack service for a project
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#edit-slack-service
func (s *ServicesService) SetSlackService(pid interface{}, opt *SetSlackServiceOptions, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/services/slack", url.QueryEscape(project))

	req, err := s.client.NewRequest("PUT", u, opt, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}

// DeleteSlackService deletes Slack service for project.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/services.md#delete-slack-service
func (s *ServicesService) DeleteSlackService(pid interface{}, options ...OptionFunc) (*Response, error) {
	project, err := parseID(pid)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("projects/%s/services/slack", url.QueryEscape(project))

	req, err := s.client.NewRequest("DELETE", u, nil, options)
	if err != nil {
		return nil, err
	}

	return s.client.Do(req, nil)
}
