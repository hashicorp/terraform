// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
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
