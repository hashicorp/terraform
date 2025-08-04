// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
				Backend: &BackendConfigState{
					Type:      "treasure_chest_buried_on_a_remote_island",
					ConfigRaw: json.RawMessage("{}"),
				},
			},
		},
		"active state_store": {
			Input: `{
				"version": 3,
				"terraform_version": "9.9.9",
				"state_store": {
					"type": "foobar_baz",
					"config": {
						"bucket": "my-bucket",
						"region": "saturn"
					},
					"provider": {
						"version": "1.2.3",
						"source": "registry.terraform.io/my-org/foobar",
						"config": {
							"credentials": "./creds.json"
						}
					}
				}
			}`,
			Want: &BackendStateFile{
				Version:   3,
				TFVersion: "9.9.9",
				StateStore: &StateStoreConfigState{
					Type: "foobar_baz",
					// Watch out - the number of tabs in the last argument here are load-bearing
					Provider: getTestProviderState(t, "1.2.3", "registry.terraform.io", "my-org", "foobar", `{
							"credentials": "./creds.json"
						}`),
					ConfigRaw: json.RawMessage(`{
						"bucket": "my-bucket",
						"region": "saturn"
					}`),
				},
			},
		},
		"detection of malformed state: conflicting 'backend' and 'state_store' sections": {
			Input: `{
				"version": 3,
				"terraform_version": "9.9.9",
				"backend": {
					"type": "treasure_chest_buried_on_a_remote_island",
					"config": {}
				},
				"state_store": {
					"type": "foobar_baz",
					"config": {
						"provider": "foobar",
						"bucket": "my-bucket"
					},
					"provider": {
						"version": "1.2.3",
						"source": "registry.terraform.io/my-org/foobar"
					}
				}
			}`,
			WantErr: `encountered a malformed backend state file that contains state for both a 'backend' and a 'state_store' block`,
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

func TestEncodeBackendStateFile(t *testing.T) {

	tests := map[string]struct {
		Input   *BackendStateFile
		Want    []byte
		WantErr string
	}{
		"encoding a backend state file when state_store is in use": {
			Input: &BackendStateFile{
				StateStore: &StateStoreConfigState{
					Type:      "foobar_baz",
					Provider:  getTestProviderState(t, "1.2.3", "registry.terraform.io", "my-org", "foobar", `{"foo": "bar"}`),
					ConfigRaw: json.RawMessage([]byte(`{"foo":"bar"}`)),
					Hash:      123,
				},
			},
			Want: []byte("{\n  \"version\": 3,\n  \"terraform_version\": \"1.13.0\",\n  \"state_store\": {\n    \"type\": \"foobar_baz\",\n    \"provider\": {\n      \"version\": \"1.2.3\",\n      \"source\": \"registry.terraform.io/my-org/foobar\",\n      \"config\": {\n        \"foo\": \"bar\"\n      }\n    },\n    \"config\": {\n      \"foo\": \"bar\"\n    },\n    \"hash\": 123\n  }\n}"),
		},
		"it returns an error when neither backend nor state_store config state are present": {
			Input: &BackendStateFile{},
			Want:  []byte("{\n  \"version\": 3,\n  \"terraform_version\": \"1.13.0\"\n}"),
		},
		"it returns an error when the provider source's hostname is missing": {
			Input: &BackendStateFile{
				StateStore: &StateStoreConfigState{
					Type:      "foobar_baz",
					Provider:  getTestProviderState(t, "1.2.3", "", "my-org", "foobar", ""),
					ConfigRaw: json.RawMessage([]byte(`{"foo":"bar"}`)),
					Hash:      123,
				},
			},
			WantErr: `state store is not valid: Unknown hostname: Expected hostname in the provider address to be set`,
		},
		"it returns an error when the provider source's hostname and namespace are missing ": {
			Input: &BackendStateFile{
				StateStore: &StateStoreConfigState{
					Type:      "foobar_baz",
					Provider:  getTestProviderState(t, "1.2.3", "", "", "foobar", ""),
					ConfigRaw: json.RawMessage([]byte(`{"foo":"bar"}`)),
					Hash:      123,
				},
			},
			WantErr: `state store is not valid: Unknown hostname: Expected hostname in the provider address to be set`,
		},
		"it returns an error when the provider source is completely missing ": {
			Input: &BackendStateFile{
				StateStore: &StateStoreConfigState{
					Type:      "foobar_baz",
					Provider:  getTestProviderState(t, "1.2.3", "", "", "", ""),
					ConfigRaw: json.RawMessage([]byte(`{"foo":"bar"}`)),
					Hash:      123,
				},
			},
			WantErr: `state store is not valid: Empty provider address: Expected address composed of hostname, provider namespace and name`,
		},
		"it returns an error when both backend and state_store config state are present": {
			Input: &BackendStateFile{
				Backend: &BackendConfigState{
					Type:      "foobar",
					ConfigRaw: json.RawMessage([]byte(`{"foo":"bar"}`)),
					Hash:      123,
				},
				StateStore: &StateStoreConfigState{
					Type:      "foobar_baz",
					Provider:  getTestProviderState(t, "1.2.3", "registry.terraform.io", "my-org", "foobar", ""),
					ConfigRaw: json.RawMessage([]byte(`{"foo":"bar"}`)),
					Hash:      123,
				},
			},
			WantErr: `attempted to encode a malformed backend state file; it contains state for both a 'backend' and a 'state_store' block`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := EncodeBackendStateFile(test.Input)

			if test.WantErr != "" {
				if err == nil {
					t.Fatalf("unexpected success\nwant error: %s", test.WantErr)
				}
				if !strings.Contains(err.Error(), test.WantErr) {
					t.Errorf("wrong error\ngot:  %s\nwant: %s", err.Error(), test.WantErr)
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

func TestBackendStateFile_DeepCopy(t *testing.T) {

	tests := map[string]struct {
		file *BackendStateFile
	}{
		"Deep copy preserves state_store data": {
			file: &BackendStateFile{
				StateStore: &StateStoreConfigState{
					Type:      "foo_bar",
					Provider:  getTestProviderState(t, "1.2.3", "A", "B", "C", ""),
					ConfigRaw: json.RawMessage([]byte(`{"foo":"bar"}`)),
					Hash:      123,
				},
			},
		},
		"Deep copy preserves backend data": {
			file: &BackendStateFile{
				Backend: &BackendConfigState{
					Type:      "foobar",
					ConfigRaw: json.RawMessage([]byte(`{"foo":"bar"}`)),
					Hash:      123,
				},
			},
		},
		"Deep copy preserves version and Terraform version data": {
			file: &BackendStateFile{
				Version:   3,
				TFVersion: "9.9.9",
			},
		},
	}

	for tn, tc := range tests {
		t.Run(tn, func(t *testing.T) {
			copy := tc.file.DeepCopy()

			if !reflect.DeepEqual(copy, tc.file) {
				t.Fatalf("unexpected difference in backend state data:\n got %#v, want %#v", copy, tc.file)
			}
		})
	}
}
