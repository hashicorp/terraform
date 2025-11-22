// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
)

func TestLocal_applyStopEarly(t *testing.T) {
	b := TestLocal(t)

	schema := applyFixtureSchema()
	schema.DataSources = map[string]providers.Schema{
		"test_data": {
			Body: &configschema.Block{},
		},
	}

	// Create a provider that blocks on ReadDataSource
	p := TestLocalProvider(t, b, "test", schema)

	// We need to make sure ReadDataSource is called and blocks
	readCalled := make(chan struct{})
	block := make(chan struct{})

	// Override the ReadDataSourceFn
	p.ReadDataSourceFn = func(req providers.ReadDataSourceRequest) providers.ReadDataSourceResponse {
		close(readCalled)
		<-block
		return providers.ReadDataSourceResponse{
			State: req.Config,
		}
	}

	op, configCleanup, done := testOperationApply(t, "./testdata/apply-stop")
	defer configCleanup()

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately to simulate "early" interrupt
	cancel()

	run, err := b.Operation(ctx, op)
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Wait for result
	doneCh := make(chan struct{})
	go func() {
		<-run.Done()
		close(doneCh)
	}()

	select {
	case <-doneCh:
		// Success (it returned)
	case <-readCalled:
		// It reached the provider! This means it didn't stop early.
		close(block) // Unblock to cleanup
		t.Fatal("Operation reached provider despite early cancellation")
	case <-time.After(5 * time.Second):
		t.Fatal("Operation timed out")
	}

	if run.Result == backendrun.OperationSuccess {
		t.Fatal("Operation succeeded but should have been cancelled")
	}

	if errOutput := done(t).Stderr(); errOutput != "" {
		// We expect some error output due to cancellation, but let's log it just in case
		t.Logf("error output:\n%s", errOutput)
	}
}
