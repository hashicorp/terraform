// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/terminal"
)

func TestStateLockerJSON_Locking(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewStateLocker(arguments.ViewJSON, view)

	v.Locking()

	// NOTE: No version log is created when the view is initialised, only a single log
	// is created via the `Locking` method call above.
	wantOutput := map[string]interface{}{
		"@level":   "info",
		"@message": "Acquiring state lock. This may take a few moments...",
		"@module":  "terraform.ui",
		"type":     "state_lock_acquire",
	}

	got := done(t).All()
	var gotMap map[string]interface{}
	if err := json.Unmarshal([]byte(got), &gotMap); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %s", err)
	}
	delete(gotMap, "@timestamp") // We cannot compare dynamic timestamps

	if diff := cmp.Diff(wantOutput, gotMap); diff != "" {
		t.Errorf("unexpected output diff:\n%s", diff)
	}
}

func TestStateLockerJSON_Unlocking(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	view := NewView(streams)
	view.Configure(&arguments.View{NoColor: true})
	v := NewStateLocker(arguments.ViewJSON, view)

	v.Unlocking()

	// NOTE: No version log is created when the view is initialised, only a single log
	// is created via the `Unlocking` method call above.
	wantOutput := map[string]interface{}{
		"@level":   "info",
		"@message": "Releasing state lock. This may take a few moments...",
		"@module":  "terraform.ui",
		"type":     "state_lock_release",
	}

	got := done(t).All()
	var gotMap map[string]interface{}
	if err := json.Unmarshal([]byte(got), &gotMap); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %s", err)
	}
	delete(gotMap, "@timestamp") // We cannot compare dynamic timestamps

	if diff := cmp.Diff(wantOutput, gotMap); diff != "" {
		t.Errorf("unexpected output diff:\n%s", diff)
	}
}
