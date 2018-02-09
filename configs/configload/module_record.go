package configload

import (
	version "github.com/hashicorp/go-version"
)

// moduleRecords represents the stored module's metadata.
// This is compared for equality using '==', so all fields needs to remain
// comparable.
type moduleRecord struct {
	// SourceAddr is the module source string from the config, minus any
	// subdirectory.
	SourceAddr string

	// Key is the locally unique identifier for this module.
	Key string

	// VersionStr is the version specifier string.
	VersionStr string

	// Version is the exact version of the module, which results from parsing
	// VersionStr. nil for un-versioned modules.
	Version *version.Version

	// Dir is the path to the local directory where the module is (or should be)
	// installed.
	Dir string

	// Root is the root directory containing the module. If the module is
	// unpacked from an archive, and not located in the root directory, this is
	// used to direct the loader to the correct subdirectory. This is
	// independent from any subdirectory in the original source string, which
	// may traverse further into the module tree.
	Root string

	// URL is the location of the module source
	URL string

	// Registry is true if this module is sourced from a registry
	Registry bool
}

type moduleRecords []moduleRecord

func (mrs moduleRecords) VersionsForAddr(addr string) moduleRecords {
	var ret moduleRecords
	for _, mr := range mrs {
		if mr.SourceAddr == addr && mr.Version != nil {
			ret = append(ret, mr)
		}
	}
	return ret
}

func (mrs moduleRecords) Newest(constraints version.Constraints) moduleRecord {
	if len(mrs) == 0 {
		panic("Newest called on zero-length moduleRecords")
	}

	ret := mrs[0]
	for i := 1; i < len(mrs); i++ {
		if mrs[i].Version.GreaterThan(ret.Version) {
			ret = mrs[i]
		}
	}

	return ret
}
