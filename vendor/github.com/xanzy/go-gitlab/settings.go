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

// SettingsService handles communication with the application SettingsService
// related methods of the GitLab API.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/settings.md
type SettingsService struct {
	client *Client
}

// Settings represents the GitLab application settings.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/settings.md
type Settings struct {
	ID                         int                    `json:"id"`
	DefaultProjectsLimit       int                    `json:"default_projects_limit"`
	SignupEnabled              bool                   `json:"signup_enabled"`
	SigninEnabled              bool                   `json:"signin_enabled"`
	GravatarEnabled            bool                   `json:"gravatar_enabled"`
	SignInText                 string                 `json:"sign_in_text"`
	CreatedAt                  *time.Time             `json:"created_at"`
	UpdatedAt                  *time.Time             `json:"updated_at"`
	HomePageURL                string                 `json:"home_page_url"`
	DefaultBranchProtection    int                    `json:"default_branch_protection"`
	TwitterSharingEnabled      bool                   `json:"twitter_sharing_enabled"`
	RestrictedVisibilityLevels []VisibilityLevelValue `json:"restricted_visibility_levels"`
	MaxAttachmentSize          int                    `json:"max_attachment_size"`
	SessionExpireDelay         int                    `json:"session_expire_delay"`
	DefaultProjectVisibility   int                    `json:"default_project_visibility"`
	DefaultSnippetVisibility   int                    `json:"default_snippet_visibility"`
	RestrictedSignupDomains    []string               `json:"restricted_signup_domains"`
	UserOauthApplications      bool                   `json:"user_oauth_applications"`
	AfterSignOutPath           string                 `json:"after_sign_out_path"`
}

func (s Settings) String() string {
	return Stringify(s)
}

// GetSettings gets the current application settings.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/settings.md#get-current-application.settings
func (s *SettingsService) GetSettings(options ...OptionFunc) (*Settings, *Response, error) {
	req, err := s.client.NewRequest("GET", "application/settings", nil, options)
	if err != nil {
		return nil, nil, err
	}

	as := new(Settings)
	resp, err := s.client.Do(req, as)
	if err != nil {
		return nil, resp, err
	}

	return as, resp, err
}

// UpdateSettingsOptions represents the available UpdateSettings() options.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/settings.md#change-application.settings
type UpdateSettingsOptions struct {
	DefaultProjectsLimit       *int                   `url:"default_projects_limit,omitempty" json:"default_projects_limit,omitempty"`
	SignupEnabled              *bool                  `url:"signup_enabled,omitempty" json:"signup_enabled,omitempty"`
	SigninEnabled              *bool                  `url:"signin_enabled,omitempty" json:"signin_enabled,omitempty"`
	GravatarEnabled            *bool                  `url:"gravatar_enabled,omitempty" json:"gravatar_enabled,omitempty"`
	SignInText                 *string                `url:"sign_in_text,omitempty" json:"sign_in_text,omitempty"`
	HomePageURL                *string                `url:"home_page_url,omitempty" json:"home_page_url,omitempty"`
	DefaultBranchProtection    *int                   `url:"default_branch_protection,omitempty" json:"default_branch_protection,omitempty"`
	TwitterSharingEnabled      *bool                  `url:"twitter_sharing_enabled,omitempty" json:"twitter_sharing_enabled,omitempty"`
	RestrictedVisibilityLevels []VisibilityLevelValue `url:"restricted_visibility_levels,omitempty" json:"restricted_visibility_levels,omitempty"`
	MaxAttachmentSize          *int                   `url:"max_attachment_size,omitempty" json:"max_attachment_size,omitempty"`
	SessionExpireDelay         *int                   `url:"session_expire_delay,omitempty" json:"session_expire_delay,omitempty"`
	DefaultProjectVisibility   *int                   `url:"default_project_visibility,omitempty" json:"default_project_visibility,omitempty"`
	DefaultSnippetVisibility   *int                   `url:"default_snippet_visibility,omitempty" json:"default_snippet_visibility,omitempty"`
	RestrictedSignupDomains    []string               `url:"restricted_signup_domains,omitempty" json:"restricted_signup_domains,omitempty"`
	UserOauthApplications      *bool                  `url:"user_oauth_applications,omitempty" json:"user_oauth_applications,omitempty"`
	AfterSignOutPath           *string                `url:"after_sign_out_path,omitempty" json:"after_sign_out_path,omitempty"`
}

// UpdateSettings updates the application settings.
//
// GitLab API docs:
// https://gitlab.com/gitlab-org/gitlab-ce/blob/8-16-stable/doc/api/settings.md#change-application.settings
func (s *SettingsService) UpdateSettings(opt *UpdateSettingsOptions, options ...OptionFunc) (*Settings, *Response, error) {
	req, err := s.client.NewRequest("PUT", "application/settings", opt, options)
	if err != nil {
		return nil, nil, err
	}

	as := new(Settings)
	resp, err := s.client.Do(req, as)
	if err != nil {
		return nil, resp, err
	}

	return as, resp, err
}
