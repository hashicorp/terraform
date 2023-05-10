// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"encoding/json"
	"testing"

	"github.com/mitchellh/cli"
)

func TestMetadataFunctions_error(t *testing.T) {
	ui := new(cli.MockUi)
	c := &MetadataFunctionsCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	// This test will always error because it's missing the -json flag
	if code := c.Run(nil); code != 1 {
		t.Fatalf("expected error, got:\n%s", ui.OutputWriter.String())
	}
}

func TestMetadataFunctions_output(t *testing.T) {
	ui := new(cli.MockUi)
	m := Meta{Ui: ui}
	c := &MetadataFunctionsCommand{Meta: m}

	if code := c.Run([]string{"-json"}); code != 0 {
		t.Fatalf("wrong exit status %d; want 0\nstderr: %s", code, ui.ErrorWriter.String())
	}

	var got functions
	gotString := ui.OutputWriter.String()
	err := json.Unmarshal([]byte(gotString), &got)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Signatures) < 100 {
		t.Fatalf("expected at least 100 function signatures, got %d", len(got.Signatures))
	}

	// check if one particular stable function is correct
	gotMax, ok := got.Signatures["max"]
	wantMax := "{\"description\":\"`max` takes one or more numbers and returns the greatest number from the set.\",\"return_type\":\"number\",\"variadic_parameter\":{\"name\":\"numbers\",\"type\":\"number\"}}"
	if !ok {
		t.Fatal(`missing function signature for "max"`)
	}
	if string(gotMax) != wantMax {
		t.Fatalf("wrong function signature for \"max\":\ngot: %q\nwant: %q", gotMax, wantMax)
	}

	stderr := ui.ErrorWriter.String()
	if stderr != "" {
		t.Fatalf("expected empty stderr, got:\n%s", stderr)
	}

	// test that ignored functions are not part of the json
	for _, v := range ignoredFunctions {
		_, ok := got.Signatures[v]
		if ok {
			t.Fatalf("found ignored function %q inside output", v)
		}
	}
}

type functions struct {
	FormatVersion string                     `json:"format_version"`
	Signatures    map[string]json.RawMessage `json:"function_signatures,omitempty"`
}
