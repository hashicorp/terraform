package tfe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Compile-time proof of interface implementation.
var _ PlanExports = (*planExports)(nil)

// PlanExports describes all the plan export related methods that the Terraform
// Enterprise API supports.
//
// TFE API docs: https://www.terraform.io/docs/enterprise/api/plan-exports.html
type PlanExports interface {
	// Export a plan by its ID with the given options.
	Create(ctx context.Context, options PlanExportCreateOptions) (*PlanExport, error)

	// Read a plan export by its ID.
	Read(ctx context.Context, planExportID string) (*PlanExport, error)

	// Delete a plan export by its ID.
	Delete(ctx context.Context, planExportID string) error

	// Download the data of an plan export.
	Download(ctx context.Context, planExportID string) ([]byte, error)
}

// planExports implements PlanExports.
type planExports struct {
	client *Client
}

// PlanExportDataType represents the type of data exported from a plan.
type PlanExportDataType string

// List all available plan export data types.
const (
	PlanExportSentinelMockBundleV0 PlanExportDataType = "sentinel-mock-bundle-v0"
)

// PlanExportStatus represents a plan export state.
type PlanExportStatus string

// List all available plan export statuses.
const (
	PlanExportCanceled PlanExportStatus = "canceled"
	PlanExportErrored  PlanExportStatus = "errored"
	PlanExportExpired  PlanExportStatus = "expired"
	PlanExportFinished PlanExportStatus = "finished"
	PlanExportPending  PlanExportStatus = "pending"
	PlanExportQueued   PlanExportStatus = "queued"
)

// PlanExportStatusTimestamps holds the timestamps for plan export statuses.
type PlanExportStatusTimestamps struct {
	CanceledAt time.Time `json:"canceled-at"`
	ErroredAt  time.Time `json:"errored-at"`
	ExpiredAt  time.Time `json:"expired-at"`
	FinishedAt time.Time `json:"finished-at"`
	QueuedAt   time.Time `json:"queued-at"`
}

// PlanExport represents an export of Terraform Enterprise plan data.
type PlanExport struct {
	ID               string                      `jsonapi:"primary,plan-exports"`
	DataType         PlanExportDataType          `jsonapi:"attr,data-type"`
	Status           PlanExportStatus            `jsonapi:"attr,status"`
	StatusTimestamps *PlanExportStatusTimestamps `jsonapi:"attr,status-timestamps"`
}

// PlanExportCreateOptions represents the options for exporting data from a plan.
type PlanExportCreateOptions struct {
	// For internal use only!
	ID string `jsonapi:"primary,plan-exports"`

	// The plan to export.
	Plan *Plan `jsonapi:"relation,plan"`

	// The name of the policy set.
	DataType *PlanExportDataType `jsonapi:"attr,data-type"`
}

func (o PlanExportCreateOptions) valid() error {
	if o.Plan == nil {
		return errors.New("plan is required")
	}
	if o.DataType == nil {
		return errors.New("data type is required")
	}
	return nil
}

func (s *planExports) Create(ctx context.Context, options PlanExportCreateOptions) (*PlanExport, error) {
	if err := options.valid(); err != nil {
		return nil, err
	}

	// Make sure we don't send a user provided ID.
	options.ID = ""

	req, err := s.client.newRequest("POST", "plan-exports", &options)
	if err != nil {
		return nil, err
	}

	pe := &PlanExport{}
	err = s.client.do(ctx, req, pe)
	if err != nil {
		return nil, err
	}

	return pe, err
}

// Read a plan export by its ID.
func (s *planExports) Read(ctx context.Context, planExportID string) (*PlanExport, error) {
	if !validStringID(&planExportID) {
		return nil, errors.New("invalid value for plan export ID")
	}

	u := fmt.Sprintf("plan-exports/%s", url.QueryEscape(planExportID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	pe := &PlanExport{}
	err = s.client.do(ctx, req, pe)
	if err != nil {
		return nil, err
	}

	return pe, nil
}

// Delete a plan export by ID.
func (s *planExports) Delete(ctx context.Context, planExportID string) error {
	if !validStringID(&planExportID) {
		return errors.New("invalid value for plan export ID")
	}

	u := fmt.Sprintf("plan-exports/%s", url.QueryEscape(planExportID))
	req, err := s.client.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}

	return s.client.do(ctx, req, nil)
}

// Download a plan export's data. Data is exported in a .tar.gz format.
func (s *planExports) Download(ctx context.Context, planExportID string) ([]byte, error) {
	if !validStringID(&planExportID) {
		return nil, errors.New("invalid value for plan export ID")
	}

	u := fmt.Sprintf("plan-exports/%s/download", url.QueryEscape(planExportID))
	req, err := s.client.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = s.client.do(ctx, req, &buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
