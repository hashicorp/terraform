package componentstree

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/terraform-ng/internal/ngaddrs"
	"github.com/hashicorp/terraform/internal/terraform-ng/internal/tfcomponents"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// LocalOnlyConfigLoader is a ConfigLoader that only supports local source
// addresses, and does so by interpreting them as direct local filesystem
// paths, relative to some base directory.
//
// This is a reasonable loader to use in tests that don't need to exercise
// the machinery for fetching remote sources.
type LocalOnlyConfigLoader struct {
	baseDir string
}

func NewLocalOnlyConfigLoader(baseDir string) LocalOnlyConfigLoader {
	return LocalOnlyConfigLoader{
		baseDir: filepath.Clean(baseDir),
	}
}

var _ ConfigLoader = LocalOnlyConfigLoader{}

func (l LocalOnlyConfigLoader) LoadConfig(path []ngaddrs.ComponentGroupCall, sourceAddr addrs.ModuleSource) (*tfcomponents.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	localAddr, ok := sourceAddr.(addrs.ModuleSourceLocal)
	if !ok {
		// FIXME: It's annoying that we can't attribute this back to a
		// source location, but this method signature is designed to support
		// both root calls and child calls, and root calls don't appear in
		// the source code anywhere.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported component group source address",
			fmt.Sprintf("Can't use component group defined at %q: only local filesystem locations are supported here.", sourceAddr),
		))
		return nil, diags
	}

	localPath := filepath.FromSlash(string(localAddr))
	fullPath := filepath.Join(l.baseDir, localPath)
	return tfcomponents.LoadConfigFile(fullPath)
}
