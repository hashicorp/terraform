// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package marks

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestDeprecationMark(t *testing.T) {
	deprecation := cty.StringVal("OldValue").Mark(NewDeprecation("This is outdated", ""))

	composite := cty.ObjectVal(map[string]cty.Value{
		"foo": deprecation,
		"bar": deprecation,
		"baz": cty.StringVal("Not deprecated"),
	})

	if !deprecation.IsMarked() {
		t.Errorf("Expected deprecation to be marked")
	}
	if composite.IsMarked() {
		t.Errorf("Expected composite to be marked")
	}

	if !Has(deprecation, Deprecation) {
		t.Errorf("Expected deprecation to be marked with Deprecation")
	}
	if Has(composite, Deprecation) {
		t.Errorf("Expected composite to be marked with Deprecation")
	}

	if !Contains(deprecation, Deprecation) {
		t.Errorf("Expected deprecation to be contain Deprecation Mark")
	}
	if !Contains(composite, Deprecation) {
		t.Errorf("Expected composite to be contain Deprecation Mark")
	}
}
