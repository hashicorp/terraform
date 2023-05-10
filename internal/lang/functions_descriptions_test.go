// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package lang

import (
	"testing"

	"github.com/hashicorp/terraform/internal/lang/funcs"
)

func TestFunctionDescriptions(t *testing.T) {
	scope := &Scope{
		ConsoleMode: true,
	}
	// This will implicitly test the parameter description count since
	// WithNewDescriptions will panic if the number doesn't match.
	allFunctions := scope.Functions()

	// plantimestamp isn't available with ConsoleMode: true
	expectedFunctionCount := len(funcs.DescriptionList) - 1

	if len(allFunctions) != expectedFunctionCount {
		t.Errorf("DescriptionList length expected: %d, got %d", len(allFunctions), expectedFunctionCount)
	}

	for name := range allFunctions {
		_, ok := funcs.DescriptionList[name]
		if !ok {
			t.Errorf("missing DescriptionList entry for function %q", name)
		}
	}
}
