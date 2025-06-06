// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestParseStateStoreConfigState_Config_SetConfig(t *testing.T) {
	// This test only really covers the happy path because Config/SetConfig is
	// largely just a thin wrapper around configschema's "ImpliedType" and
	// cty's json unmarshal/marshal and both of those are well-tested elsewhere.

	s := &StateStoreConfigState{
		Type: "whatever",
		ConfigRaw: []byte(`{
			"provider": "foobar",
			"foo": "bar"
		}`),
		Provider: getTestProviderState(t, "1.2.3", "registry.terraform.io", "my-org", "foobar"),
		Hash:     12345,
	}

	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"provider": {
				Type:     cty.String,
				Required: true,
			},
			"foo": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	// Test Config method
	got, err := s.Config(schema)
	want := cty.ObjectVal(map[string]cty.Value{
		"provider": cty.StringVal("foobar"),
		"foo":      cty.StringVal("bar"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}

	// Test SetConfig method
	err = s.SetConfig(cty.ObjectVal(map[string]cty.Value{
		"provider": cty.StringVal("foobar"),
		"foo":      cty.StringVal("baz"),
	}), schema)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	gotRaw := s.ConfigRaw
	wantRaw := []byte(`{"foo":"baz","provider":"foobar"}`)
	if !bytes.Equal(wantRaw, gotRaw) {
		t.Errorf("wrong raw config after encode\ngot:  %s\nwant: %s", gotRaw, wantRaw)
	}
}

func TestParseStateStoreConfigState_Empty(t *testing.T) {
	// Populated StateStoreConfigState isn't empty
	s := &StateStoreConfigState{
		Type: "whatever",
		ConfigRaw: []byte(`{
			"provider": "foobar",
			"foo": "bar"
		}`),
		Provider: getTestProviderState(t, "1.2.3", "registry.terraform.io", "my-org", "foobar"),
		Hash:     12345,
	}

	isEmpty := s.Empty()
	if isEmpty {
		t.Fatalf("expected config to not be reported as empty, but got empty=%v", isEmpty)
	}

	// Zero valued StateStoreConfigState is empty
	s = &StateStoreConfigState{}

	isEmpty = s.Empty()
	if isEmpty != true {
		t.Fatalf("expected config to be reported as empty, but got empty=%v", isEmpty)
	}

	// nil StateStoreConfigState is empty
	s = nil

	isEmpty = s.Empty()
	if isEmpty != true {
		t.Fatalf("expected config to be reported as empty, but got empty=%v", isEmpty)
	}
}

func TestProviderSource_MarshalText(t *testing.T) {

	cases := map[string]struct {
		host           svchost.Hostname
		namespace      string
		typeName       string
		expectedString string
	}{
		"marshals complete ProviderSource to an expected FQN": {
			host:           tfaddr.DefaultProviderRegistryHost,
			namespace:      "hashicorp",
			typeName:       "random",
			expectedString: "registry.terraform.io/hashicorp/random",
		},
		// Unhappy path cases
		"when host is unset, marshaling succeeds with an incomplete FQN": {
			namespace:      "hashicorp",
			typeName:       "random",
			expectedString: "/hashicorp/random",
		},
		"when namespace is unset, marshaling succeeds with an incomplete FQN": {
			host:           tfaddr.DefaultProviderRegistryHost,
			typeName:       "random",
			expectedString: "registry.terraform.io//random",
		},
		"when type is unset, marshaling succeeds with an incomplete FQN": {
			host:           tfaddr.DefaultProviderRegistryHost,
			namespace:      "hashicorp",
			expectedString: "registry.terraform.io/hashicorp/",
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {

			ps := Source{
				// Provider: tfaddr.NewProvider(tc.host, tc.namespace, tc.typeName),
				Provider: tfaddr.Provider{
					Type:      tc.typeName,
					Namespace: tc.namespace,
					Hostname:  tc.host,
				},
			}

			txt, err := ps.MarshalText()
			if err != nil {
				t.Fatal(err)
			}
			if string(txt) != tc.expectedString {
				t.Fatalf("expected marshalled text %q but got %q", tc.expectedString, txt)
			}
		})
	}
}

func TestProviderSource_UnmarshalText(t *testing.T) {

	cases := map[string]struct {
		host          svchost.Hostname
		namespace     string
		typeName      string
		inputText     string
		expectedError string
	}{
		"With a complete FQN string, it unmarshals to expected values": {
			host:      tfaddr.DefaultProviderRegistryHost,
			namespace: "hashicorp",
			typeName:  "random",
			inputText: "registry.terraform.io/hashicorp/random",
		},
		"When hostname is missing from the FQN string, it supplied the default hostname": {
			host:      tfaddr.DefaultProviderRegistryHost,
			namespace: "hashicorp",
			typeName:  "random",
			inputText: "hashicorp/random",
		},
		"When hostname and namespace are missing from the FQN string, it supplied the default hostname and hashicorp namespace": {
			host:      tfaddr.DefaultProviderRegistryHost,
			namespace: "hashicorp",
			typeName:  "random",
			inputText: "random",
		},
		"An error is returned when unmarshaling an empty string": {
			inputText:     "",
			expectedError: "error unmarshaling provider source from backend state file",
		},
		"An error is returned when unmarshaling a malformed FQN: missing host with leading slash": {
			inputText:     "/hashicorp/random",
			expectedError: "error unmarshaling provider source from backend state file",
		},
		"An error is returned when unmarshaling a malformed FQN: missing namespace": {
			inputText:     "registry.terraform.io//random",
			expectedError: "error unmarshaling provider source from backend state file",
		},
		"An error is returned when unmarshaling a malformed FQN: missing type": {
			inputText:     "registry.terraform.io/hashicorp/",
			expectedError: "error unmarshaling provider source from backend state file",
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {

			ps := Source{}

			err := ps.UnmarshalText([]byte(tc.inputText))
			if err != nil {
				if tc.expectedError != "" {
					if !strings.Contains(err.Error(), tc.expectedError) {
						t.Fatalf("expected error to contain the string %q, but got %q", tc.expectedError, err)
					}
					return // stop early in error cases
				}

				t.Fatalf("unexpected error: %q", err)
			}
			if ps.Provider.Hostname != tc.host {
				t.Fatalf("expected host to be %q but got %q", tc.host, ps.Provider.Hostname)
			}
			if ps.Provider.Namespace != tc.namespace {
				t.Fatalf("expected namespace to be %q but got %q", tc.namespace, ps.Provider.Namespace)
			}
			if ps.Provider.Type != tc.typeName {
				t.Fatalf("expected type to be %q but got %q", tc.typeName, ps.Provider.Type)
			}
		})
	}
}
