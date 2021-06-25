package views

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/zclconf/go-cty/cty"
)

// Ensure that the correct view type and in-automation settings propagate to the
// Operation view.
func TestRefreshHuman_operation(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	defer done(t)
	v := NewRefresh(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true)).Operation()
	if hv, ok := v.(*OperationHuman); !ok {
		t.Fatalf("unexpected return type %t", v)
	} else if hv.inAutomation != true {
		t.Fatalf("unexpected inAutomation value on Operation view")
	}
}

// Verify that Hooks includes a UI hook
func TestRefreshHuman_hooks(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	defer done(t)
	v := NewRefresh(arguments.ViewHuman, NewView(streams).SetRunningInAutomation(true))
	hooks := v.Hooks()

	var uiHook *UiHook
	for _, hook := range hooks {
		if ch, ok := hook.(*UiHook); ok {
			uiHook = ch
		}
	}
	if uiHook == nil {
		t.Fatalf("expected Hooks to include a UiHook: %#v", hooks)
	}
}

// Basic test coverage of Outputs, since most of its functionality is tested
// elsewhere.
func TestRefreshHuman_outputs(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewRefresh(arguments.ViewHuman, NewView(streams))

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
func TestRefreshHuman_outputsEmpty(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewRefresh(arguments.ViewHuman, NewView(streams))

	v.Outputs(map[string]*states.OutputValue{})

	got := done(t).Stdout()
	if got != "" {
		t.Errorf("output should be empty, but got: %q", got)
	}
}

// Basic test coverage of Outputs, since most of its functionality is tested
// elsewhere.
func TestRefreshJSON_outputs(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	v := NewRefresh(arguments.ViewJSON, NewView(streams))

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
					"value":     "horse-battery",
					"type":      "string",
				},
			},
		},
	}
	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}
