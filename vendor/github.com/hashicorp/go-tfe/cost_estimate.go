package tfe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ CostEstimates = (*costEstimates)(nil)

// CostEstimates describes all the costEstimate related methods that
// the Terraform Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/ (TBD)
type CostEstimates interface {
	// Read a costEstimate by its ID.
	Read(ctx context.Context, costEstimateID string) (*CostEstimate, error)

	// Logs retrieves the logs of a costEstimate.
	Logs(ctx context.Context, costEstimateID string) (io.Reader, error)
}

// costEstimates implements CostEstimates.
type costEstimates struct {
	client *Client
}

// CostEstimateStatus represents a costEstimate state.
type CostEstimateStatus string

//List all available costEstimate statuses.
const (
	CostEstimateCanceled CostEstimateStatus = "canceled"
	CostEstimateErrored  CostEstimateStatus = "errored"
	CostEstimateFinished CostEstimateStatus = "finished"
	CostEstimatePending  CostEstimateStatus = "pending"
	CostEstimateQueued   CostEstimateStatus = "queued"
)

// CostEstimate represents a Terraform Enterprise costEstimate.
type CostEstimate struct {
	ID                      string                        `jsonapi:"primary,cost-estimates"`
	DeltaMonthlyCost        string                        `jsonapi:"attr,delta-monthly-cost"`
	ErrorMessage            string                        `jsonapi:"attr,error-message"`
	MatchedResourcesCount   int                           `jsonapi:"attr,matched-resources-count"`
	PriorMonthlyCost        string                        `jsonapi:"attr,prior-monthly-cost"`
	ProposedMonthlyCost     string                        `jsonapi:"attr,proposed-monthly-cost"`
	ResourcesCount          int                           `jsonapi:"attr,resources-count"`
	Status                  CostEstimateStatus            `jsonapi:"attr,status"`
	StatusTimestamps        *CostEstimateStatusTimestamps `jsonapi:"attr,status-timestamps"`
	UnmatchedResourcesCount int                           `jsonapi:"attr,unmatched-resources-count"`
}

// CostEstimateStatusTimestamps holds the timestamps for individual costEstimate statuses.
type CostEstimateStatusTimestamps struct {
	CanceledAt time.Time `json:"canceled-at"`
	ErroredAt  time.Time `json:"errored-at"`
	FinishedAt time.Time `json:"finished-at"`
	PendingAt  time.Time `json:"pending-at"`
	QueuedAt   time.Time `json:"queued-at"`
}

// Read a costEstimate by its ID.
func (s *costEstimates) Read(ctx context.Context, costEstimateID string) (*CostEstimate, error) {
	if !validStringID(&costEstimateID) {
		return nil, errors.New("invalid value for cost estimate ID")
	}

	u := fmt.Sprintf("cost-estimates/%s", url.QueryEscape(costEstimateID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ce := &CostEstimate{}
	err = s.client.do(ctx, req, ce)
	if err != nil {
		return nil, err
	}

	return ce, nil
}

// Logs retrieves the logs of a costEstimate.
func (s *costEstimates) Logs(ctx context.Context, costEstimateID string) (io.Reader, error) {
	if !validStringID(&costEstimateID) {
		return nil, errors.New("invalid value for cost estimate ID")
	}

	// Loop until the context is canceled or the cost estimate is finished
	// running. The cost estimate logs are not streamed and so only available
	// once the estimate is finished.
	for {
		// Get the costEstimate to make sure it exists.
		ce, err := s.Read(ctx, costEstimateID)
		if err != nil {
			return nil, err
		}

		switch ce.Status {
		case CostEstimateQueued:
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(1000 * time.Millisecond):
				continue
			}
		}

		u := fmt.Sprintf("cost-estimates/%s/output", url.QueryEscape(costEstimateID))
		req, err := s.client.newRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}

		logs := bytes.NewBuffer(nil)
		err = s.client.do(ctx, req, logs)
		if err != nil {
			return nil, err
		}

		return logs, nil
	}
}
