// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduletest

import (
	"sync"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type File struct {
	Config *configs.TestFile

	Name   string
	Status Status

	Runs []*Run

	Diagnostics tfdiags.Diagnostics

	mu sync.Mutex
}

func NewFile(name string, config *configs.TestFile, runs []*Run) *File {
	return &File{
		Name:   name,
		Config: config,
		Runs:   runs,
		mu:     sync.Mutex{},
	}
}

func (f *File) UpdateStatus(status Status) {
	lock := f.Lock()
	defer lock()
	f.Status = f.Status.Merge(status)
}

func (f *File) Lock() func() {
	f.mu.Lock()
	return f.mu.Unlock
}
