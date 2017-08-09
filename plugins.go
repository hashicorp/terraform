package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
)

// globalPluginDirs returns directories that should be searched for
// globally-installed plugins (not specific to the current configuration).
//
// Earlier entries in this slice get priority over later when multiple copies
// of the same plugin version are found, but newer versions always override
// older versions where both satisfy the provider version constraints.
func globalPluginDirs() []string {
	var ret []string
	// Look in ~/.terraform.d/plugins/ , or its equivalent on non-UNIX
	dir, err := ConfigDir()
	if err != nil {
		log.Printf("[ERROR] Error finding global config directory: %s", err)
	} else {
		machineDir := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
		ret = append(ret, filepath.Join(dir, "plugins", machineDir))
	}

	return ret
}
