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

	lock sync.Mutex
}

func NewFile(name string, config *configs.TestFile, runs []*Run) *File {
	return &File{
		Name:   name,
		Config: config,
		Runs:   runs,
		lock:   sync.Mutex{},
	}
}

func (f *File) UpdateStatus(status Status) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.Status = f.Status.Merge(status)
}

func (f *File) Lock() func() {
	f.lock.Lock()
	return f.lock.Unlock
}
