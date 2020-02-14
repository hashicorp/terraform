package langserver

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/tfdiags"
)

type file struct {
	fullPath string
	content  []byte

	ls   sourceLines
	errs bool
	ast  *hcl.File

	parseDiags tfdiags.Diagnostics
	fullDiags  tfdiags.Diagnostics
}

func NewFile(fullPath string, content []byte) *file {
	return &file{fullPath: fullPath, content: content}
}

func (f *file) lines() sourceLines {
	if f.ls == nil {
		f.ls = makeSourceLines(f.fullPath, f.content)
	}

	return f.ls
}

func (f *file) ResolveRefAtByteOffset(offset int) *addrs.Reference {
	ast := f.hclAST()

	hclPos := f.lines().byteOffsetToHCL(offset)

	return refAtPos(hclPos, ast)
}

func (f *file) hclAST() *hcl.File {
	if f.ast != nil {
		return f.ast
	}

	hf, diags := hclsyntax.ParseConfig(f.content, f.fullPath, hcl.Pos{Line: 1, Column: 1})
	f.parseDiags = nil
	f.parseDiags = f.parseDiags.Append(diags)
	if diags.HasErrors() {
		return nil
	}

	f.ast = hf

	return hf
}
