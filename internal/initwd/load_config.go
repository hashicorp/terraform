package initwd

import (
	"fmt"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/hashicorp/terraform/internal/earlyconfig"
	"github.com/hashicorp/terraform/internal/modsdir"
	"github.com/hashicorp/terraform/tfdiags"
)

// LoadConfig loads a full configuration tree that has previously had all of
// its dependent modules installed to the given modulesDir using a
// ModuleInstaller.
//
// This uses the early configuration loader and thus only reads top-level
// metadata from the modules in the configuration. Most callers should use
// the configs/configload package to fully load a configuration.
func LoadConfig(rootDir, modulesDir string) (*earlyconfig.Config, tfdiags.Diagnostics) {
	rootMod, diags := earlyconfig.LoadModule(rootDir)
	if rootMod == nil {
		return nil, diags
	}

	manifest, err := modsdir.ReadManifestSnapshotForDir(modulesDir)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to read module manifest",
			fmt.Sprintf("Terraform failed to read its manifest of locally-cached modules: %s.", err),
		))
		return nil, diags
	}

	return earlyconfig.BuildConfig(rootMod, earlyconfig.ModuleWalkerFunc(
		func(req *earlyconfig.ModuleRequest) (*tfconfig.Module, *version.Version, tfdiags.Diagnostics) {
			var diags tfdiags.Diagnostics

			key := manifest.ModuleKey(req.Path)
			record, exists := manifest[key]
			if !exists {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Module not installed",
					fmt.Sprintf("Module %s is not yet installed. Run \"terraform init\" to install all modules required by this configuration.", req.Path.String()),
				))
				return nil, nil, diags
			}

			mod, mDiags := earlyconfig.LoadModule(record.Dir)
			diags = diags.Append(mDiags)
			return mod, record.Version, diags
		},
	))
}
