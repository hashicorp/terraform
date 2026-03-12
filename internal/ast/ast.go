// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package ast

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type AST struct{}

func FromConfig(files map[string]*hcl.File) (*AST, tfdiags.Diagnostics) {
	return &AST{}, nil
}

func WriteAST(ast *AST) (map[string][]byte, tfdiags.Diagnostics) {
	return nil, nil
}
