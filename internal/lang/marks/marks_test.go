// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package marks

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

func TestDeprecationMark(t *testing.T) {
	deprecationWithoutRange := cty.StringVal("OldValue").Mark(NewDeprecation("This is outdated", nil))
	deprecationWithRange := cty.StringVal("OldValue").Mark(NewDeprecation("This is outdated", &hcl.Range{Filename: "example.tf", Start: hcl.Pos{Line: 1, Column: 1}, End: hcl.Pos{Line: 1, Column: 10}}))

	composite := cty.ObjectVal(map[string]cty.Value{
		"foo": deprecationWithRange,
		"bar": deprecationWithoutRange,
		"baz": cty.StringVal("Not deprecated"),
	})

	if !deprecationWithRange.IsMarked() {
		t.Errorf("Expected deprecationWithRange to be marked")
	}
	if !deprecationWithoutRange.IsMarked() {
		t.Errorf("Expected deprecationWithoutRange to be marked")
	}
	if composite.IsMarked() {
		t.Errorf("Expected composite to be marked")
	}

	if !Has(deprecationWithRange, Deprecation) {
		t.Errorf("Expected deprecationWithRange to be marked with Deprecation")
	}
	if !Has(deprecationWithoutRange, Deprecation) {
		t.Errorf("Expected deprecationWithoutRange to be marked with Deprecation")
	}
	if Has(composite, Deprecation) {
		t.Errorf("Expected composite to be marked with Deprecation")
	}

	if !Contains(deprecationWithRange, Deprecation) {
		t.Errorf("Expected deprecationWithRange to be contain Deprecation Mark")
	}
	if !Contains(deprecationWithoutRange, Deprecation) {
		t.Errorf("Expected deprecationWithoutRange to be contain Deprecation Mark")
	}
	if !Contains(composite, Deprecation) {
		t.Errorf("Expected composite to be contain Deprecation Mark")
	}
}
