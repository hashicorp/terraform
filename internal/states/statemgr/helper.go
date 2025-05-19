// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statemgr

// The functions in this file are helper wrappers for common sequences of
// operations done against full state managers.

import (
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/version"
)

// NewStateFile creates a new statefile.File object, with a newly-minted
// lineage identifier and serial 0, and returns a pointer to it.
func NewStateFile() *statefile.File {
	return &statefile.File{
		Lineage:          NewLineage(),
		TerraformVersion: version.SemVer,
		State:            states.NewState(),
	}
}

// RefreshAndRead refreshes the persistent snapshot in the given state manager
// and then returns it.
//
// This is a wrapper around calling RefreshState and then State on the given
// manager.
func RefreshAndRead(mgr Storage) (*states.State, error) {
	err := mgr.RefreshState()
	if err != nil {
		return nil, err
	}
	return mgr.State(), nil
}

// WriteAndPersist writes a snapshot of the given state to the given state
// manager's transient store and then immediately persists it.
//
// The caller must ensure that the given state is not concurrently modified
// while this function is running, but it is safe to modify it after this
// function has returned.
//
// If an error is returned, it is undefined whether the state has been saved
// to the transient store or not, and so the only safe response is to bail
// out quickly with a user-facing error. In situations where more control
// is required, call WriteState and PersistState on the state manager directly
// and handle their errors.
func WriteAndPersist(mgr Storage, state *states.State, schemas *schemarepo.Schemas) error {
	err := mgr.WriteState(state)
	if err != nil {
		return err
	}
	return mgr.PersistState(schemas)
}
