// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloudplugin

import (
	"io"
)

type Cloud1 interface {
	Execute(args []string, stdout, stderr io.Writer) int
}
