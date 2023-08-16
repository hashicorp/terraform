// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduletest

type Suite struct {
	Status Status

	Files map[string]*File
}
