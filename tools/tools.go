// +build tools

package tools

import (
	_ "github.com/golang/mock/mockgen"
	_ "github.com/mitchellh/gox"
	_ "golang.org/x/tools/cmd/cover"
	_ "golang.org/x/tools/cmd/stringer"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
