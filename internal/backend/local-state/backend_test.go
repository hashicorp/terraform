package local_state

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
)

func TestLocal_backend(t *testing.T) {
	backend.TestTmpDir(t)
	b := New()
	backend.TestBackendStates(t, b)
	backend.TestBackendStateLocks(t, b, b)
}

func TestLocal_addAndRemoveStates(t *testing.T) {
	backend.TestTmpDir(t)
	dflt := backend.DefaultStateName
	expectedStates := []string{dflt}

	b := New()
	states, err := b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected []string{%q}, got %q", dflt, states)
	}

	expectedA := "test_A"
	if _, err := b.StateMgr(expectedA); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = append(expectedStates, expectedA)
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	expectedB := "test_B"
	if _, err := b.StateMgr(expectedB); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = append(expectedStates, expectedB)
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	if err := b.DeleteWorkspace(expectedA, true); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = []string{dflt, expectedB}
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	if err := b.DeleteWorkspace(expectedB, true); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = []string{dflt}
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	if err := b.DeleteWorkspace(dflt, true); err == nil {
		t.Fatal("expected error deleting default state")
	}
}
