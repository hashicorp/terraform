// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleref

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/internal/addrs"
)

const FormatVersion = "1.0"

// ModuleRecord is the implementation of a module entry defined in the module
// manifest that is declared by configuration.
type Record struct {
	Key                string
	Source             addrs.ModuleSource
	Version            *version.Version
	VersionConstraints version.Constraints
	Children           Records
}

// ModuleRecordManifest is the view implementation of module entries declared
// in configuration
type Manifest struct {
	FormatVersion string
	Records       Records
}

func (m *Manifest) addModuleEntry(entry *Record) {
	m.Records = append(m.Records, entry)
}

func (r *Record) addChild(child *Record) {
	r.Children = append(r.Children, child)
}

type Records []*Record

func (r Records) Len() int {
	return len(r)
}
func (r Records) Less(i, j int) bool {
	return r[i].Key < r[j].Key
}
func (r Records) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}
