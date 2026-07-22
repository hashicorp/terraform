// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
)

func TestFinalizeConfig_WithMockDataSources(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDirWithTests("testdata/valid-modules/with-mock-sources", "tests")
	assertNoDiagnostics(t, diags)

	cfg := testConfig(mod)
	diags = FinalizeConfig(cfg, MockDataLoaderFunc(func(provider *Provider) (*MockData, hcl.Diagnostics) {
		sourcePath := filepath.Join("testdata/valid-modules/with-mock-sources", provider.MockDataExternalSource)
		return parser.LoadMockDataDir(sourcePath, provider.MockDataDuringPlan, hcl.Range{})
	}))
	assertNoDiagnostics(t, diags)

	provider := cfg.Module.Tests["main.tftest.hcl"].Providers["aws"]
	if len(provider.MockData.MockDataSources) != 1 {
		t.Errorf("expected to load 1 mock data source but loaded %d", len(provider.MockData.MockDataSources))
	}
	if len(provider.MockData.MockResources) != 1 {
		t.Errorf("expected to load 1 mock resource but loaded %d", len(provider.MockData.MockResources))
	}
	if provider.MockData.Overrides.Len() != 1 {
		t.Errorf("expected to load 1 override but loaded %d", provider.MockData.Overrides.Len())
	}
}

func TestFinalizeConfig_WithMockDataSourcesInline(t *testing.T) {
	parser := NewParser(nil)
	mod, diags := parser.LoadConfigDirWithTests("testdata/valid-modules/with-mock-sources-inline", "tests")
	assertNoDiagnostics(t, diags)

	cfg := testConfig(mod)
	diags = FinalizeConfig(cfg, MockDataLoaderFunc(func(provider *Provider) (*MockData, hcl.Diagnostics) {
		sourcePath := filepath.Join("testdata/valid-modules/with-mock-sources-inline", provider.MockDataExternalSource)
		return parser.LoadMockDataDir(sourcePath, provider.MockDataDuringPlan, hcl.Range{})
	}))
	assertNoDiagnostics(t, diags)
}
