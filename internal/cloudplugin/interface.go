// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package cloudplugin

type Cloud1 interface {
	Execute(args []string) int
}
