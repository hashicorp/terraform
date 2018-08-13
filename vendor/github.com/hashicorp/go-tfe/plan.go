package tfe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ Plans = (*plans)(nil)

// Plans describes all the plan related methods that the Terraform Enterprise
// API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/plan.html
type Plans interface {
	// Read a plan by its ID.
	Read(ctx context.Context, planID string) (*Plan, error)

	// Logs retrieves the logs of a plan.
	Logs(ctx context.Context, planID string) (io.Reader, error)
}

// plans implements Plans.
type plans struct {
	client *Client
}

// PlanStatus represents a plan state.
type PlanStatus string

//List all available plan statuses.
const (
	PlanCanceled   PlanStatus = "canceled"
	PlanCreated    PlanStatus = "created"
	PlanErrored    PlanStatus = "errored"
	PlanFinished   PlanStatus = "finished"
	PlanMFAWaiting PlanStatus = "mfa_waiting"
	PlanPending    PlanStatus = "pending"
	PlanQueued     PlanStatus = "queued"
	PlanRunning    PlanStatus = "running"
)

// Plan represents a Terraform Enterprise plan.
type Plan struct {
	ID               string                `jsonapi:"primary,plans"`
	HasChanges       bool                  `jsonapi:"attr,has-changes"`
	LogReadURL       string                `jsonapi:"attr,log-read-url"`
	Status           PlanStatus            `jsonapi:"attr,status"`
	StatusTimestamps *PlanStatusTimestamps `jsonapi:"attr,status-timestamps"`
}

// PlanStatusTimestamps holds the timestamps for individual plan statuses.
type PlanStatusTimestamps struct {
	CanceledAt   time.Time `json:"canceled-at"`
	CreatedAt    time.Time `json:"created-at"`
	ErroredAt    time.Time `json:"errored-at"`
	FinishedAt   time.Time `json:"finished-at"`
	MFAWaitingAt time.Time `json:"mfa_waiting-at"`
	PendingAt    time.Time `json:"pending-at"`
	QueuedAt     time.Time `json:"queued-at"`
	RunningAt    time.Time `json:"running-at"`
}

// Read a plan by its ID.
func (s *plans) Read(ctx context.Context, planID string) (*Plan, error) {
	if !validStringID(&planID) {
		return nil, errors.New("Invalid value for plan ID")
	}

	u := fmt.Sprintf("plans/%s", url.QueryEscape(planID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	p := &Plan{}
	err = s.client.do(ctx, req, p)
	if err != nil {
		return nil, err
	}

	return p, nil
}

// Logs retrieves the logs of a plan.
func (s *plans) Logs(ctx context.Context, planID string) (io.Reader, error) {
	if !validStringID(&planID) {
		return nil, errors.New("Invalid value for plan ID")
	}

	// Get the plan to make sure it exists.
	p, err := s.Read(ctx, planID)
	if err != nil {
		return nil, err
	}

	// Return an error if the log URL is empty.
	if p.LogReadURL == "" {
		return nil, fmt.Errorf("Plan %s does not have a log URL", planID)
	}

	u, err := url.Parse(p.LogReadURL)
	if err != nil {
		return nil, fmt.Errorf("Invalid log URL: %v", err)
	}

	return &LogReader{
		client: s.client,
		ctx:    ctx,
		logURL: u,
		plan:   p,
	}, nil
}

// LogReader implements io.Reader for streaming plan logs.
type LogReader struct {
	client *Client
	ctx    context.Context
	logURL *url.URL
	offset int64
	plan   *Plan
	reads  uint64
}

func (r *LogReader) Read(l []byte) (int, error) {
	if written, err := r.read(l); err != io.ErrNoProgress {
		return written, err
	}

	// Loop until we can any data, the context is canceled or the plan
	// is finsished running. If we would return right away without any
	// data, we could and up causing a io.ErrNoProgress error.
	for {
		select {
		case <-r.ctx.Done():
			return 0, r.ctx.Err()
		case <-time.After(500 * time.Millisecond):
			if written, err := r.read(l); err != io.ErrNoProgress {
				return written, err
			}
		}
	}
}

func (r *LogReader) read(l []byte) (int, error) {
	// Update the query string.
	r.logURL.RawQuery = fmt.Sprintf("limit=%d&offset=%d", len(l), r.offset)

	// Create a new request.
	req, err := http.NewRequest("GET", r.logURL.String(), nil)
	if err != nil {
		return 0, err
	}
	req = req.WithContext(r.ctx)

	// Retrieve the next chunk.
	resp, err := r.client.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// Basic response checking.
	if err := checkResponseCode(resp); err != nil {
		return 0, err
	}

	// Check if we need to continue the loop and wait 500 miliseconds
	// before checking if there is a new chunk available or that the
	// plan is finished and we are done reading all chunks.
	if resp.ContentLength == 0 {
		if r.reads%2 == 0 {
			r.plan, err = r.client.Plans.Read(r.ctx, r.plan.ID)
			if err != nil {
				return 0, err
			}
		}

		switch r.plan.Status {
		case PlanCanceled, PlanErrored, PlanFinished:
			return 0, io.EOF
		default:
			r.reads++
			return 0, io.ErrNoProgress
		}
	}

	// Read the retrieved chunk.
	written, err := resp.Body.Read(l)
	if err == io.EOF {
		// Ignore io.EOF errors returned when reading from the response
		// body as this indicates the end of the chunk and not the end
		// of the logfile.
		err = nil
	}

	// Update the offset for the next read.
	r.offset += int64(written)

	return written, err
}
