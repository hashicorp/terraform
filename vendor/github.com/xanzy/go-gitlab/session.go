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

import "time"

// SessionService handles communication with the session related methods of
// the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/session.md
type SessionService struct {
	client *Client
}

// Session represents a GitLab session.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/session.md#session
type Session struct {
	ID               int         `json:"id"`
	Username         string      `json:"username"`
	Email            string      `json:"email"`
	Name             string      `json:"name"`
	PrivateToken     string      `json:"private_token"`
	Blocked          bool        `json:"blocked"`
	CreatedAt        *time.Time  `json:"created_at"`
	Bio              interface{} `json:"bio"`
	Skype            string      `json:"skype"`
	Linkedin         string      `json:"linkedin"`
	Twitter          string      `json:"twitter"`
	WebsiteURL       string      `json:"website_url"`
	DarkScheme       bool        `json:"dark_scheme"`
	ThemeID          int         `json:"theme_id"`
	IsAdmin          bool        `json:"is_admin"`
	CanCreateGroup   bool        `json:"can_create_group"`
	CanCreateTeam    bool        `json:"can_create_team"`
	CanCreateProject bool        `json:"can_create_project"`
}

// GetSessionOptions represents the available Session() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/session.md#session
type GetSessionOptions struct {
	Login    *string `url:"login,omitempty" json:"login,omitempty"`
	Email    *string `url:"email,omitempty" json:"email,omitempty"`
	Password *string `url:"password,omitempty" json:"password,omitempty"`
}

// GetSession logs in to get private token.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/session.md#session
func (s *SessionService) GetSession(opt *GetSessionOptions, options ...OptionFunc) (*Session, *Response, error) {
	req, err := s.client.NewRequest("POST", "session", opt, options)
	if err != nil {
		return nil, nil, err
	}

	session := new(Session)
	resp, err := s.client.Do(req, session)
	if err != nil {
		return nil, resp, err
	}

	return session, resp, err
}
