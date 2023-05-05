// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package terraform

import (
	"testing"
)

func TestCallbackUIOutput_impl(t *testing.T) {
	var _ UIOutput = new(CallbackUIOutput)
}
