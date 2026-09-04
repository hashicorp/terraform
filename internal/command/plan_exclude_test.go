// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestPlan_excludes(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-exclude"), td)
	t.Chdir(td)

	testState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo-id","ami":"changeme"}`),
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
				Name: "excludeme",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"excluded-id","ami":"excluded"}`),
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
				Name: "dependent_on_excludeme",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"dep-id","ami":"excluded-id"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})

	statePath := testStateFile(t, testState)
	p := planFixtureProvider()

	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		if ami := req.PriorState.GetAttr("ami"); strings.Contains(ami.AsString(), "excluded") {
			return providers.ReadResourceResponse{
				Diagnostics: tfdiags.Diagnostics{
					tfdiags.Sourceless(tfdiags.Error,
						"Excluded resource refreshed",
						"A resource that was expected to be excluded was refreshed",
					),
				},
			}
		}
		return providers.ReadResourceResponse{NewState: req.PriorState}
	}
	p.PlanResourceChangeFn = func(req providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
		if !req.ProposedNewState.IsNull() {
			if ami := req.ProposedNewState.GetAttr("ami"); strings.Contains(ami.AsString(), "excluded") {
				return providers.PlanResourceChangeResponse{
					Diagnostics: tfdiags.Diagnostics{
						tfdiags.Sourceless(tfdiags.Error,
							"Excluded resource planned",
							"A resource that was expected to be excluded was planned",
						),
					},
				}
			}
		}
		return providers.PlanResourceChangeResponse{PlannedState: req.ProposedNewState}
	}

	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			AllowExperimentalFeatures: true,
			testingOverrides:          metaOverridesForProvider(p),
			View:                      view,
		},
	}

	outPath := filepath.Join(td, "test.plan")
	args := []string{
		"-state", statePath,
		"-out", outPath,
		"-allow-deferral",
		"-exclude=test_instance.excludeme",
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	plan := testReadPlan(t, outPath)

	if plan.Complete {
		t.Error("expected partial plan but received a complete plan")
	}
	if len(plan.DeferredResources) != 2 {
		t.Errorf("expected 2 deferred resources, got: %d", len(plan.DeferredResources))
	}
	if len(plan.Changes.Resources) != 1 {
		t.Errorf("expected 1 resource change, got: %d", len(plan.Changes.Resources))
	}
}

func TestPlan_excludes_invalid_flags(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-exclude"), td)
	t.Chdir(td)

	testCases := map[string]struct {
		args    []string
		wantErr string
	}{
		"exclude-with-target": {
			args: []string{
				"-allow-deferral",
				"-exclude=test_instance.excludeme",
				"-target=test_instance.foo",
			},
			wantErr: "Incompatible plan mode options",
		},
		"exclude-without-allow-deferral": {
			args:    []string{"-exclude=test_instance.excludeme"},
			wantErr: "Incompatible plan mode options",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			p := planFixtureProvider()
			view, done := testView(t)
			c := &PlanCommand{
				Meta: Meta{
					AllowExperimentalFeatures: true,
					testingOverrides:          metaOverridesForProvider(p),
					View:                      view,
				},
			}

			code := c.Run(append(tc.args, "-no-color"))
			output := done(t)
			if code != 1 {
				t.Fatalf("expected error exit code 1, got %d\n\n%s", code, output.Stdout())
			}
			if got := output.Stderr(); !strings.Contains(got, tc.wantErr) {
				t.Fatalf("wrong error: want %q, got:\n%s", tc.wantErr, got)
			}
		})
	}
}

func TestPlan_excludes_experimental(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("plan-exclude"), td)
	t.Chdir(td)

	p := planFixtureProvider()
	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			AllowExperimentalFeatures: false,
			testingOverrides:          metaOverridesForProvider(p),
			View:                      view,
		},
	}

	args := []string{
		"-allow-deferral",
		"-exclude=test_instance.excludeme",
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)
	if code != 1 {
		t.Fatalf("expected error exit code 1, got %d\n\n%s", code, output.Stdout())
	}
	expectedErr := "only valid in experimental builds of Terraform"
	if got := output.Stderr(); !strings.Contains(got, expectedErr) {
		t.Fatalf("expected an error mentioning experimental, got:\n%s", got)
	}
}
