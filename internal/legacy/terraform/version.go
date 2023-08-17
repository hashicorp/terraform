// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mnptu

import (
	"github.com/hashicorp/mnptu/version"
)

// Deprecated: Providers should use schema.Provider.mnptuVersion instead
func VersionString() string {
	return version.String()
}
