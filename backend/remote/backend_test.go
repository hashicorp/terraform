package remote

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/zclconf/go-cty/cty"

	backendLocal "github.com/hashicorp/terraform/backend/local"
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
		config  cty.Value
		confErr string
		valErr  string
	}{
		"with_a_nonexisting_organization": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("nonexisting"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
				}),
			}),
			confErr: "organization nonexisting does not exist",
		},
		"with_an_unknown_host": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.StringVal("nonexisting.local"),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
				}),
			}),
			confErr: "Host nonexisting.local does not provide a remote backend API",
		},
		"with_a_name": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.NullVal(cty.String),
				}),
			}),
		},
		"with_a_prefix": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.NullVal(cty.String),
					"prefix": cty.StringVal("my-app-"),
				}),
			}),
		},
		"without_either_a_name_and_a_prefix": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.NullVal(cty.String),
					"prefix": cty.NullVal(cty.String),
				}),
			}),
			valErr: `Either workspace "name" or "prefix" is required`,
		},
		"with_both_a_name_and_a_prefix": {
			config: cty.ObjectVal(map[string]cty.Value{
				"hostname":     cty.NullVal(cty.String),
				"organization": cty.StringVal("hashicorp"),
				"token":        cty.NullVal(cty.String),
				"workspaces": cty.ObjectVal(map[string]cty.Value{
					"name":   cty.StringVal("prod"),
					"prefix": cty.StringVal("my-app-"),
				}),
			}),
			valErr: `Only one of workspace "name" or "prefix" is allowed`,
		},
	}

	for name, tc := range cases {
		s := testServer(t)
		b := New(testDisco(s))

		// Validate
		valDiags := b.ValidateConfig(tc.config)
		if (valDiags.Err() == nil && tc.valErr != "") ||
			(valDiags.Err() != nil && !strings.Contains(valDiags.Err().Error(), tc.valErr)) {
			t.Fatalf("%s: unexpected validation result: %v", name, valDiags.Err())
		}

		// Configure
		confDiags := b.Configure(tc.config)
		if (confDiags.Err() == nil && tc.confErr != "") ||
			(confDiags.Err() != nil && !strings.Contains(confDiags.Err().Error(), tc.confErr)) {
			t.Fatalf("%s: unexpected configure result: %v", name, confDiags.Err())
		}
	}
}

func TestRemote_localBackend(t *testing.T) {
	b := testBackendDefault(t)

	local, ok := b.local.(*backendLocal.Local)
	if !ok {
		t.Fatalf("expected b.local to be \"*local.Local\", got: %T", b.local)
	}

	remote, ok := local.Backend.(*Remote)
	if !ok {
		t.Fatalf("expected local.Backend to be *remote.Remote, got: %T", remote)
	}
}

func TestRemote_addAndRemoveWorkspacesDefault(t *testing.T) {
	b := testBackendDefault(t)
	if _, err := b.Workspaces(); err != backend.ErrWorkspacesNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrWorkspacesNotSupported, err)
	}

	if _, err := b.StateMgr(backend.DefaultStateName); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if _, err := b.StateMgr("prod"); err != backend.ErrWorkspacesNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrWorkspacesNotSupported, err)
	}

	if err := b.DeleteWorkspace(backend.DefaultStateName); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := b.DeleteWorkspace("prod"); err != backend.ErrWorkspacesNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrWorkspacesNotSupported, err)
	}
}

func TestRemote_addAndRemoveWorkspacesNoDefault(t *testing.T) {
	b := testBackendNoDefault(t)
	states, err := b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedWorkspaces := []string(nil)
	if !reflect.DeepEqual(states, expectedWorkspaces) {
		t.Fatalf("expected states %#+v, got %#+v", expectedWorkspaces, states)
	}

	if _, err := b.StateMgr(backend.DefaultStateName); err != backend.ErrDefaultWorkspaceNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrDefaultWorkspaceNotSupported, err)
	}

	expectedA := "test_A"
	if _, err := b.StateMgr(expectedA); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedWorkspaces = append(expectedWorkspaces, expectedA)
	if !reflect.DeepEqual(states, expectedWorkspaces) {
		t.Fatalf("expected %#+v, got %#+v", expectedWorkspaces, states)
	}

	expectedB := "test_B"
	if _, err := b.StateMgr(expectedB); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedWorkspaces = append(expectedWorkspaces, expectedB)
	if !reflect.DeepEqual(states, expectedWorkspaces) {
		t.Fatalf("expected %#+v, got %#+v", expectedWorkspaces, states)
	}

	if err := b.DeleteWorkspace(backend.DefaultStateName); err != backend.ErrDefaultWorkspaceNotSupported {
		t.Fatalf("expected error %v, got %v", backend.ErrDefaultWorkspaceNotSupported, err)
	}

	if err := b.DeleteWorkspace(expectedA); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedWorkspaces = []string{expectedB}
	if !reflect.DeepEqual(states, expectedWorkspaces) {
		t.Fatalf("expected %#+v got %#+v", expectedWorkspaces, states)
	}

	if err := b.DeleteWorkspace(expectedB); err != nil {
		t.Fatal(err)
	}

	states, err = b.Workspaces()
	if err != nil {
		t.Fatal(err)
	}

	expectedWorkspaces = []string(nil)
	if !reflect.DeepEqual(states, expectedWorkspaces) {
		t.Fatalf("expected %#+v, got %#+v", expectedWorkspaces, states)
	}
}
