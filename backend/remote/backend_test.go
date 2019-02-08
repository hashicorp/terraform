package remote

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/version"

	backendLocal "github.com/hashicorp/terraform/backend/local"
)

func TestRemote(t *testing.T) {
	var _ backend.Enhanced = New(nil)
	var _ backend.CLI = New(nil)
}

func TestRemote_backendDefault(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	backend.TestBackendStates(t, b)
	backend.TestBackendStateLocks(t, b, b)
	backend.TestBackendStateForceUnlock(t, b, b)
}

func TestRemote_backendNoDefault(t *testing.T) {
	b, bCleanup := testBackendNoDefault(t)
	defer bCleanup()

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
			err: errors.New("Failed to request discovery document"),
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
		if err != tc.err && err != nil && tc.err != nil && !strings.Contains(err.Error(), tc.err.Error()) {
			t.Fatalf("%s: expected error %q, got: %q", name, tc.err, err)
		}
	}
}

func TestRemote_versionConstraints(t *testing.T) {
	cases := map[string]struct {
		config     map[string]interface{}
		prerelease string
		version    string
		err        error
	}{
		"compatible version": {
			config: map[string]interface{}{
				"organization": "hashicorp",
				"workspaces": []interface{}{
					map[string]interface{}{
						"name": "prod",
					},
				},
			},
			version: "0.11.1",
		},
		"version too old": {
			config: map[string]interface{}{
				"organization": "hashicorp",
				"workspaces": []interface{}{
					map[string]interface{}{
						"name": "prod",
					},
				},
			},
			version: "0.10.1",
			err:     errors.New("upgrade Terraform to >= 0.11.8"),
		},
		"version too new": {
			config: map[string]interface{}{
				"organization": "hashicorp",
				"workspaces": []interface{}{
					map[string]interface{}{
						"name": "prod",
					},
				},
			},
			version: "0.12.0",
			err:     errors.New("downgrade Terraform to <= 0.11.11"),
		},
	}

	// Save and restore the actual version.
	p := version.Prerelease
	v := version.Version
	defer func() {
		version.Prerelease = p
		version.Version = v
	}()

	for name, tc := range cases {
		s := testServer(t)
		b := New(testDisco(s))

		// Set the version for this test.
		version.Prerelease = tc.prerelease
		version.Version = tc.version

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
		if err != tc.err && err != nil && tc.err != nil && !strings.Contains(err.Error(), tc.err.Error()) {
			t.Fatalf("%s: expected error %q, got: %q", name, tc.err, err)
		}
	}
}

func TestRemote_localBackend(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	local, ok := b.local.(*backendLocal.Local)
	if !ok {
		t.Fatalf("expected b.local to be \"*local.Local\", got: %T", b.local)
	}

	remote, ok := local.Backend.(*Remote)
	if !ok {
		t.Fatalf("expected local.Backend to be *remote.Remote, got: %T", remote)
	}
}

func TestRemote_addAndRemoveStatesDefault(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

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
	b, bCleanup := testBackendNoDefault(t)
	defer bCleanup()

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

func TestRemote_checkConstraints(t *testing.T) {
	b, bCleanup := testBackendDefault(t)
	defer bCleanup()

	cases := map[string]struct {
		constraints *disco.Constraints
		prerelease  string
		version     string
		result      string
	}{
		"compatible version": {
			constraints: &disco.Constraints{
				Minimum: "0.11.0",
				Maximum: "0.11.11",
			},
			version: "0.11.1",
			result:  "",
		},
		"version too old": {
			constraints: &disco.Constraints{
				Minimum: "0.11.0",
				Maximum: "0.11.11",
			},
			version: "0.10.1",
			result:  "upgrade Terraform to >= 0.11.0",
		},
		"version too new": {
			constraints: &disco.Constraints{
				Minimum: "0.11.0",
				Maximum: "0.11.11",
			},
			version: "0.12.0",
			result:  "downgrade Terraform to <= 0.11.11",
		},
		"version excluded - ordered": {
			constraints: &disco.Constraints{
				Minimum:   "0.11.0",
				Excluding: []string{"0.11.7", "0.11.8"},
				Maximum:   "0.11.11",
			},
			version: "0.11.7",
			result:  "upgrade Terraform to > 0.11.8",
		},
		"version excluded - unordered": {
			constraints: &disco.Constraints{
				Minimum:   "0.11.0",
				Excluding: []string{"0.11.8", "0.11.6"},
				Maximum:   "0.11.11",
			},
			version: "0.11.6",
			result:  "upgrade Terraform to > 0.11.8",
		},
		"list versions": {
			constraints: &disco.Constraints{
				Minimum: "0.11.0",
				Maximum: "0.11.11",
			},
			version: "0.10.1",
			result:  "versions >= 0.11.0, <= 0.11.11.",
		},
		"list exclusion": {
			constraints: &disco.Constraints{
				Minimum:   "0.11.0",
				Excluding: []string{"0.11.6"},
				Maximum:   "0.11.11",
			},
			version: "0.11.6",
			result:  "excluding version 0.11.6.",
		},
		"list exclusions": {
			constraints: &disco.Constraints{
				Minimum:   "0.11.0",
				Excluding: []string{"0.11.8", "0.11.6"},
				Maximum:   "0.11.11",
			},
			version: "0.11.6",
			result:  "excluding versions 0.11.6, 0.11.8.",
		},
	}

	for name, tc := range cases {
		version.Prerelease = tc.prerelease
		version.Version = tc.version

		// Check the constraints.
		err := b.checkConstraints(tc.constraints)
		if (err != nil || tc.result != "") &&
			(err == nil || !strings.Contains(err.Error(), tc.result)) {
			t.Fatalf("%s: unexpected constraints result: %v", name, err)
		}
	}
}
