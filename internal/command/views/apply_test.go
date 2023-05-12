// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/zclconf/go-cty/cty"
)

// This test is mostly because I am paranoid about having two consecutive
// boolean arguments.
func TestApply_new(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	defer done(t)
	v := NewApply(arguments.ViewHuman, false, NewView(streams).SetRunningInAutomation(true))
	hv, ok := v.(*ApplyHuman)
	if !ok {
		t.Fatalf("unexpected return type %t", v)
	}

	if hv.destroy != false {
		t.Fatalf("unexpected destroy value")
	}

	if hv.inAutomation != true {
		t.Fatalf("unexpected inAutomation value")
	}
}

// Basic test coverage of Outputs, since most of its functionality is tested
// elsewhere.
func TestApplyHuman_outputs(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewApply(arguments.ViewHuman, false, NewView(streams))

	v.Outputs(map[string]*states.OutputValue{
		"foo": {Value: cty.StringVal("secret")},
	})

	got := done(t).Stdout()
	for _, want := range []string{"Outputs:", `foo = "secret"`} {
		if !strings.Contains(got, want) {
			t.Errorf("wrong result\ngot:  %q\nwant: %q", got, want)
		}
	}
}

// Outputs should do nothing if there are no outputs to render.
func TestApplyHuman_outputsEmpty(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewApply(arguments.ViewHuman, false, NewView(streams))

	v.Outputs(map[string]*states.OutputValue{})

	got := done(t).Stdout()
	if got != "" {
		t.Errorf("output should be empty, but got: %q", got)
	}
}

// Ensure that the correct view type and in-automation settings propagate to the
// Operation view.
func TestApplyHuman_operation(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	defer done(t)
	v := NewApply(arguments.ViewHuman, false, NewView(streams).SetRunningInAutomation(true)).Operation()
	if hv, ok := v.(*OperationHuman); !ok {
		t.Fatalf("unexpected return type %t", v)
	} else if hv.inAutomation != true {
		t.Fatalf("unexpected inAutomation value on Operation view")
	}
}

// This view is used for both apply and destroy commands, so the help output
// needs to cover both.
func TestApplyHuman_help(t *testing.T) {
	testCases := map[string]bool{
		"apply":   false,
		"destroy": true,
	}

	for name, destroy := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewApply(arguments.ViewHuman, destroy, NewView(streams))
			v.HelpPrompt()
			got := done(t).Stderr()
			if !strings.Contains(got, name) {
				t.Errorf("wrong result\ngot:  %q\nwant: %q", got, name)
			}
		})
	}
}

// Hooks and ResourceCount are tangled up and easiest to test together.
func TestApply_resourceCount(t *testing.T) {
	testCases := map[string]struct {
		destroy   bool
		want      string
		importing bool
	}{
		"apply": {
			false,
			"Apply complete! Resources: 1 added, 2 changed, 3 destroyed.",
			false,
		},
		"destroy": {
			true,
			"Destroy complete! Resources: 3 destroyed.",
			false,
		},
		"import": {
			false,
			"Apply complete! Resources: 1 imported, 1 added, 2 changed, 3 destroyed.",
			true,
		},
	}

	// For compatibility reasons, these tests should hold true for both human
	// and JSON output modes
	views := []arguments.ViewType{arguments.ViewHuman, arguments.ViewJSON}

	for name, tc := range testCases {
		for _, viewType := range views {
			t.Run(fmt.Sprintf("%s (%s view)", name, viewType), func(t *testing.T) {
				streams, done := terminal.StreamsForTesting(t)
				v := NewApply(viewType, tc.destroy, NewView(streams))
				hooks := v.Hooks()

				var count *countHook
				for _, hook := range hooks {
					if ch, ok := hook.(*countHook); ok {
						count = ch
					}
				}
				if count == nil {
					t.Fatalf("expected Hooks to include a countHook: %#v", hooks)
				}

				count.Added = 1
				count.Changed = 2
				count.Removed = 3

				if tc.importing {
					count.Imported = 1
				}

				v.ResourceCount("")

				got := done(t).Stdout()
				if !strings.Contains(got, tc.want) {
					t.Errorf("wrong result\ngot:  %q\nwant: %q", got, tc.want)
				}
			})
		}
	}
}

func TestApplyHuman_resourceCountStatePath(t *testing.T) {
	testCases := map[string]struct {
		added        int
		changed      int
		removed      int
		statePath    string
		wantContains bool
	}{
		"default state path": {
			added:        1,
			changed:      2,
			removed:      3,
			statePath:    "",
			wantContains: false,
		},
		"only removed": {
			added:        0,
			changed:      0,
			removed:      5,
			statePath:    "foo.tfstate",
			wantContains: false,
		},
		"added": {
			added:        5,
			changed:      0,
			removed:      0,
			statePath:    "foo.tfstate",
			wantContains: true,
		},
		"changed": {
			added:        0,
			changed:      5,
			removed:      0,
			statePath:    "foo.tfstate",
			wantContains: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			streams, done := terminal.StreamsForTesting(t)
			v := NewApply(arguments.ViewHuman, false, NewView(streams))
			hooks := v.Hooks()

			var count *countHook
			for _, hook := range hooks {
				if ch, ok := hook.(*countHook); ok {
					count = ch
				}
			}
			if count == nil {
				t.Fatalf("expected Hooks to include a countHook: %#v", hooks)
			}

			count.Added = tc.added
			count.Changed = tc.changed
			count.Removed = tc.removed

			v.ResourceCount(tc.statePath)

			got := done(t).Stdout()
			want := "State path: " + tc.statePath
			contains := strings.Contains(got, want)
			if contains && !tc.wantContains {
				t.Errorf("wrong result\ngot:  %q\nshould not contain: %q", got, want)
			} else if !contains && tc.wantContains {
				t.Errorf("wrong result\ngot:  %q\nshould contain: %q", got, want)
			}
		})
	}
}

// Basic test coverage of Outputs, since most of its functionality is tested
// elsewhere.
func TestApplyJSON_outputs(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewApply(arguments.ViewJSON, false, NewView(streams))

	v.Outputs(map[string]*states.OutputValue{
		"boop_count": {Value: cty.NumberIntVal(92)},
		"password":   {Value: cty.StringVal("horse-battery").Mark(marks.Sensitive), Sensitive: true},
	})

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "Outputs: 2",
			"@module":  "terraform.ui",
			"type":     "outputs",
			"outputs": map[string]interface{}{
				"boop_count": map[string]interface{}{
					"sensitive": false,
					"value":     float64(92),
					"type":      "number",
				},
				"password": map[string]interface{}{
					"sensitive": true,
					"type":      "string",
				},
			},
		},
	}
	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}
