// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/hashicorp/go-checkpoint"
	"github.com/hashicorp/terraform/internal/command"
	"github.com/hashicorp/terraform/internal/command/cliconfig"
	"go.opentelemetry.io/otel/codes"
)

func init() {
	checkpointResult = make(chan *checkpoint.CheckResponse, 1)
}

var checkpointResult chan *checkpoint.CheckResponse

// runCheckpoint runs a HashiCorp Checkpoint request. You can read about
// Checkpoint here: https://github.com/hashicorp/go-checkpoint.
func runCheckpoint(ctx context.Context, c *cliconfig.Config) {
	// If the user doesn't want checkpoint at all, then return.
	if c.DisableCheckpoint {
		log.Printf("[INFO] Checkpoint disabled. Not running.")
		checkpointResult <- nil
		return
	}

	ctx, span := tracer.Start(ctx, "HashiCorp Checkpoint")
	_ = ctx // prevent staticcheck from complaining to avoid a maintenence hazard of having the wrong ctx in scope here
	defer span.End()

	configDir, err := cliconfig.ConfigDir()
	if err != nil {
		log.Printf("[ERR] Checkpoint setup error: %s", err)
		checkpointResult <- nil
		return
	}

	version := Version
	if VersionPrerelease != "" {
		version += fmt.Sprintf("-%s", VersionPrerelease)
	}

	signaturePath := filepath.Join(configDir, "checkpoint_signature")
	if c.DisableCheckpointSignature {
		log.Printf("[INFO] Checkpoint signature disabled")
		signaturePath = ""
	}

	resp, err := checkpoint.Check(&checkpoint.CheckParams{
		Product:       "terraform",
		Version:       version,
		SignatureFile: signaturePath,
		CacheFile:     filepath.Join(configDir, "checkpoint_cache"),
	})
	if err != nil {
		log.Printf("[ERR] Checkpoint error: %s", err)
		span.SetStatus(codes.Error, err.Error())
		resp = nil
	} else {
		span.SetStatus(codes.Ok, "checkpoint request succeeded")
	}

	checkpointResult <- resp
}

// commandVersionCheck implements command.VersionCheckFunc and is used
// as the version checker.
func commandVersionCheck() (command.VersionCheckInfo, error) {
	// Wait for the result to come through
	info := <-checkpointResult
	if info == nil {
		var zero command.VersionCheckInfo
		return zero, nil
	}

	// Build the alerts that we may have received about our version
	alerts := make([]string, len(info.Alerts))
	for i, a := range info.Alerts {
		alerts[i] = a.Message
	}

	return command.VersionCheckInfo{
		Outdated: info.Outdated,
		Latest:   info.CurrentVersion,
		Alerts:   alerts,
	}, nil
}
