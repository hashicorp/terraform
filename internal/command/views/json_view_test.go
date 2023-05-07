// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package views

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

// Calling NewJSONView should also always output a version message, which is a
// convenient way to test that NewJSONView works.
func TestNewJSONView(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	NewJSONView(NewView(streams))

	version := tfversion.String()
	want := []map[string]interface{}{
		{
			"@level":    "info",
			"@message":  fmt.Sprintf("Terraform %s", version),
			"@module":   "terraform.ui",
			"type":      "version",
			"terraform": version,
			"ui":        JSON_UI_VERSION,
		},
	}

	testJSONViewOutputEqualsFull(t, done(t).Stdout(), want)
}

func TestJSONView_Log(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	jv := NewJSONView(NewView(streams))

	jv.Log("hello, world")

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "hello, world",
			"@module":  "terraform.ui",
			"type":     "log",
		},
	}
	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

// This test covers only the basics of JSON diagnostic rendering, as more
// complex diagnostics are tested elsewhere.
func TestJSONView_Diagnostics(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	jv := NewJSONView(NewView(streams))

	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Warning,
		`Improper use of "less"`,
		`You probably mean "10 buckets or fewer"`,
	))
	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Unusually stripey cat detected",
		"Are you sure this random_pet isn't a cheetah?",
	))

	jv.Diagnostics(diags)

	want := []map[string]interface{}{
		{
			"@level":   "warn",
			"@message": `Warning: Improper use of "less"`,
			"@module":  "terraform.ui",
			"type":     "diagnostic",
			"diagnostic": map[string]interface{}{
				"severity": "warning",
				"summary":  `Improper use of "less"`,
				"detail":   `You probably mean "10 buckets or fewer"`,
			},
		},
		{
			"@level":   "error",
			"@message": "Error: Unusually stripey cat detected",
			"@module":  "terraform.ui",
			"type":     "diagnostic",
			"diagnostic": map[string]interface{}{
				"severity": "error",
				"summary":  "Unusually stripey cat detected",
				"detail":   "Are you sure this random_pet isn't a cheetah?",
			},
		},
	}
	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestJSONView_PlannedChange(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	jv := NewJSONView(NewView(streams))

	foo, diags := addrs.ParseModuleInstanceStr("module.foo")
	if len(diags) > 0 {
		t.Fatal(diags.Err())
	}
	managed := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_instance", Name: "bar"}
	cs := &plans.ResourceInstanceChangeSrc{
		Addr:        managed.Instance(addrs.StringKey("boop")).Absolute(foo),
		PrevRunAddr: managed.Instance(addrs.StringKey("boop")).Absolute(foo),
		ChangeSrc: plans.ChangeSrc{
			Action: plans.Create,
		},
	}
	jv.PlannedChange(viewsjson.NewResourceInstanceChange(cs))

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": `module.foo.test_instance.bar["boop"]: Plan to create`,
			"@module":  "terraform.ui",
			"type":     "planned_change",
			"change": map[string]interface{}{
				"action": "create",
				"resource": map[string]interface{}{
					"addr":             `module.foo.test_instance.bar["boop"]`,
					"implied_provider": "test",
					"module":           "module.foo",
					"resource":         `test_instance.bar["boop"]`,
					"resource_key":     "boop",
					"resource_name":    "bar",
					"resource_type":    "test_instance",
				},
			},
		},
	}
	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestJSONView_ResourceDrift(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	jv := NewJSONView(NewView(streams))

	foo, diags := addrs.ParseModuleInstanceStr("module.foo")
	if len(diags) > 0 {
		t.Fatal(diags.Err())
	}
	managed := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_instance", Name: "bar"}
	cs := &plans.ResourceInstanceChangeSrc{
		Addr:        managed.Instance(addrs.StringKey("boop")).Absolute(foo),
		PrevRunAddr: managed.Instance(addrs.StringKey("boop")).Absolute(foo),
		ChangeSrc: plans.ChangeSrc{
			Action: plans.Update,
		},
	}
	jv.ResourceDrift(viewsjson.NewResourceInstanceChange(cs))

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": `module.foo.test_instance.bar["boop"]: Drift detected (update)`,
			"@module":  "terraform.ui",
			"type":     "resource_drift",
			"change": map[string]interface{}{
				"action": "update",
				"resource": map[string]interface{}{
					"addr":             `module.foo.test_instance.bar["boop"]`,
					"implied_provider": "test",
					"module":           "module.foo",
					"resource":         `test_instance.bar["boop"]`,
					"resource_key":     "boop",
					"resource_name":    "bar",
					"resource_type":    "test_instance",
				},
			},
		},
	}
	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestJSONView_ChangeSummary(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	jv := NewJSONView(NewView(streams))

	jv.ChangeSummary(&viewsjson.ChangeSummary{
		Add:       1,
		Change:    2,
		Remove:    3,
		Operation: viewsjson.OperationApplied,
	})

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "Apply complete! Resources: 1 added, 2 changed, 3 destroyed.",
			"@module":  "terraform.ui",
			"type":     "change_summary",
			"changes": map[string]interface{}{
				"add":       float64(1),
				"import":    float64(0),
				"change":    float64(2),
				"remove":    float64(3),
				"operation": "apply",
			},
		},
	}
	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestJSONView_ChangeSummaryWithImport(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	jv := NewJSONView(NewView(streams))

	jv.ChangeSummary(&viewsjson.ChangeSummary{
		Add:       1,
		Change:    2,
		Remove:    3,
		Import:    1,
		Operation: viewsjson.OperationApplied,
	})

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "Apply complete! Resources: 1 imported, 1 added, 2 changed, 3 destroyed.",
			"@module":  "terraform.ui",
			"type":     "change_summary",
			"changes": map[string]interface{}{
				"add":       float64(1),
				"change":    float64(2),
				"remove":    float64(3),
				"import":    float64(1),
				"operation": "apply",
			},
		},
	}
	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestJSONView_Hook(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	jv := NewJSONView(NewView(streams))

	foo, diags := addrs.ParseModuleInstanceStr("module.foo")
	if len(diags) > 0 {
		t.Fatal(diags.Err())
	}
	managed := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_instance", Name: "bar"}
	addr := managed.Instance(addrs.StringKey("boop")).Absolute(foo)
	hook := viewsjson.NewApplyComplete(addr, plans.Create, "id", "boop-beep", 34*time.Second)

	jv.Hook(hook)

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": `module.foo.test_instance.bar["boop"]: Creation complete after 34s [id=boop-beep]`,
			"@module":  "terraform.ui",
			"type":     "apply_complete",
			"hook": map[string]interface{}{
				"resource": map[string]interface{}{
					"addr":             `module.foo.test_instance.bar["boop"]`,
					"implied_provider": "test",
					"module":           "module.foo",
					"resource":         `test_instance.bar["boop"]`,
					"resource_key":     "boop",
					"resource_name":    "bar",
					"resource_type":    "test_instance",
				},
				"action":          "create",
				"id_key":          "id",
				"id_value":        "boop-beep",
				"elapsed_seconds": float64(34),
			},
		},
	}
	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestJSONView_Outputs(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	jv := NewJSONView(NewView(streams))

	jv.Outputs(viewsjson.Outputs{
		"boop_count": {
			Sensitive: false,
			Value:     json.RawMessage(`92`),
			Type:      json.RawMessage(`"number"`),
		},
		"password": {
			Sensitive: true,
			Value:     json.RawMessage(`"horse-battery"`),
			Type:      json.RawMessage(`"string"`),
		},
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
					"value":     "horse-battery",
					"type":      "string",
				},
			},
		},
	}
	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

// This helper function tests a possibly multi-line JSONView output string
// against a slice of structs representing the desired log messages. It
// verifies that the output of JSONView is in JSON log format, one message per
// line.
func testJSONViewOutputEqualsFull(t *testing.T, output string, want []map[string]interface{}) {
	t.Helper()

	// Remove final trailing newline
	output = strings.TrimSuffix(output, "\n")

	// Split log into lines, each of which should be a JSON log message
	gotLines := strings.Split(output, "\n")

	if len(gotLines) != len(want) {
		t.Errorf("unexpected number of messages. got %d, want %d", len(gotLines), len(want))
	}

	// Unmarshal each line and compare to the expected value
	for i := range gotLines {
		var gotStruct map[string]interface{}
		if i >= len(want) {
			t.Error("reached end of want messages too soon")
			break
		}
		wantStruct := want[i]

		if err := json.Unmarshal([]byte(gotLines[i]), &gotStruct); err != nil {
			t.Fatal(err)
		}

		if timestamp, ok := gotStruct["@timestamp"]; !ok {
			t.Errorf("message has no timestamp: %#v", gotStruct)
		} else {
			// Remove the timestamp value from the struct to allow comparison
			delete(gotStruct, "@timestamp")

			// Verify the timestamp format
			if _, err := time.Parse("2006-01-02T15:04:05.000000Z07:00", timestamp.(string)); err != nil {
				t.Errorf("error parsing timestamp on line %d: %s", i, err)
			}
		}

		if !cmp.Equal(wantStruct, gotStruct) {
			t.Errorf("unexpected output on line %d:\n%s", i, cmp.Diff(wantStruct, gotStruct))
		}
	}
}

// testJSONViewOutputEquals skips the first line of output, since it ought to
// be a version message that we don't care about for most of our tests.
func testJSONViewOutputEquals(t *testing.T, output string, want []map[string]interface{}) {
	t.Helper()

	// Remove up to the first newline
	index := strings.Index(output, "\n")
	if index >= 0 {
		output = output[index+1:]
	}
	testJSONViewOutputEqualsFull(t, output, want)
}
