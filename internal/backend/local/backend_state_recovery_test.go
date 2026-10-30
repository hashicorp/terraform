// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package local

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestLocal_backupStateForErrorClosesFile(t *testing.T) {
	t.Chdir(t.TempDir())
	streams, _ := terminal.StreamsForTesting(t)
	view := views.NewOperation(arguments.ViewHuman, false, views.NewView(streams))
	want := statefile.New(statemgr.TestFullInitialState(), "test-lineage", 42)

	diags := New().backupStateForError(want, errors.New("backend unavailable"), view)
	if len(diags) != 2 || diags[0].Description().Summary != "Failed to save state" || diags[1].Description().Summary != "Failed to persist state to backend" {
		t.Fatalf("unexpected recovery diagnostics: %v", diags.Err())
	}

	data, err := os.ReadFile("errored.tfstate")
	if err != nil {
		t.Fatal(err)
	}
	got, err := statefile.Read(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if !got.State.Equal(want.State) || got.Lineage != want.Lineage || got.Serial != want.Serial {
		t.Fatal("recovery file did not preserve the state and its metadata")
	}
	if err := os.Remove("errored.tfstate"); err != nil {
		t.Fatalf("recovery file remains open after saving: %s", err)
	}
}
