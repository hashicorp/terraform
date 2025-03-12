// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package marks

import (
	"testing"

	"github.com/zclconf/go-cty/cty"
)

func TestMarkDeprecated(t *testing.T) {
	val := cty.ObjectVal(map[string]cty.Value{
		"key":  cty.StringVal("value").Mark(Deprecated{Message: "foo"}),
		"key2": cty.StringVal("value2"),
	})

	marks := DeprecationMarks(val)
	if len(marks) != 1 {
		t.Fatalf("expected 1 mark, got %d", len(marks))
	}
	if marks[0].Message != "foo" {
		t.Fatalf("expected message 'foo', got %q", marks[0].Message)
	}
}
