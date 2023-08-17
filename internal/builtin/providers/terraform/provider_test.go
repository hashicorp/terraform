// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mnptu

import (
	backendInit "github.com/hashicorp/mnptu/internal/backend/init"
)

func init() {
	// Initialize the backends
	backendInit.Init(nil)
}
