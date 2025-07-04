// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduletest

import (
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type File struct {
	Config *configs.TestFile

	Name   string
	Status Status

	Runs []*Run

	Diagnostics tfdiags.Diagnostics

	sync.Mutex
}

func NewFile(name string, config *configs.TestFile, runs []*Run) *File {
	return &File{
		Name:   name,
		Config: config,
		Runs:   runs,
		Mutex:  sync.Mutex{},
	}
}

func (f *File) UpdateStatus(status Status) {
	f.Lock()
	defer f.Unlock()
	f.Status = f.Status.Merge(status)
}

func (f *File) GetStatus() Status {
	f.Lock()
	defer f.Unlock()
	return f.Status
}

func (f *File) AppendDiagnostics(diags tfdiags.Diagnostics) {
	f.Lock()
	defer f.Unlock()
	f.Diagnostics = f.Diagnostics.Append(diags)

	if diags.HasErrors() {
		f.Status = f.Status.Merge(Error)
	}
}

// WithSourceCode updates the file's runs with their source code
// extracted from the HCL file.
func (f *File) WithSourceCode() (diags tfdiags.Diagnostics) {
	hfile, hdiags := hclwrite.ParseConfig(f.Config.Source, f.Name, hcl.InitialPos)
	diags = diags.Append(hdiags)
	if hdiags.HasErrors() {
		return diags
	}
	idx := 0
	for _, bl := range hfile.Body().Blocks() {
		if bl.Type() == "run" {
			run := f.Runs[idx]
			tokens := bl.BuildTokens(nil)
			codeStr := string(tokens.Bytes())
			run.Source = codeStr
			idx++
		}
	}
	return diags
}
