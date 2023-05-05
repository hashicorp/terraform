// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build tools
// +build tools

package tools

// This file tracks some external tools we use during development and release
// processes. These are not used at runtime but having them here allows the
// Go toolchain to see that we need to include them in go.mod and go.sum.

import (
	_ "github.com/nishanths/exhaustive/cmd/exhaustive"
	_ "golang.org/x/tools/cmd/stringer"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
