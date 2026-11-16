// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package structured_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/command/jsonformat/computed"
	"github.com/hashicorp/terraform/internal/command/jsonformat/differ"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured"
	"github.com/hashicorp/terraform/internal/command/jsonformat/structured/attribute_path"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/plans"
)

func TestFromJsonResource(t *testing.T) {
	tests := map[string]struct {
		resource  jsonstate.Resource
		values    map[string]interface{}
		sensitive interface{}
	}{
		"nil values": {
			values: map[string]interface{}{},
		},
		"empty values": {
			resource: jsonstate.Resource{
				AttributeValues: jsonstate.AttributeValues{},
				SensitiveValues: json.RawMessage(`{}`),
			},
			values:    map[string]interface{}{},
			sensitive: map[string]interface{}{},
		},
		"nested values": {
			resource: jsonstate.Resource{
				AttributeValues: jsonstate.AttributeValues{
					"value": json.RawMessage(`{"large":9007199254740993,"decimal":1.234567890123456789,"list":[true,"",null]}`),
					"null":  json.RawMessage(`null`),
				},
				SensitiveValues: json.RawMessage(`{"value":{"list":[false,true,false]}}`),
			},
			values: map[string]interface{}{
				"value": map[string]interface{}{
					"large":   json.Number("9007199254740993"),
					"decimal": json.Number("1.234567890123456789"),
					"list":    []interface{}{true, "", nil},
				},
				"null": nil,
			},
			sensitive: map[string]interface{}{
				"value": map[string]interface{}{"list": []interface{}{false, true, false}},
			},
		},
		"sensitive resource": {
			resource: jsonstate.Resource{
				AttributeValues: jsonstate.AttributeValues{"value": json.RawMessage(`"secret"`)},
				SensitiveValues: json.RawMessage(`true`),
			},
			values:    map[string]interface{}{"value": "secret"},
			sensitive: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := structured.FromJsonResource(test.resource)
			want := structured.Change{
				Before:             test.values,
				After:              test.values,
				Unknown:            false,
				BeforeSensitive:    test.sensitive,
				AfterSensitive:     test.sensitive,
				ReplacePaths:       attribute_path.Empty(false),
				RelevantAttributes: attribute_path.AlwaysMatcher(),
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("unexpected change (-want +got):\n%s", diff)
			}
			if got.CalculateAction() != plans.NoOp {
				t.Errorf("expected a no-op, got %s", got.CalculateAction())
			}
		})
	}
}

func TestFromJsonResourceRendering(t *testing.T) {
	resource := jsonstate.Resource{
		AttributeValues: jsonstate.AttributeValues{
			"large":   json.RawMessage(`9007199254740993`),
			"decimal": json.RawMessage(`1.234567890123456789`),
			"empty":   json.RawMessage(`""`),
			"null":    json.RawMessage(`null`),
			"items":   json.RawMessage(`[{"name":"visible","secret":"do-not-display"}]`),
		},
		SensitiveValues: json.RawMessage(`{"items":[{"secret":true}]}`),
	}
	block := &jsonprovider.Block{Attributes: map[string]*jsonprovider.Attribute{
		"large":   {AttributeType: json.RawMessage(`"number"`)},
		"decimal": {AttributeType: json.RawMessage(`"number"`)},
		"empty":   {AttributeType: json.RawMessage(`"string"`)},
		"null":    {AttributeType: json.RawMessage(`"string"`)},
		"items":   {AttributeType: json.RawMessage(`["list",["object",{"name":"string","secret":"string"}]]`)},
	}}
	change := structured.FromJsonResource(resource)
	wantValues, err := json.Marshal([]interface{}{change.Before, change.After, change.BeforeSensitive, change.AfterSensitive})
	if err != nil {
		t.Fatal(err)
	}
	opts := computed.NewRenderHumanOpts(&colorstring.Colorize{Colors: colorstring.DefaultColors, Disable: true})
	opts.ShowUnchangedChildren = true
	var previous string
	for i := 0; i < 2; i++ {
		diff := differ.ComputeDiffForBlock(change, block)
		if diff.Action != plans.NoOp {
			t.Fatalf("expected a no-op, got %s", diff.Action)
		}
		got := diff.RenderHuman(0, opts)
		for _, want := range []string{"9007199254740993", "1.234567890123456789", "visible", "(sensitive value)"} {
			if !strings.Contains(got, want) {
				t.Errorf("rendered output is missing %q:\n%s", want, got)
			}
		}
		if strings.Contains(got, "do-not-display") {
			t.Error("rendered output exposes a sensitive value")
		}
		if i > 0 && got != previous {
			t.Error("rendering the same change twice produced different output")
		}
		previous = got
		gotValues, err := json.Marshal([]interface{}{change.Before, change.After, change.BeforeSensitive, change.AfterSensitive})
		if err != nil {
			t.Fatal(err)
		}
		if string(gotValues) != string(wantValues) {
			t.Error("rendering mutated the decoded values or sensitivity masks")
		}
	}
}
