// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
)

// ProviderMeta represents a "provider_meta" block inside a "terraform" block
// in a module or file.
type ProviderMeta struct {
	Provider string
	Config   hcl.Body

	ProviderRange hcl.Range
	DeclRange     hcl.Range
}

// ValidateProviderMetas checks for metadata declarations with distinct local
// names that resolve to the same provider, including in child and alternate test
// modules. Provider requirements must already have been evaluated.
func (c *Config) ValidateProviderMetas() hcl.Diagnostics {
	var diags hcl.Diagnostics
	c.DeepEach(func(cfg *Config) {
		metas := make(map[addrs.Provider]*ProviderMeta)
		for _, pm := range cfg.Module.ProviderMetaConfigs {
			provider := cfg.Module.ProviderForLocalConfig(addrs.LocalProviderConfig{LocalName: pm.Provider})
			// Duplicate local names are already diagnosed during module construction.
			if existing, exists := metas[provider]; exists && existing.Provider != pm.Provider {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate provider_meta block",
					Detail:   fmt.Sprintf("A provider_meta block for provider %q was already declared at %s. Providers may only have one provider_meta block per module.", existing.Provider, existing.DeclRange),
					Subject:  &pm.DeclRange,
				})
			}
			metas[provider] = pm
		}

		for _, file := range cfg.Module.Tests {
			for _, run := range file.Runs {
				if run.ConfigUnderTest != nil {
					diags = append(diags, run.ConfigUnderTest.ValidateProviderMetas()...)
				}
			}
		}
	})
	return diags
}

func decodeProviderMetaBlock(block *hcl.Block) (*ProviderMeta, hcl.Diagnostics) {
	// provider_meta must be a static map. We can verify this by attempting to
	// evaluate the values.
	attrs, diags := block.Body.JustAttributes()
	if diags.HasErrors() {
		return nil, diags
	}

	for _, attr := range attrs {
		_, d := attr.Expr.Value(nil)
		diags = append(diags, d...)
	}

	// verify that the local name is already localized or produce an error.
	nameDiags := checkProviderNameNormalized(block.Labels[0], block.DefRange)
	diags = append(diags, nameDiags...)
	if nameDiags.HasErrors() {
		return nil, diags
	}

	return &ProviderMeta{
		Provider:      block.Labels[0],
		ProviderRange: block.LabelRanges[0],
		Config:        block.Body,
		DeclRange:     block.DefRange,
	}, diags
}
