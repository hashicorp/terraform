package langserver

import (
	"path/filepath"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/tfdiags"
)

// configFile reads the config of a single file, for localized analysis.
func configFile(filename string, src []byte) (*configs.File, tfdiags.Diagnostics) {
	// Our config-loading API isn't really designed for loading a single file
	// from an in-memory buffer, so this is a bit roundabout. Maybe in future
	// we can add some better affordances for this use-case.
	dir := filepath.Dir(filename)
	fn := filepath.Base(filename)
	snap := &configload.Snapshot{
		Modules: map[string]*configload.SnapshotModule{
			"": {
				Dir: dir,
				Files: map[string][]byte{
					fn: src,
				},
			},
		},
	}

	loader := configload.NewLoaderFromSnapshot(snap)
	parser := loader.Parser()
	var diags tfdiags.Diagnostics
	f, hclDiags := parser.LoadConfigFile(filename)
	diags = diags.Append(hclDiags)
	return f, diags
}
