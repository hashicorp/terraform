// Copyright 2016 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
	"time"
)

// AppsService provides access to the installation related functions
// in the GitHub API.
//
// GitHub API docs: https://developer.github.com/v3/apps/
type AppsService service

// App represents a GitHub App.
type App struct {
	ID          *int64     `json:"id,omitempty"`
	Owner       *User      `json:"owner,omitempty"`
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	ExternalURL *string    `json:"external_url,omitempty"`
	HTMLURL     *string    `json:"html_url,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// InstallationToken represents an installation token.
type InstallationToken struct {
	Token     *string    `json:"token,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// InstallationPermissions lists the permissions for metadata, contents, issues and single file for an installation.
type InstallationPermissions struct {
	Metadata   *string `json:"metadata,omitempty"`
	Contents   *string `json:"contents,omitempty"`
	Issues     *string `json:"issues,omitempty"`
	SingleFile *string `json:"single_file,omitempty"`
}

// Installation represents a GitHub Apps installation.
type Installation struct {
	ID                  *int64                   `json:"id,omitempty"`
	AppID               *int64                   `json:"app_id,omitempty"`
	TargetID            *int64                   `json:"target_id,omitempty"`
	Account             *User                    `json:"account,omitempty"`
	AccessTokensURL     *string                  `json:"access_tokens_url,omitempty"`
	RepositoriesURL     *string                  `json:"repositories_url,omitempty"`
	HTMLURL             *string                  `json:"html_url,omitempty"`
	TargetType          *string                  `json:"target_type,omitempty"`
	SingleFileName      *string                  `json:"single_file_name,omitempty"`
	RepositorySelection *string                  `json:"repository_selection,omitempty"`
	Events              []string                 `json:"events,omitempty"`
	Permissions         *InstallationPermissions `json:"permissions,omitempty"`
}

func (i Installation) String() string {
	return Stringify(i)
}

// Get a single GitHub App. Passing the empty string will get
// the authenticated GitHub App.
//
// Note: appSlug is just the URL-friendly name of your GitHub App.
// You can find this on the settings page for your GitHub App
// (e.g., https://github.com/settings/apps/:app_slug).
//
// GitHub API docs: https://developer.github.com/v3/apps/#get-a-single-github-app
func (s *AppsService) Get(ctx context.Context, appSlug string) (*App, *Response, error) {
	var u string
	if appSlug != "" {
		u = fmt.Sprintf("apps/%v", appSlug)
	} else {
		u = "app"
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept header when this API fully launches.
	req.Header.Set("Accept", mediaTypeIntegrationPreview)

	app := new(App)
	resp, err := s.client.Do(ctx, req, app)
	if err != nil {
		return nil, resp, err
	}

	return app, resp, nil
}

// ListInstallations lists the installations that the current GitHub App has.
//
// GitHub API docs: https://developer.github.com/v3/apps/#find-installations
func (s *AppsService) ListInstallations(ctx context.Context, opt *ListOptions) ([]*Installation, *Response, error) {
	u, err := addOptions("app/installations", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept header when this API fully launches.
	req.Header.Set("Accept", mediaTypeIntegrationPreview)

	var i []*Installation
	resp, err := s.client.Do(ctx, req, &i)
	if err != nil {
		return nil, resp, err
	}

	return i, resp, nil
}

// GetInstallation returns the specified installation.
//
// GitHub API docs: https://developer.github.com/v3/apps/#get-a-single-installation
func (s *AppsService) GetInstallation(ctx context.Context, id int64) (*Installation, *Response, error) {
	u := fmt.Sprintf("app/installations/%v", id)

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept header when this API fully launches.
	req.Header.Set("Accept", mediaTypeIntegrationPreview)

	i := new(Installation)
	resp, err := s.client.Do(ctx, req, i)
	if err != nil {
		return nil, resp, err
	}

	return i, resp, nil
}

// ListUserInstallations lists installations that are accessible to the authenticated user.
//
// GitHub API docs: https://developer.github.com/v3/apps/#list-installations-for-user
func (s *AppsService) ListUserInstallations(ctx context.Context, opt *ListOptions) ([]*Installation, *Response, error) {
	u, err := addOptions("user/installations", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept header when this API fully launches.
	req.Header.Set("Accept", mediaTypeIntegrationPreview)

	var i struct {
		Installations []*Installation `json:"installations"`
	}
	resp, err := s.client.Do(ctx, req, &i)
	if err != nil {
		return nil, resp, err
	}

	return i.Installations, resp, nil
}

// CreateInstallationToken creates a new installation token.
//
// GitHub API docs: https://developer.github.com/v3/apps/#create-a-new-installation-token
func (s *AppsService) CreateInstallationToken(ctx context.Context, id int64) (*InstallationToken, *Response, error) {
	u := fmt.Sprintf("installations/%v/access_tokens", id)

	req, err := s.client.NewRequest("POST", u, nil)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept header when this API fully launches.
	req.Header.Set("Accept", mediaTypeIntegrationPreview)

	t := new(InstallationToken)
	resp, err := s.client.Do(ctx, req, t)
	if err != nil {
		return nil, resp, err
	}

	return t, resp, nil
}
