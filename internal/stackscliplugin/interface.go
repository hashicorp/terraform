// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackscliplugin

import (
	"io"
)

type StacksCLI1 interface {
	Execute(args []string, stdout, stderr io.Writer) int
}
