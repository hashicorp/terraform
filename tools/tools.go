// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build tools
// +build tools

package tools

import (
	_ "github.com/mitchellh/gox"
	_ "github.com/nishanths/exhaustive"
	_ "go.uber.org/mock/mockgen"
	_ "golang.org/x/tools/cmd/cover"
	_ "golang.org/x/tools/cmd/stringer"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
