package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ TeamTokens = (*teamTokens)(nil)

// TeamTokens describes all the team token related methods that the
// Terraform Enterprise API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/enterprise/api/team-tokens.html
type TeamTokens interface {
	// Generate a new team token, replacing any existing token.
	Generate(ctx context.Context, teamID string) (*TeamToken, error)

	// Read a team token by its ID.
	Read(ctx context.Context, teamID string) (*TeamToken, error)

	// Delete a team token by its ID.
	Delete(ctx context.Context, teamID string) error
}

// teamTokens implements TeamTokens.
type teamTokens struct {
	client *Client
}

// TeamToken represents a Terraform Enterprise team token.
type TeamToken struct {
	ID          string    `jsonapi:"primary,authentication-tokens"`
	CreatedAt   time.Time `jsonapi:"attr,created-at,iso8601"`
	Description string    `jsonapi:"attr,description"`
	LastUsedAt  time.Time `jsonapi:"attr,last-used-at,iso8601"`
	Token       string    `jsonapi:"attr,token"`
}

// Generate a new team token, replacing any existing token.
func (s *teamTokens) Generate(ctx context.Context, teamID string) (*TeamToken, error) {
	if !validStringID(&teamID) {
		return nil, errors.New("invalid value for team ID")
	}

	u := fmt.Sprintf("teams/%s/authentication-token", url.QueryEscape(teamID))
	req, err := s.client.newRequest("POST", u, nil)
	if err != nil {
		return nil, err
	}

	tt := &TeamToken{}
	err = s.client.do(ctx, req, tt)
	if err != nil {
		return nil, err
	}

	return tt, err
}

// Read a team token by its ID.
func (s *teamTokens) Read(ctx context.Context, teamID string) (*TeamToken, error) {
	if !validStringID(&teamID) {
		return nil, errors.New("invalid value for team ID")
	}

	u := fmt.Sprintf("teams/%s/authentication-token", url.QueryEscape(teamID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	tt := &TeamToken{}
	err = s.client.do(ctx, req, tt)
	if err != nil {
		return nil, err
	}

	return tt, err
}

// Delete a team token by its ID.
func (s *teamTokens) Delete(ctx context.Context, teamID string) error {
	if !validStringID(&teamID) {
		return errors.New("invalid value for team ID")
	}

	u := fmt.Sprintf("teams/%s/authentication-token", url.QueryEscape(teamID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
