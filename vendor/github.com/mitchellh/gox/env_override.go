package main

import (
	"fmt"
	"os"
	"strings"
)

// envOverride overrides the given target based on if there is a
// env var in the format of GOX_{OS}_{ARCH}_{KEY}.
func envOverride(target *string, platform Platform, key string) {
	key = strings.ToUpper(fmt.Sprintf(
		"GOX_%s_%s_%s", platform.OS, platform.Arch, key))
	if v := os.Getenv(key); v != "" {
		*target = v
	}
}
