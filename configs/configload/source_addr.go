package configload

import (
	"strings"

	"github.com/hashicorp/go-getter"

	"github.com/hashicorp/terraform/registry/regsrc"
)

var localSourcePrefixes = []string{
	"./",
	"../",
	".\\",
	"..\\",
}

func isLocalSourceAddr(addr string) bool {
	for _, prefix := range localSourcePrefixes {
		if strings.HasPrefix(addr, prefix) {
			return true
		}
	}
	return false
}

func isRegistrySourceAddr(addr string) bool {
	_, err := regsrc.ParseModuleSource(addr)
	return err == nil
}

// splitAddrSubdir splits the given address (which is assumed to be a
// registry address or go-getter-style address) into a package portion
// and a sub-directory portion.
//
// The package portion defines what should be downloaded and then the
// sub-directory portion, if present, specifies a sub-directory within
// the downloaded object (an archive, VCS repository, etc) that contains
// the module's configuration files.
//
// The subDir portion will be returned as empty if no subdir separator
// ("//") is present in the address.
func splitAddrSubdir(addr string) (packageAddr, subDir string) {
	return getter.SourceDirSubdir(addr)
}
