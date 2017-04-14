package main

import (
	"log"
	"path/filepath"

	"github.com/kardianos/osext"
)

// globalPluginDirs returns directories that should be searched for
// globally-installed plugins (not specific to the current configuration).
//
// Earlier entries in this slice get priority over later when multiple copies
// of the same plugin version are found, but newer versions always override
// older versions where both satisfy the provider version constraints.
func globalPluginDirs() []string {
	var ret []string

	// Look in the same directory as the Terraform executable.
	// If found, this replaces what we found in the config path.
	exePath, err := osext.Executable()
	if err != nil {
		log.Printf("[ERROR] Error discovering exe directory: %s", err)
	} else {
		ret = append(ret, filepath.Dir(exePath))
	}

	// Look in ~/.terraform.d/plugins/ , or its equivalent on non-UNIX
	dir, err := ConfigDir()
	if err != nil {
		log.Printf("[ERROR] Error finding global config directory: %s", err)
	} else {
		ret = append(ret, filepath.Join(dir, "plugins"))
	}

	return ret
}
