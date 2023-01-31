// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	tfe "github.com/hashicorp/go-tfe"
)

// instanceAddr contains the state target that will be forcibly replaced every run
const instanceAddr = "module.tflocal.module.tfbox.aws_instance.tfbox"

// tokenAddr contains the target token that will be forcibly replaced every run
const tokenAddr = "module.tflocal.var.tflocal_cloud_admin_token"

var workspace string
var organization string
var isDestroy bool

func init() {
	flag.StringVar(&organization, "o", "hashicorp-v2", "the TFC organization that owns the specified workspace.")
	flag.StringVar(&workspace, "w", "tflocal-go-tfe", "the TFC workspace to create a run in.")
	flag.BoolVar(&isDestroy, "d", false, "trigger a destroy run.")
	flag.Parse()
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	config := &tfe.Config{
		RetryServerErrors: true,
	}

	client, err := tfe.NewClient(config)
	if err != nil {
		log.Fatalf("client initialization error: %v", err)
	}

	var runID string
	if runID, err = createRun(ctx, client); err != nil {
		log.Fatal(err)
	}

	// we should only wait if we are creating an instance
	if !isDestroy {
		if err = waitForRun(ctx, client, runID); err != nil {
			log.Fatal(err)
		}

		log.Printf("Run with ID successfully applied: %s", runID)
	}
}

func createRun(ctx context.Context, client *tfe.Client) (string, error) {
	wk, err := client.Workspaces.Read(ctx, organization, workspace)
	if err != nil {
		return "", fmt.Errorf("failed to read workspace: %w", err)
	}

	opts := tfe.RunCreateOptions{
		IsDestroy: tfe.Bool(isDestroy),
		Message:   tfe.String("Queued nightly from GH Actions via go-tfe"),
		Workspace: wk,
		AutoApply: tfe.Bool(true),
	}

	if !isDestroy {
		opts.ReplaceAddrs = []string{instanceAddr, tokenAddr}
	}

	run, err := client.Runs.Create(ctx, opts)
	if err != nil {
		return "", fmt.Errorf("failed to trigger run: %w", err)
	}
	log.Printf("Run created: %s", run.ID)

	return run.ID, nil
}

func waitForRun(ctx context.Context, client *tfe.Client, runID string) error {
	// The run should take about 5 minutes to complete;
	// polling the status of the run every 20 seconds or so
	// should be frequent enough. It's also long enough to ensure
	// no ticks are dropped.
	ticker := time.NewTicker(time.Second * 20)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("Context canceled: %w", ctx.Err())
		case <-ticker.C:
			run, err := client.Runs.Read(ctx, runID)
			if err != nil {
				return err
			}

			switch run.Status {
			case tfe.RunCanceled, tfe.RunErrored, tfe.RunDiscarded:
				return fmt.Errorf("Could not complete run: %s", string(run.Status))
			case tfe.RunApplied:
				// run is complete
				return nil
			default:
				log.Printf("Polling run %s, has status: %s", runID, string(run.Status))
			}
		}
	}
}
