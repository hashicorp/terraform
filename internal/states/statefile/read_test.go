// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package statefile

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestReadStateV4(t *testing.T) {
	src := []byte(`{
  "version": 4,
  "terraform_version": "1.15.0",
  "serial": 42,
  "lineage": "test-lineage",
  "outputs": {
    "large": {"value": 9007199254740993, "type": "number", "sensitive": true}
  },
  "resources": [{
    "mode": "managed",
    "type": "test_resource",
    "name": "example",
    "provider": "provider[\"registry.terraform.io/hashicorp/test\"]",
    "instances": [{
      "index_key": 0,
      "schema_version": 1,
      "attributes": {"id": "example", "secret": "hidden", "nested": [{"secret": "nested-hidden"}]},
      "sensitive_attributes": [
        [{"type": "get_attr", "value": "secret"}],
        [{"type": "get_attr", "value": "nested"}, {"type": "index", "value": {"value": 0, "type": "number"}}, {"type": "get_attr", "value": "secret"}]
      ],
      "identity_schema_version": 1,
      "identity": {"id": "example"},
      "private": "cHJpdmF0ZQ==",
      "create_before_destroy": true
    }]
  }],
  "check_results": []
}`)
	var compact bytes.Buffer
	if err := json.Compact(&compact, src); err != nil {
		t.Fatal(err)
	}
	versionLast := bytes.Replace(src, []byte("  \"version\": 4,\n"), nil, 1)
	versionLast = append(bytes.Clone(versionLast[:len(versionLast)-1]), []byte(",\n  \"version\": 4\n}")...)
	duplicate := append(bytes.Clone(src[:len(src)-1]), []byte(",\n  \"version\": 4\n}")...)
	var members map[string]json.RawMessage
	if err := json.Unmarshal(src, &members); err != nil {
		t.Fatal(err)
	}
	rewritten, err := json.MarshalIndent(members, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	for name, input := range map[string][]byte{
		"version first":       src,
		"compact":             compact.Bytes(),
		"version last":        versionLast,
		"rewritten JSON":      rewritten,
		"case insensitive":    bytes.Replace(src, []byte(`"version"`), []byte(`"VERSION"`), 1),
		"escaped name":        bytes.Replace(src, []byte(`"version"`), []byte(`"\u0076ersion"`), 1),
		"duplicate version":   duplicate,
		"duplicate overrides": bytes.Replace(duplicate, []byte(`"version": 4`), []byte(`"version": 5`), 1),
	} {
		t.Run(name, func(t *testing.T) {
			want, diags := readStateV4(input)
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}
			got, err := Read(bytes.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Error("decoded state differs from the v4 reader")
			}
			addr := addrs.Resource{Mode: addrs.ManagedResourceMode, Type: "test_resource", Name: "example"}.Instance(addrs.IntKey(0))
			obj := got.State.RootModule().ResourceInstance(addr).Current
			paths := []cty.Path{cty.GetAttrPath("secret"), cty.GetAttrPath("nested").IndexInt(0).GetAttr("secret")}
			if len(obj.AttrSensitivePaths) != len(paths) {
				t.Fatalf("unexpected sensitive paths: %#v", obj.AttrSensitivePaths)
			}
			for i, path := range paths {
				if !obj.AttrSensitivePaths[i].Equals(path) {
					t.Errorf("unexpected sensitive path: %#v", obj.AttrSensitivePaths[i])
				}
			}
			if output := got.State.RootOutputValues["large"]; !output.Sensitive || !output.Value.RawEquals(cty.NumberIntVal(9007199254740993)) {
				t.Error("output sensitivity or number precision changed")
			}
		})
	}
}

func TestReadStateErrors(t *testing.T) {
	missingVersion := tfdiags.Description{
		Summary: unsupportedFormat,
		Detail:  "The state file does not have a \"version\" attribute, which is required to identify the format version.",
	}
	invalidVersion := tfdiags.Description{
		Summary: unsupportedFormat,
		Detail:  "The version in the state file is string. A positive whole number is required.",
	}
	for name, test := range map[string]struct {
		src      string
		unusable bool
		want     []tfdiags.Description
	}{
		"missing version": {src: `{}`, want: []tfdiags.Description{missingVersion}},
		"truncated v4": {
			src: `{"version":4`,
			want: []tfdiags.Description{
				{Summary: unsupportedFormat, Detail: "The state file could not be parsed as JSON: syntax error at byte offset 12."},
				missingVersion,
			},
		},
		"trailing JSON": {
			src: `{"version":4} {}`,
			want: []tfdiags.Description{
				{Summary: unsupportedFormat, Detail: "The state file could not be parsed as JSON: syntax error at byte offset 15."},
				missingVersion,
			},
		},
		"serial type": {
			src: `{"version":4,"serial":-1}`, unusable: true,
			want: []tfdiags.Description{{Summary: invalidFormat, Detail: "The state file does not conform to the expected JSON data structure."}},
		},
		"resource type": {
			src: `{"version":4,"resources":{}}`, unusable: true,
			want: []tfdiags.Description{{Summary: invalidFormat, Detail: "The state file does not conform to the expected JSON data structure."}},
		},
		"preparation error": {
			src: `{"version":4,"resources":[{"mode":"invalid","type":"test_resource","name":"example"}]}`, unusable: true,
			want: []tfdiags.Description{{Summary: "Invalid resource mode in state", Detail: "State contains a resource with mode \"invalid\" (\"test_resource\" \"example\") which is not supported."}},
		},
		"version error after field error": {
			src:  `{"version":4,"serial":-1,"version":"bad"}`,
			want: []tfdiags.Description{invalidVersion},
		},
		"null version after field error": {
			src:  `{"version":4,"serial":-1,"version":null}`,
			want: []tfdiags.Description{missingVersion},
		},
		"invalid version before valid duplicate": {
			src:  `{"version":"bad","version":4}`,
			want: []tfdiags.Description{invalidVersion},
		},
		"syntax error after field error": {
			src: `{"version":4,"serial":-1`,
			want: []tfdiags.Description{
				{Summary: unsupportedFormat, Detail: "The state file could not be parsed as JSON: syntax error at byte offset 24."},
				missingVersion,
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := Read(bytes.NewBufferString(test.src))
			if err == nil || got != nil {
				t.Fatalf("expected an error and no state, got %v, %v", got, err)
			}
			var unusable *ErrUnusableState
			if errors.As(err, &unusable) != test.unusable {
				t.Errorf("unexpected error classification: %T", err)
			}
			var descriptions []tfdiags.Description
			for _, diag := range (tfdiags.Diagnostics{}).Append(err) {
				if diag.Severity() != tfdiags.Error {
					t.Errorf("unexpected diagnostic severity: %s", diag.Severity())
				}
				descriptions = append(descriptions, diag.Description())
			}
			if !reflect.DeepEqual(descriptions, test.want) {
				t.Errorf("unexpected diagnostics:\n got: %#v\nwant: %#v", descriptions, test.want)
			}
		})
	}
}

func TestDecodeJSONState(t *testing.T) {
	for name, test := range map[string]struct {
		src     string
		version uint64
		decoded bool
		errors  bool
	}{
		"v4":                 {src: `{"version":4}`, version: 4, decoded: true},
		"version last":       {src: `{"serial":42,"version":4}`, version: 4, decoded: true},
		"invalid v4 field":   {src: `{"serial":-1,"version":4}`, version: 4},
		"legacy after error": {src: `{"serial":-1,"version":3}`, version: 3},
		"invalid version":    {src: `{"version":"bad"}`, errors: true},
		"missing version":    {src: `{}`, errors: true},
		"invalid syntax":     {src: `{"version":4`, errors: true},
	} {
		t.Run(name, func(t *testing.T) {
			version, current, diags := decodeJSONState([]byte(test.src))
			if version != test.version || (current != nil) != test.decoded || diags.HasErrors() != test.errors {
				t.Fatalf("got version=%d decoded=%t errors=%t; want version=%d decoded=%t errors=%t",
					version, current != nil, diags.HasErrors(), test.version, test.decoded, test.errors)
			}
		})
	}
}

func TestReadEmpty(t *testing.T) {
	for name, reader := range map[string]io.Reader{
		"empty":     bytes.NewReader(nil),
		"typed nil": (*os.File)(nil),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Read(reader)
			if !errors.Is(err, ErrNoState) {
				t.Fatalf("expected ErrNoState, got %v", err)
			}
		})
	}
}
