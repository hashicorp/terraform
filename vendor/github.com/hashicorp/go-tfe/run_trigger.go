package tfe

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ RunTriggers = (*runTriggers)(nil)

// RunTriggers describes all the Run Trigger
// related methods that the Terraform Cloud API supports.
//
// TFE API docs:
// https://www.terraform.io/docs/cloud/api/run-triggers.html
type RunTriggers interface {
	// List all the run triggers within a workspace.
	List(ctx context.Context, workspaceID string, options RunTriggerListOptions) (*RunTriggerList, error)

	// Create a new run trigger with the given options.
	Create(ctx context.Context, workspaceID string, options RunTriggerCreateOptions) (*RunTrigger, error)

	// Read a run trigger by its ID.
	Read(ctx context.Context, RunTriggerID string) (*RunTrigger, error)

	// Delete a run trigger by its ID.
	Delete(ctx context.Context, RunTriggerID string) error
}

// runTriggers implements RunTriggers.
type runTriggers struct {
	client *Client
}

// RunTriggerList represents a list of Run Triggers
type RunTriggerList struct {
	*Pagination
	Items []*RunTrigger
}

// RunTrigger represents a run trigger.
type RunTrigger struct {
	ID             string    `jsonapi:"primary,run-triggers"`
	CreatedAt      time.Time `jsonapi:"attr,created-at,iso8601"`
	SourceableName string    `jsonapi:"attr,sourceable-name"`
	WorkspaceName  string    `jsonapi:"attr,workspace-name"`

	// Relations
	// TODO: this will eventually need to be polymorphic
	Sourceable *Workspace `jsonapi:"relation,sourceable"`
	Workspace  *Workspace `jsonapi:"relation,workspace"`
}

// RunTriggerListOptions represents the options for listing
// run triggers.
type RunTriggerListOptions struct {
	ListOptions
	RunTriggerType *string `url:"filter[run-trigger][type]"`
}

func (o RunTriggerListOptions) valid() error {
	if !validString(o.RunTriggerType) {
		return errors.New("run-trigger type is required")
	}
	if *o.RunTriggerType != "inbound" && *o.RunTriggerType != "outbound" {
		return errors.New("invalid value for run-trigger type")
	}
	return nil
}

// List all the run triggers associated with a workspace.
func (s *runTriggers) List(ctx context.Context, workspaceID string, options RunTriggerListOptions) (*RunTriggerList, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("invalid value for workspace ID")
	}

	if err := options.valid(); err != nil {
		return nil, err
	}

	u := fmt.Sprintf("workspaces/%s/run-triggers", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("GET", u, options)
	if err != nil {
		return nil, err
	}

	rtl := &RunTriggerList{}
	err = s.client.do(ctx, req, rtl)
	if err != nil {
		return nil, err
	}

	return rtl, nil
}

// RunTriggerCreateOptions represents the options for
// creating a new run trigger.
type RunTriggerCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,run-triggers"`

	// The source workspace
	Sourceable *Workspace `jsonapi:"relation,sourceable"`
}

func (o RunTriggerCreateOptions) valid() error {
	if o.Sourceable == nil {
		return errors.New("sourceable is required")
	}
	return nil
}

// Creates a run trigger with the given options.
func (s *runTriggers) Create(ctx context.Context, workspaceID string, options RunTriggerCreateOptions) (*RunTrigger, error) {
	if !validStringID(&workspaceID) {
		return nil, errors.New("invalid value for workspace ID")
	}
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	u := fmt.Sprintf("workspaces/%s/run-triggers", url.QueryEscape(workspaceID))
	req, err := s.client.newRequest("POST", u, &options)
	if err != nil {
		return nil, err
	}

	rt := &RunTrigger{}
	err = s.client.do(ctx, req, rt)
	if err != nil {
		return nil, err
	}

	return rt, nil
}

// Read a run trigger by its ID.
func (s *runTriggers) Read(ctx context.Context, runTriggerID string) (*RunTrigger, error) {
	if !validStringID(&runTriggerID) {
		return nil, errors.New("invalid value for run trigger ID")
	}

	u := fmt.Sprintf("run-triggers/%s", url.QueryEscape(runTriggerID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	rt := &RunTrigger{}
	err = s.client.do(ctx, req, rt)
	if err != nil {
		return nil, err
	}

	return rt, nil
}

// Delete a run trigger by its ID.
func (s *runTriggers) Delete(ctx context.Context, runTriggerID string) error {
	if !validStringID(&runTriggerID) {
		return errors.New("invalid value for run trigger ID")
	}

	u := fmt.Sprintf("run-triggers/%s", url.QueryEscape(runTriggerID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}
