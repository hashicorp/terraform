package moduleref

import "github.com/hashicorp/terraform/internal/modsdir"

// ModuleRecord is the implementation of a module entry defined in the module
// manifest including config reference information
type Record struct {
	Source                    string `json:"Source"`
	Version                   string `json:"Version"`
	Key                       string `json:"Key"`
	Dir                       string `json:"Dir"`
	ReferencedInConfiguration bool   `json:"Referenced"`
}

// ModuleRecordManifest is the ModuleView implementation of the module manifest
type Manifest struct {
	Records []Record `json:"Modules"`
}

// AddModuleEntry will append an module manifest record and includes whether or
// not the entry is referenced by configuration.
func (m *Manifest) AddModuleEntry(entry modsdir.Record, referenced bool) {
	m.Records = append(m.Records, Record{
		Source:                    entry.SourceAddr,
		Version:                   entry.VersionStr,
		Key:                       entry.Key,
		Dir:                       entry.Dir,
		ReferencedInConfiguration: referenced,
	})
}
