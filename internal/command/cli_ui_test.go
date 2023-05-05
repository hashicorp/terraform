// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"testing"

	"github.com/mitchellh/cli"
)

func TestColorizeUi_impl(t *testing.T) {
	var _ cli.Ui = new(ColorizeUi)
}
