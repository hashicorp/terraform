// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform/internal/command/views"
)

func TestMetadataFunctions_error(t *testing.T) {
	t.Parallel()
	view, done := testView(t)
	c := &MetadataFunctionsCommand{
		Meta: Meta{
			View: view,
		},
	}

	// This test will always error because it's missing the -json flag
	if code := c.Run(nil); code != 1 {
		t.Fatalf("expected error, got:\n%s", done(t).Stdout())
	}
}

func TestMetadataFunctions_output(t *testing.T) {
	t.Parallel()
	view, done := testView(t)
	c := &MetadataFunctionsCommand{
		Meta: Meta{
			View: view,
		},
	}

	if code := c.Run([]string{"-json"}); code != 0 {
		t.Fatalf("wrong exit status %d; want 0\nstderr: %s", code, done(t).Stderr())
	}
	output := done(t)

	var got views.MetadataFunctionsOutput
	gotString := output.Stdout()
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
	if !ok || gotMax == nil {
		t.Fatal(`missing function signature for "max"`)
	}
	gotMaxBytes, err := json.Marshal(gotMax)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotMaxBytes) != wantMax {
		t.Fatalf("wrong function signature for \"max\":\ngot: %q\nwant: %q", string(gotMaxBytes), wantMax)
	}

	stderr := output.Stderr()
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
