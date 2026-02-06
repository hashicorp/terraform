// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package inmem

import (
	"testing"

	statespkg "github.com/hashicorp/terraform/internal/states"
)

func ReadState(t *testing.T, wsName string) *statespkg.State {
	states.Lock()
	defer states.Unlock()

	stateMgr, ok := states.m[wsName]
	if !ok {
		t.Fatalf("state not found for workspace %s", wsName)
	}

	return stateMgr.State()
}

func ReadWorkspaces(t *testing.T) []string {
	states.Lock()
	defer states.Unlock()

	workspaces := make([]string, 0, len(states.m))
	for wsName := range states.m {
		workspaces = append(workspaces, wsName)
	}

	return workspaces
}
