package main

import (
	"fmt"
)

func mainListOSArch(version string) int {
	fmt.Printf(
		"Supported OS/Arch combinations for %s are shown below. The \"default\"\n"+
			"boolean means that if you don't specify an OS/Arch, it will be\n"+
			"included by default. If it isn't a default OS/Arch, you must explicitly\n"+
			"specify that OS/Arch combo for Gox to use it.\n\n",
		version)
	for _, p := range SupportedPlatforms(version) {
		fmt.Printf("%s\t(default: %v)\n", p.String(), p.Default)
	}

	return 0
}
