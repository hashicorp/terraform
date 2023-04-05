// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package local

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
)

func TestStateHook_impl(t *testing.T) {
	var _ terraform.Hook = new(StateHook)
}

func TestStateHook(t *testing.T) {
	is := statemgr.NewTransientInMemory(nil)
	var hook terraform.Hook = &StateHook{StateMgr: is}

	s := statemgr.TestFullInitialState()
	action, err := hook.PostStateUpdate(s)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("bad: %v", action)
	}
	if !is.State().Equal(s) {
		t.Fatalf("bad state: %#v", is.State())
	}
}

func TestStateHookStopping(t *testing.T) {
	is := &testPersistentState{}
	hook := &StateHook{
		StateMgr:        is,
		Schemas:         &terraform.Schemas{},
		PersistInterval: 4 * time.Hour,
		intermediatePersist: IntermediateStatePersistInfo{
			LastPersist: time.Now(),
		},
	}

	s := statemgr.TestFullInitialState()
	action, err := hook.PostStateUpdate(s)
	if err != nil {
		t.Fatalf("unexpected error from PostStateUpdate: %s", err)
	}
	if got, want := action, terraform.HookActionContinue; got != want {
		t.Fatalf("wrong hookaction %#v; want %#v", got, want)
	}
	if is.Written == nil || !is.Written.Equal(s) {
		t.Fatalf("mismatching state written")
	}
	if is.Persisted != nil {
		t.Fatalf("persisted too soon")
	}

	// We'll now force lastPersist to be long enough ago that persisting
	// should be due on the next call.
	hook.intermediatePersist.LastPersist = time.Now().Add(-5 * time.Hour)
	hook.PostStateUpdate(s)
	if is.Written == nil || !is.Written.Equal(s) {
		t.Fatalf("mismatching state written")
	}
	if is.Persisted == nil || !is.Persisted.Equal(s) {
		t.Fatalf("mismatching state persisted")
	}
	hook.PostStateUpdate(s)
	if is.Written == nil || !is.Written.Equal(s) {
		t.Fatalf("mismatching state written")
	}
	if is.Persisted == nil || !is.Persisted.Equal(s) {
		t.Fatalf("mismatching state persisted")
	}

	gotLog := is.CallLog
	wantLog := []string{
		// Initial call before we reset lastPersist
		"WriteState",

		// Write and then persist after we reset lastPersist
		"WriteState",
		"PersistState",

		// Final call when persisting wasn't due yet.
		"WriteState",
	}
	if diff := cmp.Diff(wantLog, gotLog); diff != "" {
		t.Fatalf("wrong call log so far\n%s", diff)
	}

	// We'll reset the log now before we try seeing what happens after
	// we use "Stopped".
	is.CallLog = is.CallLog[:0]
	is.Persisted = nil

	hook.Stopping()
	if is.Persisted == nil || !is.Persisted.Equal(s) {
		t.Fatalf("mismatching state persisted")
	}

	is.Persisted = nil
	hook.PostStateUpdate(s)
	if is.Persisted == nil || !is.Persisted.Equal(s) {
		t.Fatalf("mismatching state persisted")
	}
	is.Persisted = nil
	hook.PostStateUpdate(s)
	if is.Persisted == nil || !is.Persisted.Equal(s) {
		t.Fatalf("mismatching state persisted")
	}

	gotLog = is.CallLog
	wantLog = []string{
		// "Stopping" immediately persisted
		"PersistState",

		// PostStateUpdate then writes and persists on every call,
		// on the assumption that we're now bailing out after
		// being cancelled and trying to save as much state as we can.
		"WriteState",
		"PersistState",
		"WriteState",
		"PersistState",
	}
	if diff := cmp.Diff(wantLog, gotLog); diff != "" {
		t.Fatalf("wrong call log once in stopping mode\n%s", diff)
	}
}

func TestStateHookCustomPersistRule(t *testing.T) {
	is := &testPersistentStateThatRefusesToPersist{}
	hook := &StateHook{
		StateMgr:        is,
		Schemas:         &terraform.Schemas{},
		PersistInterval: 4 * time.Hour,
		intermediatePersist: IntermediateStatePersistInfo{
			LastPersist: time.Now(),
		},
	}

	s := statemgr.TestFullInitialState()
	action, err := hook.PostStateUpdate(s)
	if err != nil {
		t.Fatalf("unexpected error from PostStateUpdate: %s", err)
	}
	if got, want := action, terraform.HookActionContinue; got != want {
		t.Fatalf("wrong hookaction %#v; want %#v", got, want)
	}
	if is.Written == nil || !is.Written.Equal(s) {
		t.Fatalf("mismatching state written")
	}
	if is.Persisted != nil {
		t.Fatalf("persisted too soon")
	}

	// We'll now force lastPersist to be long enough ago that persisting
	// should be due on the next call.
	hook.intermediatePersist.LastPersist = time.Now().Add(-5 * time.Hour)
	hook.PostStateUpdate(s)
	if is.Written == nil || !is.Written.Equal(s) {
		t.Fatalf("mismatching state written")
	}
	if is.Persisted != nil {
		t.Fatalf("has a persisted state, but shouldn't")
	}
	hook.PostStateUpdate(s)
	if is.Written == nil || !is.Written.Equal(s) {
		t.Fatalf("mismatching state written")
	}
	if is.Persisted != nil {
		t.Fatalf("has a persisted state, but shouldn't")
	}

	gotLog := is.CallLog
	wantLog := []string{
		// Initial call before we reset lastPersist
		"WriteState",
		"ShouldPersistIntermediateState",
		// Previous call should return false, preventing a "PersistState" call

		// Write and then decline to persist
		"WriteState",
		"ShouldPersistIntermediateState",
		// Previous call should return false, preventing a "PersistState" call

		// Final call before we start "stopping".
		"WriteState",
		"ShouldPersistIntermediateState",
		// Previous call should return false, preventing a "PersistState" call
	}
	if diff := cmp.Diff(wantLog, gotLog); diff != "" {
		t.Fatalf("wrong call log so far\n%s", diff)
	}

	// We'll reset the log now before we try seeing what happens after
	// we use "Stopped".
	is.CallLog = is.CallLog[:0]
	is.Persisted = nil

	hook.Stopping()
	if is.Persisted == nil || !is.Persisted.Equal(s) {
		t.Fatalf("mismatching state persisted")
	}

	is.Persisted = nil
	hook.PostStateUpdate(s)
	if is.Persisted == nil || !is.Persisted.Equal(s) {
		t.Fatalf("mismatching state persisted")
	}
	is.Persisted = nil
	hook.PostStateUpdate(s)
	if is.Persisted == nil || !is.Persisted.Equal(s) {
		t.Fatalf("mismatching state persisted")
	}

	gotLog = is.CallLog
	wantLog = []string{
		"ShouldPersistIntermediateState",
		// Previous call should return true, allowing the following "PersistState" call
		"PersistState",
		"WriteState",
		"ShouldPersistIntermediateState",
		// Previous call should return true, allowing the following "PersistState" call
		"PersistState",
		"WriteState",
		"ShouldPersistIntermediateState",
		// Previous call should return true, allowing the following "PersistState" call
		"PersistState",
	}
	if diff := cmp.Diff(wantLog, gotLog); diff != "" {
		t.Fatalf("wrong call log once in stopping mode\n%s", diff)
	}
}

type testPersistentState struct {
	CallLog []string

	Written   *states.State
	Persisted *states.State
}

var _ statemgr.Writer = (*testPersistentState)(nil)
var _ statemgr.Persister = (*testPersistentState)(nil)

func (sm *testPersistentState) WriteState(state *states.State) error {
	sm.CallLog = append(sm.CallLog, "WriteState")
	sm.Written = state
	return nil
}

func (sm *testPersistentState) PersistState(schemas *terraform.Schemas) error {
	if schemas == nil {
		return fmt.Errorf("no schemas")
	}
	sm.CallLog = append(sm.CallLog, "PersistState")
	sm.Persisted = sm.Written
	return nil
}

type testPersistentStateThatRefusesToPersist struct {
	CallLog []string

	Written   *states.State
	Persisted *states.State
}

var _ statemgr.Writer = (*testPersistentStateThatRefusesToPersist)(nil)
var _ statemgr.Persister = (*testPersistentStateThatRefusesToPersist)(nil)
var _ IntermediateStateConditionalPersister = (*testPersistentStateThatRefusesToPersist)(nil)

func (sm *testPersistentStateThatRefusesToPersist) WriteState(state *states.State) error {
	sm.CallLog = append(sm.CallLog, "WriteState")
	sm.Written = state
	return nil
}

func (sm *testPersistentStateThatRefusesToPersist) PersistState(schemas *terraform.Schemas) error {
	if schemas == nil {
		return fmt.Errorf("no schemas")
	}
	sm.CallLog = append(sm.CallLog, "PersistState")
	sm.Persisted = sm.Written
	return nil
}

// ShouldPersistIntermediateState implements IntermediateStateConditionalPersister
func (sm *testPersistentStateThatRefusesToPersist) ShouldPersistIntermediateState(info *IntermediateStatePersistInfo) bool {
	sm.CallLog = append(sm.CallLog, "ShouldPersistIntermediateState")
	return info.ForcePersist
}
