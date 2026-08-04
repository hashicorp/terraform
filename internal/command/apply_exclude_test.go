// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

func TestApply_excluded(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("excluded"), td)
	t.Chdir(td)

	view, done := testView(t)
	c := &ApplyCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(planFixtureProvider()),
			View:             view,
		},
	}

	args := []string{"-auto-approve", "-exclude", "test_instance.baz"}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if got, want := output.Stdout(), "2 added, 0 changed, 0 destroyed"; !strings.Contains(got, want) {
		t.Fatalf("expected apply summary %q, got:\n%s", want, got)
	}
}

func TestApplyDestroy_excluded(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("excluded"), td)
	t.Chdir(td)

	statePath := testStateFile(t, excludeFixtureState(t))

	view, done := testView(t)
	c := &ApplyCommand{
		Destroy: true,
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(planFixtureProvider()),
			View:             view,
		},
	}

	args := []string{
		"-auto-approve",
		"-state", statePath,
		// TODO:@austinvalle: I know the dependencies in destroy work differently, so excluding baz won't
		// automatically exclude quux. Investigate this further to understand it and see if there
		// are some use-cases to document for practitioners to understand this. (targeting might not
		// have this problem?)
		"-exclude", "test_instance.baz",
		"-exclude", "test_instance.quux",
	}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	if got, want := output.Stdout(), "Resources: 2 destroyed."; !strings.Contains(got, want) {
		t.Fatalf("expected destroy summary %q, got:\n%s", want, got)
	}
}

func excludeFixtureState(t *testing.T) *states.State {
	t.Helper()

	return states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"foo"}`),
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
				Name: "bar",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
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
				AttrsJSON: []byte(`{"id":"baz"}`),
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
				Name: "quux",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"quux"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("test"),
				Module:   addrs.RootModule,
			},
		)
	})
}
