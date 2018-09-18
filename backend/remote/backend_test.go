package remote

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
)

func TestRemote(t *testing.T) {
	var _ backend.Enhanced = New(nil)
	var _ backend.CLI = New(nil)
}

func TestRemote_backendDefault(t *testing.T) {
	b := testBackendDefault(t)
	backend.TestBackendStates(t, b)
	backend.TestBackendStateLocks(t, b, b)
	backend.TestBackendStateForceUnlock(t, b, b)
}

func TestRemote_backendNoDefault(t *testing.T) {
	b := testBackendNoDefault(t)
	backend.TestBackendStates(t, b)
}

func TestRemote_config(t *testing.T) {
	cases := map[string]struct {
		config map[string]interface{}
		err    error
	}{
		"with_a_name": {
			config: map[string]interface{}{
				"organization": "hashicorp",
				"workspaces": []interface{}{
					map[string]interface{}{
						"name": "prod",
					},
				},
			},
			err: nil,
		},
		"with_a_prefix": {
			config: map[string]interface{}{
				"organization": "hashicorp",
				"workspaces": []interface{}{
					map[string]interface{}{
						"prefix": "my-app-",
					},
				},
			},
			err: nil,
		},
		"without_either_a_name_and_a_prefix": {
			config: map[string]interface{}{
				"organization": "hashicorp",
				"workspaces": []interface{}{
					map[string]interface{}{},
				},
			},
			err: errors.New("either workspace 'name' or 'prefix' is required"),
		},
		"with_both_a_name_and_a_prefix": {
			config: map[string]interface{}{
				"organization": "hashicorp",
				"workspaces": []interface{}{
					map[string]interface{}{
						"name":   "prod",
						"prefix": "my-app-",
					},
				},
			},
			err: errors.New("only one of workspace 'name' or 'prefix' is allowed"),
		},
		"with_an_unknown_host": {
			config: map[string]interface{}{
				"hostname":     "nonexisting.local",
				"organization": "hashicorp",
				"workspaces": []interface{}{
					map[string]interface{}{
						"name": "prod",
					},
				},
			},
			err: errors.New("host nonexisting.local does not provide a remote backend API"),
		},
	}

	for name, tc := range cases {
		s := testServer(t)
		b := New(testDisco(s))

		// Get the proper config structure
		rc, err := config.NewRawConfig(tc.config)
		if err != nil {
			t.Fatalf("%s: error creating raw config: %v", name, err)
		}
		conf := terraform.NewResourceConfig(rc)

		// Validate
		warns, errs := b.Validate(conf)
		if len(warns) > 0 {
			t.Fatalf("%s: validation warnings: %v", name, warns)
		}
		if len(errs) > 0 {
			t.Fatalf("%s: validation errors: %v", name, errs)
		}

		// Configure
		err = b.Configure(conf)
		if err != tc.err && err != nil && tc.err != nil && err.Error() != tc.err.Error() {
			t.Fatalf("%s: expected error %q, got: %q", name, tc.err, err)
		}
	}
}

func TestRemote_nonexistingOrganization(t *testing.T) {
	msg := "does not exist"

	b := testBackendNoDefault(t)
	b.organization = "nonexisting"

	if _, err := b.State("prod"); err == nil || !strings.Contains(err.Error(), msg) {
		t.Fatalf("expected %q error, got: %v", msg, err)
	}

	if err := b.DeleteState("prod"); err == nil || !strings.Contains(err.Error(), msg) {
		t.Fatalf("expected %q error, got: %v", msg, err)
	}

	if _, err := b.States(); err == nil || !strings.Contains(err.Error(), msg) {
		t.Fatalf("expected %q error, got: %v", msg, err)
	}
}

func TestRemote_addAndRemoveStatesDefault(t *testing.T) {
	b := testBackendDefault(t)
	if _, err := b.States(); err != backend.ErrNamedStatesNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrNamedStatesNotSupported, err)
	}

	if _, err := b.State(backend.DefaultStateName); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, err := b.State("prod"); err != backend.ErrNamedStatesNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrNamedStatesNotSupported, err)
	}

	if err := b.DeleteState(backend.DefaultStateName); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := b.DeleteState("prod"); err != backend.ErrNamedStatesNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrNamedStatesNotSupported, err)
	}
}

func TestRemote_addAndRemoveStatesNoDefault(t *testing.T) {
	b := testBackendNoDefault(t)
	states, err := b.States()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates := []string(nil)
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected states %#+v, got %#+v", expectedStates, states)
	}

	if _, err := b.State(backend.DefaultStateName); err != backend.ErrDefaultStateNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrDefaultStateNotSupported, err)
	}

	expectedA := "test_A"
	if _, err := b.State(expectedA); err != nil {
		t.Fatal(err)
	}

	states, err = b.States()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = append(expectedStates, expectedA)
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %#+v, got %#+v", expectedStates, states)
	}

	expectedB := "test_B"
	if _, err := b.State(expectedB); err != nil {
		t.Fatal(err)
	}

	states, err = b.States()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = append(expectedStates, expectedB)
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %#+v, got %#+v", expectedStates, states)
	}

	if err := b.DeleteState(backend.DefaultStateName); err != backend.ErrDefaultStateNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrDefaultStateNotSupported, err)
	}

	if err := b.DeleteState(expectedA); err != nil {
		t.Fatal(err)
	}

	states, err = b.States()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = []string{expectedB}
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %#+v got %#+v", expectedStates, states)
	}

	if err := b.DeleteState(expectedB); err != nil {
		t.Fatal(err)
	}

	states, err = b.States()
	if err != nil {
		t.Fatal(err)
	}

	expectedStates = []string(nil)
	if !reflect.DeepEqual(states, expectedStates) {
		t.Fatalf("expected %#+v, got %#+v", expectedStates, states)
	}
}
