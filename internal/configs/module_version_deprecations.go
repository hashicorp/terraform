// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/mitchellh/colorstring"
)

type DirectoryDeprecationInfo struct {
	ModuleVersionDeprecationInfos []*ModuleVersionDeprecationInfo
}

type ModuleVersionDeprecationInfo struct {
	SourceName           string
	RegistryDeprecation  *RegistryModuleVersionDeprecation
	ExternalDependencies []*ModuleVersionDeprecationInfo
}

type RegistryModuleVersionDeprecation struct {
	Version string
	Link    string
	Message string
}

type ModuleVersionDeprecationDiagnosticExtra struct {
	MessageCode  string                                                    `json:"message_code"`
	Deprecations []*ModuleVersionDeprecationDiagnosticExtraDeprecationItem `json:"deprecations"`
}

func (m *ModuleVersionDeprecationDiagnosticExtra) IsPublic() {}

type ModuleVersionDeprecationDiagnosticExtraDeprecationItem struct {
	Version            string `json:"version"`
	SourceName         string `json:"source_name"`
	DeprecationMessage string `json:"deprecation_message"`
	Link               string `json:"link"`
}

func (i *DirectoryDeprecationInfo) HasDeprecations() bool {
	if i == nil || i.ModuleVersionDeprecationInfos == nil {
		return false
	}
	for _, deprecationInfo := range i.ModuleVersionDeprecationInfos {
		if deprecationInfo != nil && deprecationInfo.hasDeprecations() {
			return true
		}
	}
	return false
}

func (i *ModuleVersionDeprecationInfo) hasDeprecations() bool {
	if i.RegistryDeprecation != nil {
		return true
	}
	for _, dependencyDeprecationInfo := range i.ExternalDependencies {
		if dependencyDeprecationInfo != nil && dependencyDeprecationInfo.hasDeprecations() {
			return true
		}
	}
	return false
}

// Deprecation info is placed as an string in the Diagnostic Detail for console view,
// as well as placed in the Diagnostic Extra for parsing for the SRO view in HCP Terraform
func (i *DirectoryDeprecationInfo) BuildDeprecationWarning() *hcl.Diagnostic {
	modDeprecations := []string{}
	color := colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: false,
		Reset:   true,
	}
	deprecationList := make([]*ModuleVersionDeprecationDiagnosticExtraDeprecationItem, 0, len(i.ModuleVersionDeprecationInfos))
	for _, modDeprecationInfo := range i.ModuleVersionDeprecationInfos {
		if modDeprecationInfo != nil && modDeprecationInfo.RegistryDeprecation != nil {
			msg := color.Color("[reset][bold]Version %s of %s[reset]")
			modDeprecation := fmt.Sprintf(msg, modDeprecationInfo.RegistryDeprecation.Version, modDeprecationInfo.SourceName)
			// Link and Message are optional fields, if unset they are an empty string by default
			if modDeprecationInfo.RegistryDeprecation.Message != "" {
				modDeprecation = modDeprecation + fmt.Sprintf("\n\n%s", modDeprecationInfo.RegistryDeprecation.Message)
			}
			if modDeprecationInfo.RegistryDeprecation.Link != "" {
				modDeprecation = modDeprecation + fmt.Sprintf("\n\nLink for more information: %s", modDeprecationInfo.RegistryDeprecation.Link)
			}
			deprecationList = append(deprecationList, &ModuleVersionDeprecationDiagnosticExtraDeprecationItem{
				Version:            modDeprecationInfo.RegistryDeprecation.Version,
				SourceName:         modDeprecationInfo.SourceName,
				DeprecationMessage: modDeprecationInfo.RegistryDeprecation.Message,
				Link:               modDeprecationInfo.RegistryDeprecation.Link,
			})
			modDeprecations = append(modDeprecations, modDeprecation)
		}
		deprecationStrings, deprecationStructs := buildChildModuleDeprecations(modDeprecationInfo.ExternalDependencies, []string{modDeprecationInfo.SourceName})
		deprecationList = append(deprecationList, deprecationStructs...)
		modDeprecations = append(modDeprecations, deprecationStrings...)
	}
	deprecationsMessage := strings.Join(modDeprecations, "\n\n")

	return &hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Deprecated module versions found, consider installing updated versions. The following are affected:",
		Detail:   deprecationsMessage,
		Extra: &ModuleVersionDeprecationDiagnosticExtra{
			MessageCode:  "module_deprecation_warning",
			Deprecations: deprecationList,
		},
	}
}

func buildChildModuleDeprecations(modDeprecations []*ModuleVersionDeprecationInfo, parentMods []string) ([]string, []*ModuleVersionDeprecationDiagnosticExtraDeprecationItem) {
	color := colorstring.Colorize{
		Colors:  colorstring.DefaultColors,
		Disable: false,
		Reset:   true,
	}
	modDeprecationStrings := []string{}
	var deprecationList []*ModuleVersionDeprecationDiagnosticExtraDeprecationItem
	for _, deprecation := range modDeprecations {
		if deprecation.RegistryDeprecation != nil {
			msg := color.Color("[reset][bold]Version %s of %s %s[reset]")
			modDeprecation := fmt.Sprintf(msg, deprecation.RegistryDeprecation.Version, deprecation.SourceName, buildModHierarchy(parentMods, deprecation.SourceName))
			// Link and Message are optional fields, if unset they are an empty string by default
			if deprecation.RegistryDeprecation.Message != "" {
				modDeprecation = modDeprecation + fmt.Sprintf("\n\n%s", deprecation.RegistryDeprecation.Message)
			}
			if deprecation.RegistryDeprecation.Link != "" {
				modDeprecation = modDeprecation + fmt.Sprintf("\n\nLink for more information: %s", deprecation.RegistryDeprecation.Link)
			}
			modDeprecationStrings = append(modDeprecationStrings, modDeprecation)
			deprecationList = append(deprecationList, &ModuleVersionDeprecationDiagnosticExtraDeprecationItem{
				Version:            deprecation.RegistryDeprecation.Version,
				SourceName:         deprecation.SourceName,
				DeprecationMessage: deprecation.RegistryDeprecation.Message,
				Link:               deprecation.RegistryDeprecation.Link,
			})
		}
		newParentMods := append(parentMods, deprecation.SourceName)
		deprecationStrings, deprecationStructs := buildChildModuleDeprecations(deprecation.ExternalDependencies, newParentMods)
		modDeprecationStrings = append(modDeprecationStrings, deprecationStrings...)
		deprecationList = append(deprecationList, deprecationStructs...)
	}
	return modDeprecationStrings, deprecationList
}

func buildModHierarchy(parentMods []string, modName string) string {
	heirarchy := ""
	for _, parent := range parentMods {
		heirarchy += fmt.Sprintf("%s -> ", parent)
	}
	heirarchy += modName
	return fmt.Sprintf("(Root: %s)", heirarchy)
}
