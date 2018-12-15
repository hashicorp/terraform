package langserver

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// formatSource will apply hclwrite formatting to the given source code
// if it has valid syntax, or just return it verbatim if not. The second
// argument is true if the returned buffer might be different than the given
// buffer, or false if it's guaranteed identical (allowing the caller to skip
// invalidating caches, etc.)
func formatSource(in []byte) ([]byte, bool) {
	_, diags := hclsyntax.ParseConfig(in, "", hcl.Pos{})
	if diags.HasErrors() {
		return in, false
	}

	return hclwrite.Format(in), true
}
