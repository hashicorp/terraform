// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"
)

func TestRefresh_excluded(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("excluded"), td)
	t.Chdir(td)

	statePath := testStateFile(t, excludeFixtureState(t))

	view, done := testView(t)
	c := &RefreshCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(planFixtureProvider()),
			View:             view,
		},
	}

	args := []string{"-state", statePath, "-exclude", "test_instance.baz"}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	got := output.Stdout()
	if want := "test_instance.foo: Refreshing"; !strings.Contains(got, want) {
		t.Errorf("expected output to contain %q, got:\n%s", want, got)
	}
	if want := "test_instance.bar: Refreshing"; !strings.Contains(got, want) {
		t.Errorf("expected output to contain %q, got:\n%s", want, got)
	}
	if doNotWant := "test_instance.baz: Refreshing"; strings.Contains(got, doNotWant) {
		t.Errorf("expected output NOT to contain %q, got:\n%s", doNotWant, got)
	}
	if doNotWant := "test_instance.quux: Refreshing"; strings.Contains(got, doNotWant) {
		t.Errorf("expected output NOT to contain %q, got:\n%s", doNotWant, got)
	}
}
