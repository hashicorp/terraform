// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mnptu

import (
	"testing"
)

func TestMockUIOutput(t *testing.T) {
	var _ UIOutput = new(MockUIOutput)
}
