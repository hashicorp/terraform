// Copyright IBM Corp. 2014, 2025
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
)

type OperationResult interface {
	GetID() string
	Read(ctx context.Context) (OperationResult, error)
	Cancel(ctx context.Context, op *backendrun.Operation) error
	IsCanceled() bool
	IsErrored() bool
	HasResult() bool
	HasChanges() bool
}
type RunResult struct {
	run     *tfe.Run
	backend *Cloud
}

func (r *RunResult) GetID() string    { return r.run.ID }
func (r *RunResult) HasChanges() bool { return r.run.HasChanges }
func (r *RunResult) IsCanceled() bool { return r.run.Status == tfe.RunCanceled }
func (r *RunResult) IsErrored() bool  { return r.run.Status == tfe.RunErrored }
func (r *RunResult) HasResult() bool  { return r.run != nil }

func (r *RunResult) Read(ctx context.Context) (OperationResult, error) {
	run, err := r.backend.client.Runs.Read(ctx, r.run.ID)
	if err != nil {
		return nil, err
	}
	return &RunResult{run: run, backend: r.backend}, nil
}

func (r *RunResult) Cancel(ctx context.Context, op *backendrun.Operation) error {
	return r.backend.cancel(ctx, op, r.run)
}

type QueryRunResult struct {
	run     *tfe.QueryRun
	backend *Cloud
}

func (qr *QueryRunResult) GetID() string    { return qr.run.ID }
func (qr *QueryRunResult) HasChanges() bool { return false }
func (qr *QueryRunResult) IsCanceled() bool { return qr.run.Status == tfe.QueryRunCanceled }
func (qr *QueryRunResult) IsErrored() bool  { return qr.run.Status == tfe.QueryRunErrored }
func (qr *QueryRunResult) HasResult() bool  { return qr.run != nil }

func (qr *QueryRunResult) Read(ctx context.Context) (OperationResult, error) {
	run, err := qr.backend.client.QueryRuns.Read(ctx, qr.run.ID)
	if err != nil {
		return nil, err
	}
	return &QueryRunResult{run: run, backend: qr.backend}, nil
}

func (qr *QueryRunResult) Cancel(ctx context.Context, op *backendrun.Operation) error {
	return qr.backend.cancelQueryRun(ctx, op, qr.run)
}
