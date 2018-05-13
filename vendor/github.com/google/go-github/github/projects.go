// Copyright 2016 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
	"strings"
)

// ProjectsService provides access to the projects functions in the
// GitHub API.
//
// GitHub API docs: https://developer.github.com/v3/projects/
type ProjectsService service

// Project represents a GitHub Project.
type Project struct {
	ID        *int64     `json:"id,omitempty"`
	URL       *string    `json:"url,omitempty"`
	OwnerURL  *string    `json:"owner_url,omitempty"`
	Name      *string    `json:"name,omitempty"`
	Body      *string    `json:"body,omitempty"`
	Number    *int       `json:"number,omitempty"`
	CreatedAt *Timestamp `json:"created_at,omitempty"`
	UpdatedAt *Timestamp `json:"updated_at,omitempty"`
	NodeID    *string    `json:"node_id,omitempty"`

	// The User object that generated the project.
	Creator *User `json:"creator,omitempty"`
}

func (p Project) String() string {
	return Stringify(p)
}

// GetProject gets a GitHub Project for a repo.
//
// GitHub API docs: https://developer.github.com/v3/projects/#get-a-project
func (s *ProjectsService) GetProject(ctx context.Context, id int64) (*Project, *Response, error) {
	u := fmt.Sprintf("projects/%v", id)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept headers when APIs fully launch.
	acceptHeaders := []string{mediaTypeProjectsPreview, mediaTypeGraphQLNodeIDPreview}
	req.Header.Set("Accept", strings.Join(acceptHeaders, ", "))

	project := &Project{}
	resp, err := s.client.Do(ctx, req, project)
	if err != nil {
		return nil, resp, err
	}

	return project, resp, nil
}

// ProjectOptions specifies the parameters to the
// RepositoriesService.CreateProject and
// ProjectsService.UpdateProject methods.
type ProjectOptions struct {
	// The name of the project. (Required for creation; optional for update.)
	Name string `json:"name,omitempty"`
	// The body of the project. (Optional.)
	Body string `json:"body,omitempty"`

	// The following field(s) are only applicable for update.
	// They should be left with zero values for creation.

	// State of the project. Either "open" or "closed". (Optional.)
	State string `json:"state,omitempty"`
}

// UpdateProject updates a repository project.
//
// GitHub API docs: https://developer.github.com/v3/projects/#update-a-project
func (s *ProjectsService) UpdateProject(ctx context.Context, id int64, opt *ProjectOptions) (*Project, *Response, error) {
	u := fmt.Sprintf("projects/%v", id)
	req, err := s.client.NewRequest("PATCH", u, opt)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept headers when APIs fully launch.
	acceptHeaders := []string{mediaTypeProjectsPreview, mediaTypeGraphQLNodeIDPreview}
	req.Header.Set("Accept", strings.Join(acceptHeaders, ", "))

	project := &Project{}
	resp, err := s.client.Do(ctx, req, project)
	if err != nil {
		return nil, resp, err
	}

	return project, resp, nil
}

// DeleteProject deletes a GitHub Project from a repository.
//
// GitHub API docs: https://developer.github.com/v3/projects/#delete-a-project
func (s *ProjectsService) DeleteProject(ctx context.Context, id int64) (*Response, error) {
	u := fmt.Sprintf("projects/%v", id)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	// TODO: remove custom Accept header when this API fully launches.
	req.Header.Set("Accept", mediaTypeProjectsPreview)

	return s.client.Do(ctx, req, nil)
}

// ProjectColumn represents a column of a GitHub Project.
//
// GitHub API docs: https://developer.github.com/v3/repos/projects/
type ProjectColumn struct {
	ID         *int64     `json:"id,omitempty"`
	Name       *string    `json:"name,omitempty"`
	ProjectURL *string    `json:"project_url,omitempty"`
	CreatedAt  *Timestamp `json:"created_at,omitempty"`
	UpdatedAt  *Timestamp `json:"updated_at,omitempty"`
	NodeID     *string    `json:"node_id,omitempty"`
}

// ListProjectColumns lists the columns of a GitHub Project for a repo.
//
// GitHub API docs: https://developer.github.com/v3/projects/columns/#list-project-columns
func (s *ProjectsService) ListProjectColumns(ctx context.Context, projectID int64, opt *ListOptions) ([]*ProjectColumn, *Response, error) {
	u := fmt.Sprintf("projects/%v/columns", projectID)
	u, err := addOptions(u, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept headers when APIs fully launch.
	acceptHeaders := []string{mediaTypeProjectsPreview, mediaTypeGraphQLNodeIDPreview}
	req.Header.Set("Accept", strings.Join(acceptHeaders, ", "))

	columns := []*ProjectColumn{}
	resp, err := s.client.Do(ctx, req, &columns)
	if err != nil {
		return nil, resp, err
	}

	return columns, resp, nil
}

// GetProjectColumn gets a column of a GitHub Project for a repo.
//
// GitHub API docs: https://developer.github.com/v3/projects/columns/#get-a-project-column
func (s *ProjectsService) GetProjectColumn(ctx context.Context, id int64) (*ProjectColumn, *Response, error) {
	u := fmt.Sprintf("projects/columns/%v", id)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept headers when APIs fully launch.
	acceptHeaders := []string{mediaTypeProjectsPreview, mediaTypeGraphQLNodeIDPreview}
	req.Header.Set("Accept", strings.Join(acceptHeaders, ", "))

	column := &ProjectColumn{}
	resp, err := s.client.Do(ctx, req, column)
	if err != nil {
		return nil, resp, err
	}

	return column, resp, nil
}

// ProjectColumnOptions specifies the parameters to the
// ProjectsService.CreateProjectColumn and
// ProjectsService.UpdateProjectColumn methods.
type ProjectColumnOptions struct {
	// The name of the project column. (Required for creation and update.)
	Name string `json:"name"`
}

// CreateProjectColumn creates a column for the specified (by number) project.
//
// GitHub API docs: https://developer.github.com/v3/projects/columns/#create-a-project-column
func (s *ProjectsService) CreateProjectColumn(ctx context.Context, projectID int64, opt *ProjectColumnOptions) (*ProjectColumn, *Response, error) {
	u := fmt.Sprintf("projects/%v/columns", projectID)
	req, err := s.client.NewRequest("POST", u, opt)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept headers when APIs fully launch.
	acceptHeaders := []string{mediaTypeProjectsPreview, mediaTypeGraphQLNodeIDPreview}
	req.Header.Set("Accept", strings.Join(acceptHeaders, ", "))

	column := &ProjectColumn{}
	resp, err := s.client.Do(ctx, req, column)
	if err != nil {
		return nil, resp, err
	}

	return column, resp, nil
}

// UpdateProjectColumn updates a column of a GitHub Project.
//
// GitHub API docs: https://developer.github.com/v3/projects/columns/#update-a-project-column
func (s *ProjectsService) UpdateProjectColumn(ctx context.Context, columnID int64, opt *ProjectColumnOptions) (*ProjectColumn, *Response, error) {
	u := fmt.Sprintf("projects/columns/%v", columnID)
	req, err := s.client.NewRequest("PATCH", u, opt)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept headers when APIs fully launch.
	acceptHeaders := []string{mediaTypeProjectsPreview, mediaTypeGraphQLNodeIDPreview}
	req.Header.Set("Accept", strings.Join(acceptHeaders, ", "))

	column := &ProjectColumn{}
	resp, err := s.client.Do(ctx, req, column)
	if err != nil {
		return nil, resp, err
	}

	return column, resp, nil
}

// DeleteProjectColumn deletes a column from a GitHub Project.
//
// GitHub API docs: https://developer.github.com/v3/projects/columns/#delete-a-project-column
func (s *ProjectsService) DeleteProjectColumn(ctx context.Context, columnID int64) (*Response, error) {
	u := fmt.Sprintf("projects/columns/%v", columnID)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	// TODO: remove custom Accept header when this API fully launches.
	req.Header.Set("Accept", mediaTypeProjectsPreview)

	return s.client.Do(ctx, req, nil)
}

// ProjectColumnMoveOptions specifies the parameters to the
// ProjectsService.MoveProjectColumn method.
type ProjectColumnMoveOptions struct {
	// Position can be one of "first", "last", or "after:<column-id>", where
	// <column-id> is the ID of a column in the same project. (Required.)
	Position string `json:"position"`
}

// MoveProjectColumn moves a column within a GitHub Project.
//
// GitHub API docs: https://developer.github.com/v3/projects/columns/#move-a-project-column
func (s *ProjectsService) MoveProjectColumn(ctx context.Context, columnID int64, opt *ProjectColumnMoveOptions) (*Response, error) {
	u := fmt.Sprintf("projects/columns/%v/moves", columnID)
	req, err := s.client.NewRequest("POST", u, opt)
	if err != nil {
		return nil, err
	}

	// TODO: remove custom Accept header when this API fully launches.
	req.Header.Set("Accept", mediaTypeProjectsPreview)

	return s.client.Do(ctx, req, nil)
}

// ProjectCard represents a card in a column of a GitHub Project.
//
// GitHub API docs: https://developer.github.com/v3/projects/cards/#get-a-project-card
type ProjectCard struct {
	URL        *string    `json:"url,omitempty"`
	ColumnURL  *string    `json:"column_url,omitempty"`
	ContentURL *string    `json:"content_url,omitempty"`
	ID         *int64     `json:"id,omitempty"`
	Note       *string    `json:"note,omitempty"`
	Creator    *User      `json:"creator,omitempty"`
	CreatedAt  *Timestamp `json:"created_at,omitempty"`
	UpdatedAt  *Timestamp `json:"updated_at,omitempty"`
	NodeID     *string    `json:"node_id,omitempty"`

	// The following fields are only populated by Webhook events.
	ColumnID *int64 `json:"column_id,omitempty"`
}

// ListProjectCards lists the cards in a column of a GitHub Project.
//
// GitHub API docs: https://developer.github.com/v3/projects/cards/#list-project-cards
func (s *ProjectsService) ListProjectCards(ctx context.Context, columnID int64, opt *ListOptions) ([]*ProjectCard, *Response, error) {
	u := fmt.Sprintf("projects/columns/%v/cards", columnID)
	u, err := addOptions(u, opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept headers when APIs fully launch.
	acceptHeaders := []string{mediaTypeProjectsPreview, mediaTypeGraphQLNodeIDPreview}
	req.Header.Set("Accept", strings.Join(acceptHeaders, ", "))

	cards := []*ProjectCard{}
	resp, err := s.client.Do(ctx, req, &cards)
	if err != nil {
		return nil, resp, err
	}

	return cards, resp, nil
}

// GetProjectCard gets a card in a column of a GitHub Project.
//
// GitHub API docs: https://developer.github.com/v3/projects/cards/#get-a-project-card
func (s *ProjectsService) GetProjectCard(ctx context.Context, columnID int64) (*ProjectCard, *Response, error) {
	u := fmt.Sprintf("projects/columns/cards/%v", columnID)
	req, err := s.client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept headers when APIs fully launch.
	acceptHeaders := []string{mediaTypeProjectsPreview, mediaTypeGraphQLNodeIDPreview}
	req.Header.Set("Accept", strings.Join(acceptHeaders, ", "))

	card := &ProjectCard{}
	resp, err := s.client.Do(ctx, req, card)
	if err != nil {
		return nil, resp, err
	}

	return card, resp, nil
}

// ProjectCardOptions specifies the parameters to the
// ProjectsService.CreateProjectCard and
// ProjectsService.UpdateProjectCard methods.
type ProjectCardOptions struct {
	// The note of the card. Note and ContentID are mutually exclusive.
	Note string `json:"note,omitempty"`
	// The ID (not Number) of the Issue to associate with this card.
	// Note and ContentID are mutually exclusive.
	ContentID int64 `json:"content_id,omitempty"`
	// The type of content to associate with this card. Possible values are: "Issue".
	ContentType string `json:"content_type,omitempty"`
}

// CreateProjectCard creates a card in the specified column of a GitHub Project.
//
// GitHub API docs: https://developer.github.com/v3/projects/cards/#create-a-project-card
func (s *ProjectsService) CreateProjectCard(ctx context.Context, columnID int64, opt *ProjectCardOptions) (*ProjectCard, *Response, error) {
	u := fmt.Sprintf("projects/columns/%v/cards", columnID)
	req, err := s.client.NewRequest("POST", u, opt)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept headers when APIs fully launch.
	acceptHeaders := []string{mediaTypeProjectsPreview, mediaTypeGraphQLNodeIDPreview}
	req.Header.Set("Accept", strings.Join(acceptHeaders, ", "))

	card := &ProjectCard{}
	resp, err := s.client.Do(ctx, req, card)
	if err != nil {
		return nil, resp, err
	}

	return card, resp, nil
}

// UpdateProjectCard updates a card of a GitHub Project.
//
// GitHub API docs: https://developer.github.com/v3/projects/cards/#update-a-project-card
func (s *ProjectsService) UpdateProjectCard(ctx context.Context, cardID int64, opt *ProjectCardOptions) (*ProjectCard, *Response, error) {
	u := fmt.Sprintf("projects/columns/cards/%v", cardID)
	req, err := s.client.NewRequest("PATCH", u, opt)
	if err != nil {
		return nil, nil, err
	}

	// TODO: remove custom Accept headers when APIs fully launch.
	acceptHeaders := []string{mediaTypeProjectsPreview, mediaTypeGraphQLNodeIDPreview}
	req.Header.Set("Accept", strings.Join(acceptHeaders, ", "))

	card := &ProjectCard{}
	resp, err := s.client.Do(ctx, req, card)
	if err != nil {
		return nil, resp, err
	}

	return card, resp, nil
}

// DeleteProjectCard deletes a card from a GitHub Project.
//
// GitHub API docs: https://developer.github.com/v3/projects/cards/#delete-a-project-card
func (s *ProjectsService) DeleteProjectCard(ctx context.Context, cardID int64) (*Response, error) {
	u := fmt.Sprintf("projects/columns/cards/%v", cardID)
	req, err := s.client.NewRequest("DELETE", u, nil)
	if err != nil {
		return nil, err
	}

	// TODO: remove custom Accept header when this API fully launches.
	req.Header.Set("Accept", mediaTypeProjectsPreview)

	return s.client.Do(ctx, req, nil)
}

// ProjectCardMoveOptions specifies the parameters to the
// ProjectsService.MoveProjectCard method.
type ProjectCardMoveOptions struct {
	// Position can be one of "top", "bottom", or "after:<card-id>", where
	// <card-id> is the ID of a card in the same project.
	Position string `json:"position"`
	// ColumnID is the ID of a column in the same project. Note that ColumnID
	// is required when using Position "after:<card-id>" when that card is in
	// another column; otherwise it is optional.
	ColumnID int64 `json:"column_id,omitempty"`
}

// MoveProjectCard moves a card within a GitHub Project.
//
// GitHub API docs: https://developer.github.com/v3/projects/cards/#move-a-project-card
func (s *ProjectsService) MoveProjectCard(ctx context.Context, cardID int64, opt *ProjectCardMoveOptions) (*Response, error) {
	u := fmt.Sprintf("projects/columns/cards/%v/moves", cardID)
	req, err := s.client.NewRequest("POST", u, opt)
	if err != nil {
		return nil, err
	}

	// TODO: remove custom Accept header when this API fully launches.
	req.Header.Set("Accept", mediaTypeProjectsPreview)

	return s.client.Do(ctx, req, nil)
}
