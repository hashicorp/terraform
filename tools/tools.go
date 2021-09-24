//go:build tools
// +build tools

package tools

import (
	_ "github.com/golang/mock/mockgen"
	_ "github.com/golang/protobuf/protoc-gen-go"
	_ "github.com/mitchellh/gox"
	_ "github.com/nishanths/exhaustive"
	_ "golang.org/x/tools/cmd/cover"
	_ "golang.org/x/tools/cmd/stringer"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
