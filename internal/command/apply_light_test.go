// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/zclconf/go-cty/cty"
)

func TestApply_light(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-light"), td)
	t.Chdir(td)

	testState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				// No refresh should occur because the state + config will produce a no-op
				AttrsJSON: []byte(`{"id":"bar","ami": "bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "baz",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				// Will prompt a refresh since the config value (ami) has changed
				AttrsJSON: []byte(`{"id":"quux","ami": "old-value"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})

	statePath := testStateFile(t, testState)

	p := applyFixtureProvider()
	fooRefreshed := false
	bazRefreshed := false
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		amiVal := req.PriorState.GetAttr("ami")
		if amiVal.RawEquals(cty.StringVal("bar")) {
			fooRefreshed = true
		}
		if amiVal.RawEquals(cty.StringVal("old-value")) {
			bazRefreshed = true
		}
		return providers.ReadResourceResponse{
			NewState: req.PriorState,
		}
	}

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		"-light",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if fooRefreshed {
		t.Error(`Unexpected call to ReadResource for the "foo" resource. This resource should not be refreshed with ` +
			`the -light flag as the configuration did not change from prior state.`)
	}

	if !bazRefreshed {
		t.Error(`Expected a call to ReadResource for the "baz" resource but received none. This resource should be refreshed with ` +
			`the -light flag as the configuration changed from prior state.`)
	}
}

func TestApply_light_invalid_flags(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-light"), td)
	t.Chdir(td)

	testCases := map[string]struct {
		args    []string
		wantErr string
	}{
		"destroy": {
			args:    []string{"-light", "-destroy"},
			wantErr: "Incompatible plan mode options",
		},
		"refresh-only": {
			args:    []string{"-light", "-refresh-only"},
			wantErr: "Incompatible plan mode options",
		},
		"refresh-false": {
			args:    []string{"-light", "-refresh=false"},
			wantErr: "Incompatible refresh options",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			p := applyFixtureProvider()
			view, done := testView(t)
			c := &ApplyCommand{
				Meta: Meta{
					testingOverrides: metaOverridesForProvider(p),
					View:             view,
				},
			}

			code := c.Run(append(tc.args, "-no-color"))
			output := done(t)
			if code != 1 {
				t.Fatalf("expected error exit code 1, got %d\n\n%s", code, output.Stdout())
			}
			if got := output.Stderr(); !strings.Contains(got, tc.wantErr) {
				t.Fatalf("wrong error, want %q, got:\n%s", tc.wantErr, got)
			}
		})
	}
}
