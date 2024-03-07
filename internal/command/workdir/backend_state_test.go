// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
)

func TestParseBackendStateFile(t *testing.T) {
	tests := map[string]struct {
		Input   string
		Want    *BackendStateFile
		WantErr string
	}{
		"empty": {
			Input:   ``,
			WantErr: `invalid syntax: unexpected end of JSON input`,
		},
		"empty but valid JSON syntax": {
			Input:   `{}`,
			WantErr: `invalid syntax: no format version number`,
		},
		"older version": {
			Input: `{
				"version": 2,
				"terraform_version": "0.3.0"
			}`,
			WantErr: `unsupported backend state version 2; you may need to use Terraform CLI v0.3.0 to work in this directory`,
		},
		"newer version": {
			Input: `{
				"version": 4,
				"terraform_version": "54.23.9"
			}`,
			WantErr: `unsupported backend state version 4; you may need to use Terraform CLI v54.23.9 to work in this directory`,
		},
		"legacy remote state is active": {
			Input: `{
				"version": 3,
				"terraform_version": "0.8.0",
				"remote": {
					"anything": "goes"
				}
			}`,
			WantErr: `this working directory uses legacy remote state and so must first be upgraded using Terraform v0.9`,
		},
		"active backend": {
			Input: `{
				"version": 3,
				"terraform_version": "0.8.0",
				"backend": {
					"type": "treasure_chest_buried_on_a_remote_island",
					"config": {}
				}
			}`,
			Want: &BackendStateFile{
				Version:   3,
				TFVersion: "0.8.0",
				Backend: &BackendState{
					Type:      "treasure_chest_buried_on_a_remote_island",
					ConfigRaw: json.RawMessage("{}"),
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseBackendStateFile([]byte(test.Input))

			if test.WantErr != "" {
				if err == nil {
					t.Fatalf("unexpected success\nwant error: %s", test.WantErr)
				}
				if got, want := err.Error(), test.WantErr; got != want {
					t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(test.Want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

func ParseBackendStateConfig(t *testing.T) {
	// This test only really covers the happy path because Config/SetConfig is
	// largely just a thin wrapper around configschema's "ImpliedType" and
	// cty's json unmarshal/marshal and both of those are well-tested elsewhere.

	s := &BackendState{
		Type: "whatever",
		ConfigRaw: []byte(`{
			"foo": "bar"
		}`),
	}

	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}
	got, err := s.Config(schema)
	want := cty.ObjectVal(map[string]cty.Value{
		"foo": cty.StringVal("bar"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}

	err = s.SetConfig(cty.ObjectVal(map[string]cty.Value{
		"foo": cty.StringVal("baz"),
	}), schema)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	gotRaw := s.ConfigRaw
	wantRaw := []byte(`{"foo":"baz"}`)
	if !bytes.Equal(wantRaw, gotRaw) {
		t.Errorf("wrong raw config after encode\ngot:  %s\nwant: %s", gotRaw, wantRaw)
	}
}
