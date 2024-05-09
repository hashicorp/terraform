// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/mitchellh/colorstring"
)

func TestBuildDeprecationWarning(t *testing.T) {
	workspaceDeprecations := &WorkspaceDeprecationInfo{
		ModuleDeprecationInfos: []*ModuleDeprecationInfo{
			{
				SourceName: "test1",
				RegistryDeprecation: &RegistryModuleDeprecation{
					Version: "1.0.0",
					Link:    "https://test1.com",
					Message: "Deprecation message for module test1",
				},
				ExternalDependencies: []*ModuleDeprecationInfo{
					{
						SourceName: "test1-external-dependency",
						RegistryDeprecation: &RegistryModuleDeprecation{
							Version: "1.0.0",
							Link:    "https://test1-external-dependency.com",
							Message: "Deprecation message for module test1-external-dependency",
						},
					},
				},
			},
			{
				SourceName: "test2",
				RegistryDeprecation: &RegistryModuleDeprecation{
					Version: "1.0.0",
					Link:    "https://test2.com",
					Message: "Deprecation message for module test2",
				},
				ExternalDependencies: []*ModuleDeprecationInfo{
					{
						SourceName: "test2-external-dependency",
						RegistryDeprecation: &RegistryModuleDeprecation{
							Version: "1.0.0",
							Link:    "https://test2-external-dependency.com",
							Message: "Deprecation message for module test2-external-dependency",
						},
					},
					{
						SourceName: "test2b-external-dependency",
						RegistryDeprecation: &RegistryModuleDeprecation{
							Version: "1.0.0",
							Link:    "https://test2b-external-dependency.com",
							Message: "Deprecation message for module test2b-external-dependency",
						},
					},
				},
			},
			{
				SourceName: "test3",
				RegistryDeprecation: &RegistryModuleDeprecation{
					Version: "1.0.0",
					Link:    "https://test3.com",
					Message: "Deprecation message for module test3",
				},
				ExternalDependencies: []*ModuleDeprecationInfo{},
			},
		},
	}

	color := colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: false,
		Reset:   true,
	}

	detailStringArray := []string{
		color.Color("[reset][bold]Version 1.0.0 of test1[reset]"), "Deprecation message for module test1", "Link for more information: https://test1.com", color.Color("[reset][bold]Version 1.0.0 of test1-external-dependency (Root: test1 -> test1-external-dependency)[reset]"), "Deprecation message for module test1-external-dependency", "Link for more information: https://test1-external-dependency.com", color.Color("[reset][bold]Version 1.0.0 of test2[reset]"), "Deprecation message for module test2", "Link for more information: https://test2.com", color.Color("[reset][bold]Version 1.0.0 of test2-external-dependency (Root: test2 -> test2-external-dependency)[reset]"), "Deprecation message for module test2-external-dependency", "Link for more information: https://test2-external-dependency.com", color.Color("[reset][bold]Version 1.0.0 of test2b-external-dependency (Root: test2 -> test2b-external-dependency)[reset]"), "Deprecation message for module test2b-external-dependency", "Link for more information: https://test2b-external-dependency.com", color.Color("[reset][bold]Version 1.0.0 of test3[reset]"), "Deprecation message for module test3", "Link for more information: https://test3.com",
	}
	diagWant := &hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Deprecated modules found, consider installing updated versions. The following are affected:",
		Detail:   strings.Join(detailStringArray, "\n\n"),
		Extra: &ModuleDeprecationDiagnosticExtra{
			MessageCode: "module_deprecation_warning",
			Deprecations: []*ModuleDeprecationDiagnosticExtraDeprecationItem{
				{
					Version:            "1.0.0",
					SourceName:         "test1",
					DeprecationMessage: "Deprecation message for module test1",
					Link:               "https://test1.com",
				},
				{
					Version:            "1.0.0",
					SourceName:         "test1-external-dependency",
					DeprecationMessage: "Deprecation message for module test1-external-dependency",
					Link:               "https://test1-external-dependency.com",
				},
				{
					Version:            "1.0.0",
					SourceName:         "test2",
					DeprecationMessage: "Deprecation message for module test2",
					Link:               "https://test2.com",
				},
				{
					Version:            "1.0.0",
					SourceName:         "test2-external-dependency",
					DeprecationMessage: "Deprecation message for module test2-external-dependency",
					Link:               "https://test2-external-dependency.com",
				},
				{
					Version:            "1.0.0",
					SourceName:         "test2b-external-dependency",
					DeprecationMessage: "Deprecation message for module test2b-external-dependency",
					Link:               "https://test2b-external-dependency.com",
				},
				{
					Version:            "1.0.0",
					SourceName:         "test3",
					DeprecationMessage: "Deprecation message for module test3",
					Link:               "https://test3.com",
				},
			},
		},
	}
	diagGot := workspaceDeprecations.BuildDeprecationWarning()

	assertResultDeepEqual(t, *diagGot, *diagWant)

}

func TestHasDeprecations_root_module(t *testing.T) {
	workspaceDeprecationsAtRoot := &WorkspaceDeprecationInfo{
		ModuleDeprecationInfos: []*ModuleDeprecationInfo{
			{
				SourceName: "test1",
				RegistryDeprecation: &RegistryModuleDeprecation{
					Version: "1.0.0",
					Link:    "https://test1.com",
					Message: "Deprecation message for module test1",
				},
				ExternalDependencies: []*ModuleDeprecationInfo{},
			},
		},
	}

	if !workspaceDeprecationsAtRoot.HasDeprecations() {
		t.Error("Expected deprecations to be present, but none were found")
	}

	workspaceDeprecationsNoneAtRoot := &WorkspaceDeprecationInfo{
		ModuleDeprecationInfos: []*ModuleDeprecationInfo{
			{
				SourceName:           "test1",
				RegistryDeprecation:  nil,
				ExternalDependencies: []*ModuleDeprecationInfo{},
			},
		},
	}

	if workspaceDeprecationsNoneAtRoot.HasDeprecations() {
		t.Error("Expected no deprecations to be present, but some were found")
	}

}

func TestHasDeprecations_external_dependencies(t *testing.T) {
	workspaceDeprecationsInExternalDependency := &WorkspaceDeprecationInfo{
		ModuleDeprecationInfos: []*ModuleDeprecationInfo{
			{
				SourceName:          "test2",
				RegistryDeprecation: nil,
				ExternalDependencies: []*ModuleDeprecationInfo{
					{
						SourceName:          "test2-external-dependency",
						RegistryDeprecation: nil,
					},
					{
						SourceName: "test2b-external-dependency",
						RegistryDeprecation: &RegistryModuleDeprecation{
							Version: "1.0.0",
							Link:    "https://test2b-external-dependency.com",
							Message: "Deprecation message for module test2b-external-dependency",
						},
					},
				},
			},
		},
	}

	if !workspaceDeprecationsInExternalDependency.HasDeprecations() {
		t.Error("Expected deprecations to be present, but none were found")
	}
}

func TestHasDeprecations_with_none_present(t *testing.T) {
	workspaceDeprecationsNoModules := &WorkspaceDeprecationInfo{
		ModuleDeprecationInfos: []*ModuleDeprecationInfo{},
	}

	if workspaceDeprecationsNoModules.HasDeprecations() {
		t.Error("Expected no deprecations to be present, but some were found")
	}
}
