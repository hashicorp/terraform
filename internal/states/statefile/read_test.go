// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package statefile

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

func TestSniffJSONStateMaybeCurrentV4(t *testing.T) {
	for name, src := range stateReadTestV4Sources() {
		t.Run(name, func(t *testing.T) {
			version, current, diags := sniffJSONStateMaybeCurrent([]byte(src))
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}
			if version != 4 {
				t.Fatalf("wrong format version: got %d, want 4", version)
			}
			// A successful probe must retain its decoded state, rather than
			// falling back and requiring another full decode.
			if current == nil {
				t.Fatal("successful v4 decode returned no reusable state")
			}
			if current.Serial != 42 || current.Lineage != "state-reader-test" {
				t.Errorf("wrong state metadata: serial=%d lineage=%q", current.Serial, current.Lineage)
			}
			if output := current.RootOutputs["large"]; !output.Sensitive || string(output.ValueRaw) != "9007199254740993" {
				t.Errorf("wrong decoded output: %#v", output)
			}
		})
	}
}

func TestReadVersionErrorsWithMalformedField(t *testing.T) {
	stringVersionError := tfdiags.Diagnostics{tfdiags.Sourceless(
		tfdiags.Error,
		"Unsupported state file format",
		"The version in the state file is string. A positive whole number is required.",
	)}
	missingVersionError := tfdiags.Diagnostics{tfdiags.Sourceless(
		tfdiags.Error,
		"Unsupported state file format",
		`The state file does not have a "version" attribute, which is required to identify the format version.`,
	)}
	tests := map[string]struct {
		src      string
		want     tfdiags.Diagnostics
		unusable bool
	}{
		"invalid version before field": {
			src:  `{"version":"4","serial":"bad"}`,
			want: stringVersionError,
		},
		"invalid version after field": {
			src:  `{"serial":"bad","version":"4"}`,
			want: stringVersionError,
		},
		"invalid version overwritten by valid duplicate": {
			src:  `{"version":"4","serial":"bad","version":4}`,
			want: stringVersionError,
		},
		"valid version overwritten by invalid duplicate": {
			src:  `{"version":4,"serial":"bad","version":"4"}`,
			want: stringVersionError,
		},
		"negative version after field": {
			src: `{"serial":"bad","version":-1}`,
			want: tfdiags.Diagnostics{tfdiags.Sourceless(
				tfdiags.Error,
				"Unsupported state file format",
				"The version in the state file is number -1. A positive whole number is required.",
			)},
		},
		"missing version": {
			src:  `{"serial":"bad"}`,
			want: missingVersionError,
		},
		"valid version overwritten by null": {
			src:  `{"version":4,"serial":"bad","version":null}`,
			want: missingVersionError,
		},
		"valid version overwritten by unsupported duplicate": {
			src: `{"version":4,"serial":"bad","version":5}`,
			want: tfdiags.Diagnostics{tfdiags.Sourceless(
				tfdiags.Error,
				"Unsupported state file format",
				// Only the executing Terraform version varies across releases.
				fmt.Sprintf("The state file uses format version 5, which is not supported by Terraform %s. This state file may have been created by a newer version of Terraform.", tfversion.SemVer.String()),
			)},
			unusable: true,
		},
		"unsupported version overwritten by current duplicate": {
			src: `{"version":5,"serial":"bad","version":4}`,
			want: tfdiags.Diagnostics{tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid state file format",
				`The state file field "serial" has invalid value string`,
			)},
			unusable: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assertStateReadError(t, test.src, test.want, test.unusable)
		})
	}
}

func TestReadSyntaxErrors(t *testing.T) {
	// regression test for Read errors
	tests := map[string]struct {
		src        string
		wantDetail string
	}{
		"truncated after version": {
			src:        `{"version":4,`,
			wantDetail: "The state file could not be parsed as JSON: syntax error at byte offset 13.",
		},
		"truncated after malformed field and version": {
			src:        `{"serial":"bad","version":4,`,
			wantDetail: "The state file could not be parsed as JSON: syntax error at byte offset 28.",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			want := tfdiags.Diagnostics{
				tfdiags.Sourceless(tfdiags.Error, "Unsupported state file format", test.wantDetail),
				tfdiags.Sourceless(
					tfdiags.Error,
					"Unsupported state file format",
					`The state file does not have a "version" attribute, which is required to identify the format version.`,
				),
			}
			assertStateReadError(t, test.src, want, false)
		})
	}
}

func assertStateReadError(t *testing.T, src string, want tfdiags.Diagnostics, wantUnusable bool) {
	t.Helper()
	file, err := Read(strings.NewReader(src))
	if err == nil {
		t.Fatal("accepted invalid state")
	}
	if file != nil {
		t.Error("returned partially decoded state on error")
	}
	var unusable *ErrUnusableState
	if got := errors.As(err, &unusable); got != wantUnusable {
		t.Errorf("wrong unusable-state classification: got %t, want %t: %s", got, wantUnusable, err)
	}
	if got, want := err.Error(), want.Err().Error(); got != want {
		t.Errorf("wrong error text\ngot:  %s\nwant: %s", got, want)
	}
	// Compare all diagnostics, not substrings: severity, summary, detail,
	// source information, and count must remain unchanged from main.
	tfdiags.AssertDiagnosticsMatch(t, (tfdiags.Diagnostics{}).Append(err), want)
}

func TestReadV4VersionOrder(t *testing.T) {
	sources := stateReadTestV4Sources()
	// The existing v4 reader defines the state preparation behavior that the
	// optimized public reader must preserve, regardless of member order.
	want, diags := readStateV4([]byte(sources["version first"]))
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}
	for name, src := range sources {
		t.Run(name, func(t *testing.T) {
			got, err := Read(strings.NewReader(src))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Error("public reader differs from the existing v4 reader")
			}
			output := got.State.RootOutputValues["large"]
			if output == nil {
				t.Fatal("missing output")
			}
			if !output.Sensitive || !output.Value.RawEquals(cty.MustParseNumberVal("9007199254740993")) {
				t.Errorf("output sensitivity or number precision changed: %#v", output)
			}
		})
	}
}

func stateReadTestV4Sources() map[string]string {
	const body = `
		"terraform_version": "1.15.0",
		"serial": 42,
		"lineage": "state-reader-test",
		"outputs": {
			"large": {"type": "number", "value": 9007199254740993, "sensitive": true}
		},
		"resources": [{
			"mode": "managed",
			"type": "test_resource",
			"name": "example",
			"provider": "provider[\"registry.terraform.io/hashicorp/test\"]",
			"instances": [{
				"schema_version": 1,
				"attributes": {"secret": "secret-value"},
				"sensitive_attributes": [[{"type": "get_attr", "value": "secret"}]]
			}]
		}]
	`
	return map[string]string{
		"version first": `{"version":4,` + body + `}`,
		"version last":  `{` + body + `,"version":4}`,
	}
}
