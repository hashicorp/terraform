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

func TestBackendConfigState_Config_SetConfig(t *testing.T) {
	// This test only really covers the happy path because Config/SetConfig is
	// largely just a thin wrapper around configschema's "ImpliedType" and
	// cty's json unmarshal/marshal and both of those are well-tested elsewhere.

	s := &BackendConfigState{
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
	// Test Config method
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

	// Test SetConfig method
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

func TestBackendStateConfig_Empty(t *testing.T) {

	// Populated BackendConfigState isn't empty
	s := &BackendConfigState{
		Type: "whatever",
		ConfigRaw: []byte(`{
			"foo": "bar"
		}`),
	}

	isEmpty := s.Empty()
	if isEmpty {
		t.Fatalf("expected config to not be reported as empty, but got empty=%v", isEmpty)
	}

	// Zero values BackendConfigState is empty
	s = &BackendConfigState{}

	isEmpty = s.Empty()
	if isEmpty != true {
		t.Fatalf("expected config to be reported as empty, but got empty=%v", isEmpty)
	}

	// nil BackendConfigState is empty
	s = nil

	isEmpty = s.Empty()
	if isEmpty != true {
		t.Fatalf("expected config to be reported as empty, but got empty=%v", isEmpty)
	}
}

func TestBackendStateConfig_PlanData(t *testing.T) {

	workspace := "default"
	s := &BackendConfigState{
		Type: "whatever",
		ConfigRaw: []byte(`{
			"foo": "bar"
		}`),
		Hash: 123,
	}

	schema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"foo": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}

	plan, err := s.PlanData(schema, nil, workspace)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if plan.Type != s.Type {
		t.Fatalf("incorrect Type value, got %q, want %q", plan.Type, s.Type)
	}
	if plan.Workspace != workspace {
		t.Fatalf("incorrect Workspace value, got %q, want %q", plan.Workspace, workspace)
	}
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
		t.Fatalf("expected config to include one attribute called \"foo\", instead got attribute(s): %s", attrs)
	}
}
