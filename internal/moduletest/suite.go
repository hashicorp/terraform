// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package moduletest

type Suite struct {
	Status Status

	Files map[string]*File
}
