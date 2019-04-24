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
var _ CostEstimations = (*costEstimations)(nil)

// CostEstimations describes all the costEstimation related methods that
// the Terraform Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/ (TBD)
type CostEstimations interface {
	// Read a costEstimation by its ID.
	Read(ctx context.Context, costEstimationID string) (*CostEstimation, error)

	// Logs retrieves the logs of a costEstimation.
	Logs(ctx context.Context, costEstimationID string) (io.Reader, error)
}

// costEstimations implements CostEstimations.
type costEstimations struct {
	client *Client
}

// CostEstimationStatus represents a costEstimation state.
type CostEstimationStatus string

//List all available costEstimation statuses.
const (
	CostEstimationCanceled CostEstimationStatus = "canceled"
	CostEstimationErrored  CostEstimationStatus = "errored"
	CostEstimationFinished CostEstimationStatus = "finished"
	CostEstimationQueued   CostEstimationStatus = "queued"
)

// CostEstimation represents a Terraform Enterprise costEstimation.
type CostEstimation struct {
	ID               string                          `jsonapi:"primary,cost-estimations"`
	ErrorMessage     string                          `jsonapi:"attr,error-message"`
	Status           CostEstimationStatus            `jsonapi:"attr,status"`
	StatusTimestamps *CostEstimationStatusTimestamps `jsonapi:"attr,status-timestamps"`
}

// CostEstimationStatusTimestamps holds the timestamps for individual costEstimation statuses.
type CostEstimationStatusTimestamps struct {
	CanceledAt time.Time `json:"canceled-at"`
	ErroredAt  time.Time `json:"errored-at"`
	FinishedAt time.Time `json:"finished-at"`
	QueuedAt   time.Time `json:"queued-at"`
}

// Read a costEstimation by its ID.
func (s *costEstimations) Read(ctx context.Context, costEstimationID string) (*CostEstimation, error) {
	if !validStringID(&costEstimationID) {
		return nil, errors.New("invalid value for cost estimation ID")
	}

	u := fmt.Sprintf("cost-estimations/%s", url.QueryEscape(costEstimationID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	ce := &CostEstimation{}
	err = s.client.do(ctx, req, ce)
	if err != nil {
		return nil, err
	}

	return ce, nil
}

// Logs retrieves the logs of a costEstimation.
func (s *costEstimations) Logs(ctx context.Context, costEstimationID string) (io.Reader, error) {
	if !validStringID(&costEstimationID) {
		return nil, errors.New("invalid value for cost estimation ID")
	}

	// Loop until the context is canceled or the cost estimation is finished
	// running. The cost estimation logs are not streamed and so only available
	// once the estimation is finished.
	for {
		// Get the costEstimation to make sure it exists.
		ce, err := s.Read(ctx, costEstimationID)
		if err != nil {
			return nil, err
		}

		switch ce.Status {
		case CostEstimationQueued:
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}

		u := fmt.Sprintf("cost-estimations/%s/output", url.QueryEscape(costEstimationID))
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
