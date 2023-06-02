// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

//go:build tools
// +build tools

package main

import (
	_ "github.com/hashicorp/copywrite"
)

//go:generate go run github.com/hashicorp/copywrite headers
