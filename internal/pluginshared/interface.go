// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pluginshared

import (
	"io"
)

type CustomPluginClient interface {
	Execute(args []string, stdout, stderr io.Writer) int
}
