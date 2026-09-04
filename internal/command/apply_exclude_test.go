// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestApply_excludes(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("apply-exclude"), td)
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
	p := applyFixtureProvider()

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
	p.ApplyResourceChangeFn = func(req providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
		if !req.PlannedState.IsNull() {
			if ami := req.PlannedState.GetAttr("ami"); strings.Contains(ami.AsString(), "excluded") {
				return providers.ApplyResourceChangeResponse{
					Diagnostics: tfdiags.Diagnostics{
						tfdiags.Sourceless(tfdiags.Error,
							"Excluded resource applied",
							"A resource that was expected to be excluded was applied",
						),
					},
				}
			}
		}
		return providers.ApplyResourceChangeResponse{NewState: cty.UnknownAsNull(req.PlannedState)}
	}

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			AllowExperimentalFeatures: true,
			testingOverrides:          metaOverridesForProvider(p),
			View:                      view,
		},
	}

	args := []string{
		"-state", statePath,
		"-auto-approve",
		"-allow-deferral",
		"-exclude=test_instance.excludeme",
		"-no-color",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	expectedOutput := "Apply complete! Resources: 0 added, 1 changed, 0 destroyed."
	if got := output.Stdout(); !strings.Contains(got, expectedOutput) {
		t.Fatalf("expected apply output to contain %q, got:\n%q", expectedOutput, got)
	}
}
