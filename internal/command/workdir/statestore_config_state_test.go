// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"bytes"
	"maps"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestStateStoreConfigState_Config_SetConfig(t *testing.T) {
	// This test only really covers the happy path because Config/SetConfig is
	// largely just a thin wrapper around configschema's "ImpliedType" and
	// cty's json unmarshal/marshal and both of those are well-tested elsewhere.
	pConfig := `{
		"foo": "bar"
	}`
	s := &StateStoreConfigState{
		Type: "whatever",
		ConfigRaw: []byte(`{
			"foo": "bar",
			"fizz": "buzz"
		}`),
		Provider: getTestProviderState(t, "1.2.3", "registry.terraform.io", "my-org", "foobar", pConfig),
		Hash:     12345,
	}

	ssSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"fizz": {
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
	got, err := s.Config(ssSchema)
	want := cty.ObjectVal(map[string]cty.Value{
		"foo":  cty.StringVal("bar"),
		"fizz": cty.StringVal("buzz"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}

	// Test SetConfig method
	err = s.SetConfig(cty.ObjectVal(map[string]cty.Value{
		"fizz": cty.StringVal("abc"),
		"foo":  cty.StringVal("def"),
	}), ssSchema)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	gotRaw := s.ConfigRaw
	wantRaw := []byte(`{"fizz":"abc","foo":"def"}`)
	if !bytes.Equal(wantRaw, gotRaw) {
		t.Errorf("wrong raw config after encode\ngot:  %s\nwant: %s", gotRaw, wantRaw)
	}
}

func TestProviderConfigState_Config_SetConfig(t *testing.T) {
	// This test only really covers the happy path because Config/SetConfig is
	// largely just a thin wrapper around configschema's "ImpliedType" and
	// cty's json unmarshal/marshal and both of those are well-tested elsewhere.

	pConfig := `{
		"foo": "bar"
	}`
	s := getTestProviderState(t, "1.2.3", "registry.terraform.io", "my-org", "foobar", pConfig)
	s.ConfigRaw = []byte(`{
		"credentials": "./creds.json",
		"region": "saturn"
	}
	`)

	pSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"credentials": {
				Type:     cty.String,
				Required: true,
			},
			"region": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	// Test Config method
	got, err := s.Config(pSchema)
	want := cty.ObjectVal(map[string]cty.Value{
		"credentials": cty.StringVal("./creds.json"),
		"region":      cty.StringVal("saturn"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if diff := cmp.Diff(want, got, ctydebug.CmpOptions); diff != "" {
		t.Errorf("wrong result\n%s", diff)
	}

	// Test SetConfig method
	err = s.SetConfig(cty.ObjectVal(map[string]cty.Value{
		"credentials": cty.StringVal("abc"),
		"region":      cty.StringVal("def"),
	}), pSchema)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	gotRaw := s.ConfigRaw
	wantRaw := []byte(`{"credentials":"abc","region":"def"}`)
	if !bytes.Equal(wantRaw, gotRaw) {
		t.Errorf("wrong raw config after encode\ngot:  %s\nwant: %s", gotRaw, wantRaw)
	}
}

func TestStateStoreConfigState_Empty(t *testing.T) {
	// Populated StateStoreConfigState isn't empty
	pConfig := `{
		"foo": "bar"
	}`
	s := &StateStoreConfigState{
		Type: "whatever",
		ConfigRaw: []byte(`{
			"fizz": "buzz",
			"foo": "bar"
		}`),
		Provider: getTestProviderState(t, "1.2.3", "registry.terraform.io", "my-org", "foobar", pConfig),
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

func TestProviderConfigState_Empty(t *testing.T) {
	// Populated StateStoreConfigState isn't empty
	pConfig := `{
		"foo": "bar"
	}`
	s := getTestProviderState(t, "1.2.3", "registry.terraform.io", "my-org", "foobar", pConfig)

	isEmpty := s.Empty()
	if isEmpty {
		t.Fatalf("expected config to not be reported as empty, but got empty=%v", isEmpty)
	}

	// Zero valued ProviderConfigState is empty
	s = &ProviderConfigState{}

	isEmpty = s.Empty()
	if isEmpty != true {
		t.Fatalf("expected config to be reported as empty, but got empty=%v", isEmpty)
	}

	// nil ProviderConfigState is empty
	s = nil

	isEmpty = s.Empty()
	if isEmpty != true {
		t.Fatalf("expected config to be reported as empty, but got empty=%v", isEmpty)
	}
}

func TestStateStoreConfigState_PlanData(t *testing.T) {

	workspace := "default"

	pConfig := `{
	"credentials": "./creds.json"
}`
	provider := getTestProviderState(t, "1.2.3", "registry.terraform.io", "my-org", "foobar", pConfig)

	s := &StateStoreConfigState{
		Type: "whatever",
		ConfigRaw: []byte(`{
			"foo": "bar"
		}`),
		Hash:     123,
		Provider: provider,
	}

	ssSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	pSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"credentials": {
				Type:     cty.String,
				Required: true,
			},
		},
	}

	plan, err := s.PlanData(ssSchema, pSchema, workspace)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Check state store details
	if plan.Type != s.Type {
		t.Fatalf("incorrect Type value, got %q, want %q", plan.Type, s.Type)
	}
	if plan.Workspace != workspace {
		t.Fatalf("incorrect Workspace value, got %q, want %q", plan.Workspace, workspace)
	}
	// Config
	imType, err := plan.Config.ImpliedType()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	val, err := plan.Config.Decode(imType)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	valMap := val.AsValueMap()
	if len(valMap) != 1 || valMap["foo"] == cty.NilVal {
		attrs := slices.Sorted(maps.Keys(valMap))
		t.Fatalf("expected plan's config data to include one attribute called \"foo\", instead got attribute(s): %s", attrs)
	}

	// Check provider details
	if plan.Provider == nil {
		t.Fatal("expected plan to include provider data, but it was nil")
	}
	if plan.Provider.Version != s.Provider.Version {
		t.Fatalf("incorrect provider Version value, got %q, want %q", plan.Workspace, workspace)
	}
	if plan.Provider.Source.Hostname != s.Provider.Source.Hostname ||
		plan.Provider.Source.Namespace != s.Provider.Source.Namespace ||
		plan.Provider.Source.Type != s.Provider.Source.Type {
		t.Fatalf("incorrect provider Version value, got %q, want %q", plan.Workspace, workspace)
	}
	// Config
	imType, err = plan.Provider.Config.ImpliedType()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	val, err = plan.Provider.Config.Decode(imType)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	valMap = val.AsValueMap()
	if len(valMap) != 1 || valMap["credentials"] == cty.NilVal {
		attrs := slices.Sorted(maps.Keys(valMap))
		t.Fatalf("expected plan's provider config data to include one attribute called \"credentials\", instead got attribute(s): %s", attrs)
	}

}
