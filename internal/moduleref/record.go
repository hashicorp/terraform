// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleref

import "github.com/hashicorp/terraform/internal/modsdir"

const FormatVersion = "1.0"

// ModuleRecord is the implementation of a module entry defined in the module
// manifest that is declared by configuration.
type Record struct {
	Key     string `json:"key"`
	Source  string `json:"source"`
	Version string `json:"version"`
}

// ModuleRecordManifest is the view implementation of module entries declared
// in configuration
type Manifest struct {
	FormatVersion string   `json:"format_version"`
	Records       []Record `json:"modules"`
}

func (m *Manifest) addModuleEntry(entry modsdir.Record) {
	m.Records = append(m.Records, Record{
		Key:     entry.Key,
		Source:  entry.SourceAddr,
		Version: entry.VersionStr,
	})
}
