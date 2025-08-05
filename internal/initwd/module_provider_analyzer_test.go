// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package initwd

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/registry"
)

func TestModuleProviderAnalyzer_extractProviderRequirementsFromModule(t *testing.T) {
	tests := []struct {
		name           string
		packageAddr    addrs.ModuleRegistryPackage
		versionStr     string
		expectedReqs   providerreqs.Requirements
		expectedError  bool
		skipIfNoAcc    bool
	}{
		{
			name: "valid module with provider requirements",
			packageAddr: addrs.ModuleRegistryPackage{
				Host:         "registry.terraform.io",
				Namespace:    "terraform-aws-modules",
				Name:         "vpc",
				TargetSystem: "aws",
			},
			versionStr:   "5.21.0",
			skipIfNoAcc:  true, // This test requires network access to registry
			expectedReqs: providerreqs.Requirements{
				addrs.MustParseProviderSourceString("hashicorp/aws"): providerreqs.MustParseVersionConstraints(">= 5.0"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipIfNoAcc && testing.Short() {
				t.Skip("skipping network-dependent test in short mode")
			}
			
			loader, close := configload.NewLoaderForTests(t)
			defer close()
			
			inst := NewModuleInstaller("", loader, registry.NewClient(nil, nil))
			analyzer := NewModuleProviderAnalyzer(inst)
			
			reqs, diags := analyzer.extractProviderRequirementsFromModule(t.Context(), tt.packageAddr, tt.versionStr)
			
			if tt.expectedError {
				if !diags.HasErrors() {
					t.Errorf("expected error, but got none")
				}
				return
			}
			
			if diags.HasErrors() {
				t.Errorf("unexpected error: %s", diags.Err())
				return
			}
			
			// For network tests, we just verify we got a non-empty result
			// since exact provider requirements may change over time
			if tt.skipIfNoAcc {
				if len(reqs) == 0 {
					t.Errorf("expected non-empty provider requirements")
				}
				return
			}
			
			if diff := cmp.Diff(tt.expectedReqs, reqs); diff != "" {
				t.Errorf("wrong provider requirements\n%s", diff)
			}
		})
	}
}

func TestModuleProviderAnalyzer_creation(t *testing.T) {
	loader, close := configload.NewLoaderForTests(t)
	defer close()
	
	inst := NewModuleInstaller("", loader, registry.NewClient(nil, nil))
	analyzer := NewModuleProviderAnalyzer(inst)
	
	// Just verify it was created successfully
	if analyzer == nil {
		t.Errorf("NewModuleProviderAnalyzer returned nil")
	}
}