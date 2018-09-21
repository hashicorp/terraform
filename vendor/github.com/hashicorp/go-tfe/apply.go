package tfe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ Applies = (*applies)(nil)

// Applies describes all the apply related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/apply.html
type Applies interface {
	// Read an apply by its ID.
	Read(ctx context.Context, applyID string) (*Apply, error)

	// Logs retrieves the logs of an apply.
	Logs(ctx context.Context, applyID string) (io.Reader, error)
}

// applies implements Applys.
type applies struct {
	client *Client
}

// ApplyStatus represents an apply state.
type ApplyStatus string

//List all available apply statuses.
const (
	ApplyCanceled   ApplyStatus = "canceled"
	ApplyCreated    ApplyStatus = "created"
	ApplyErrored    ApplyStatus = "errored"
	ApplyFinished   ApplyStatus = "finished"
	ApplyMFAWaiting ApplyStatus = "mfa_waiting"
	ApplyPending    ApplyStatus = "pending"
	ApplyQueued     ApplyStatus = "queued"
	ApplyRunning    ApplyStatus = "running"
)

// Apply represents a Terraform Enterprise apply.
type Apply struct {
	ID                   string                 `jsonapi:"primary,applies"`
	LogReadURL           string                 `jsonapi:"attr,log-read-url"`
	ResourceAdditions    int                    `jsonapi:"attr,resource-additions"`
	ResourceChanges      int                    `jsonapi:"attr,resource-changes"`
	ResourceDestructions int                    `jsonapi:"attr,resource-destructions"`
	Status               ApplyStatus            `jsonapi:"attr,status"`
	StatusTimestamps     *ApplyStatusTimestamps `jsonapi:"attr,status-timestamps"`
}

// ApplyStatusTimestamps holds the timestamps for individual apply statuses.
type ApplyStatusTimestamps struct {
	CanceledAt      time.Time `json:"canceled-at"`
	ErroredAt       time.Time `json:"errored-at"`
	FinishedAt      time.Time `json:"finished-at"`
	ForceCanceledAt time.Time `json:"force-canceled-at"`
	QueuedAt        time.Time `json:"queued-at"`
	StartedAt       time.Time `json:"started-at"`
}

// Read an apply by its ID.
func (s *applies) Read(ctx context.Context, applyID string) (*Apply, error) {
	if !validStringID(&applyID) {
		return nil, errors.New("Invalid value for apply ID")
	}

	u := fmt.Sprintf("applies/%s", url.QueryEscape(applyID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	a := &Apply{}
	err = s.client.do(ctx, req, a)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Logs retrieves the logs of an apply.
func (s *applies) Logs(ctx context.Context, applyID string) (io.Reader, error) {
	if !validStringID(&applyID) {
		return nil, errors.New("Invalid value for apply ID")
	}

	// Get the apply to make sure it exists.
	a, err := s.Read(ctx, applyID)
	if err != nil {
		return nil, err
	}

	// Return an error if the log URL is empty.
	if a.LogReadURL == "" {
		return nil, fmt.Errorf("Apply %s does not have a log URL", applyID)
	}

	u, err := url.Parse(a.LogReadURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid log URL: %v", err)
	}

	done := func() (bool, error) {
		a, err := s.Read(ctx, a.ID)
		if err != nil {
			return false, err
		}

		switch a.Status {
		case ApplyCanceled, ApplyErrored, ApplyFinished:
			return true, nil
		default:
			return false, nil
		}
	}

	return &LogReader{
		client: s.client,
		ctx:    ctx,
		done:   done,
		logURL: u,
	}, nil
}
