// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"os"
	"testing"
)

// TestStateFile writes the given state to the path.
func TestStateFile(t *testing.T, path string, state *State) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	if err := WriteState(state, f); err != nil {
		t.Fatalf("err: %s", err)
	}
}
