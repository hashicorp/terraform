// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"strings"
	"testing"
)

func TestPlan_excluded(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("excluded"), td)
	t.Chdir(td)

	view, done := testView(t)
	c := &PlanCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(planFixtureProvider()),
			View:             view,
		},
	}

	args := []string{"-exclude", "test_instance.baz"}
	code := c.Run(args)
	output := done(t)
	if code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
	}

	got := output.Stdout()
	if want := "2 to add, 0 to change, 0 to destroy"; !strings.Contains(got, want) {
		t.Fatalf("expected plan summary %q, got:\n%s", want, got)
	}
}
