// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"testing"
)

func TestFunctionDescriptions(t *testing.T) {
	scope := &Scope{
		ConsoleMode: true,
	}
	for name, fn := range scope.Functions() {
		if fn.Description() == "" {
			t.Errorf("missing DescriptionList entry for function %q", name)
		}
	}
}
