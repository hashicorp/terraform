// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"testing"

	"github.com/hashicorp/cli"
)

func TestColorizeUi_impl(t *testing.T) {
	var _ cli.Ui = new(ColorizeUi)
}
