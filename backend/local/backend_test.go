package local

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/terraform"
)

func TestLocal_impl(t *testing.T) {
	var _ backend.Enhanced = new(Local)
	var _ backend.Local = new(Local)
	var _ backend.CLI = new(Local)
	var _ backend.MultiState = new(Local)
}

func checkState(t *testing.T, path, expected string) {
	// Read the state
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	state, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected = strings.TrimSpace(expected)
	if actual != expected {
		t.Fatalf("state does not match! actual:\n%s\n\nexpected:\n%s", actual, expected)
	}
}

func TestLocal_addAndRemoveStates(t *testing.T) {
	defer testTmpDir(t)()
	dflt := backend.DefaultStateName
	expectedStates := []string{dflt}

	b := &Local{}
	states, current, err := b.States()
	if err != nil {
		t.Fatal(err)
	}

	if current != dflt {
		t.Fatalf("expected %q, got %q", dflt, current)
	}

	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected []string{%q}, got %q", dflt, states)
	}

	expectedA := "test_A"
	if err := b.ChangeState(expectedA); err != nil {
		t.Fatal(err)
	}

	states, current, err = b.States()
	if err != nil {
		t.Fatal(err)
	}
	if current != expectedA {
		t.Fatalf("expected %q, got %q", expectedA, current)
	}

	expectedStates = append(expectedStates, expectedA)
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	expectedB := "test_B"
	if err := b.ChangeState(expectedB); err != nil {
		t.Fatal(err)
	}

	states, current, err = b.States()
	if err != nil {
		t.Fatal(err)
	}
	if current != expectedB {
		t.Fatalf("expected %q, got %q", expectedB, current)
	}

	expectedStates = append(expectedStates, expectedB)
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	if err := b.DeleteState(expectedA); err != nil {
		t.Fatal(err)
	}

	states, current, err = b.States()
	if err != nil {
		t.Fatal(err)
	}
	if current != expectedB {
		t.Fatalf("expected %q, got %q", dflt, current)
	}

	expectedStates = []string{dflt, expectedB}
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	if err := b.DeleteState(expectedB); err != nil {
		t.Fatal(err)
	}

	states, current, err = b.States()
	if err != nil {
		t.Fatal(err)
	}
	if current != dflt {
		t.Fatalf("expected %q, got %q", dflt, current)
	}

	expectedStates = []string{dflt}
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	if err := b.DeleteState(dflt); err == nil {
		t.Fatal("expected error deleting default state")
	}
}

// verify the behavior with a backend that doesn't support multiple states
func TestLocal_noMultiStateBackend(t *testing.T) {
	type noMultiState struct {
		backend.Backend
	}

	b := &Local{
		Backend: &noMultiState{},
	}

	_, _, err := b.States()
	if err != ErrEnvNotSupported {
		t.Fatal("backend does not support environments.", err)
	}

	err = b.ChangeState("test")
	if err != ErrEnvNotSupported {
		t.Fatal("backend does not support environments.", err)
	}

	err = b.ChangeState("test")
	if err != ErrEnvNotSupported {
		t.Fatal("backend does not support environments.", err)
	}
}

// verify that the MultiState methods are dispatched to the correct Backend.
func TestLocal_multiStateBackend(t *testing.T) {
	defer testTmpDir(t)()

	dflt := backend.DefaultStateName
	expectedStates := []string{dflt}

	// make a second tmp dir for the sub-Backend.
	// we verify the corret backend was called by checking the paths.
	tmp, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	fmt.Println("second tmp:", tmp)

	b := &Local{
		Backend: &Local{
			workingDir: tmp,
		},
	}

	testA := "test_A"
	if err := b.ChangeState(testA); err != nil {
		t.Fatal(err)
	}

	states, current, err := b.States()
	if err != nil {
		t.Fatal(err)
	}
	if current != testA {
		t.Fatalf("expected %q, got %q", testA, current)
	}

	expectedStates = append(expectedStates, testA)
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

	// verify that no environment paths were created for the top-level Backend
	if _, err := os.Stat(DefaultDataDir); !os.IsNotExist(err) {
		t.Fatal("remote state operations should not have written local files")
	}

	if _, err := os.Stat(filepath.Join(DefaultEnvDir, testA)); !os.IsNotExist(err) {
		t.Fatal("remote state operations should not have written local files")
	}

	// remove the new state
	if err := b.DeleteState(testA); err != nil {
		t.Fatal(err)
	}

	states, current, err = b.States()
	if err != nil {
		t.Fatal(err)
	}
	if current != dflt {
		t.Fatalf("expected %q, got %q", dflt, current)
	}

	if !reflect.DeepEqual(states, expectedStates[:1]) {
		t.Fatalf("expected %q, got %q", expectedStates, states)
	}

}

// change into a tmp dir and return a deferable func to change back and cleanup
func testTmpDir(t *testing.T) func() {
	tmp, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatal(err)
	}

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	return func() {
		// ignore errors and try to clean up
		os.Chdir(old)
		os.RemoveAll(tmp)
	}
}
