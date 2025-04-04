package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
)

type QueryFile struct {
	Providers map[string]*Provider

	Lists []*List
}

type List struct {
	Name string
}

func loadQueryFile(body hcl.Body) (*QueryFile, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	tf := &QueryFile{
		Providers: make(map[string]*Provider),
	}

	content, contentDiags := body.Content(testFileSchema)
	diags = append(diags, contentDiags...)
	if diags.HasErrors() {
		return nil, diags
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "list":
		// TODO
		case "provider":
			provider, providerDiags := decodeProviderBlock(block, true)
			diags = append(diags, providerDiags...)
			if provider != nil {
				key := provider.moduleUniqueKey()
				if previous, exists := tf.Providers[key]; exists {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate provider block",
						Detail:   fmt.Sprintf("A provider for %s is already defined at %s.", key, previous.NameRange),
						Subject:  provider.DeclRange.Ptr(),
					})
					continue
				}
				tf.Providers[key] = provider
			}
		}
	}

	return tf, diags
}
