// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
)

type WorkspaceDeprecationInfo struct {
	ModuleDeprecationInfos []*ModuleDeprecationInfo
}

type ModuleDeprecationInfo struct {
	SourceName           string
	RegistryDeprecation  *RegistryModuleDeprecation
	ExternalDependencies []*ModuleDeprecationInfo
}

type RegistryModuleDeprecation struct {
	Version string
	Link    string
	Message string
}

type ModuleDeprecationDiagnosticExtra struct {
	MessageCode  string                                             `json:"message_code"`
	Deprecations []*ModuleDeprecationDiagnosticExtraDeprecationItem `json:"deprecations"`
}

type ModuleDeprecationDiagnosticExtraDeprecationItem struct {
	Version            string `json:"version"`
	SourceName         string `json:"source_name"`
	DeprecationMessage string `json:"deprecation_message"`
	Link               string `json:"link"`
}

func (i *WorkspaceDeprecationInfo) HasDeprecations() bool {
	if i == nil || i.ModuleDeprecationInfos == nil {
		return false
	}
	for _, deprecationInfo := range i.ModuleDeprecationInfos {
		if deprecationInfo != nil && deprecationInfo.hasDeprecations() {
			return true
		}
	}
	return false
}

func (i *ModuleDeprecationInfo) hasDeprecations() bool {
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
func (i *WorkspaceDeprecationInfo) BuildDeprecationWarning() *hcl.Diagnostic {
	modDeprecationStrings := []string{}
	var deprecationList []*ModuleDeprecationDiagnosticExtraDeprecationItem
	for _, modDeprecationInfo := range i.ModuleDeprecationInfos {
		if modDeprecationInfo != nil && modDeprecationInfo.RegistryDeprecation != nil {
			modDeprecation := fmt.Sprintf("\x1b[1mVersion %s of %s\x1b[0m", modDeprecationInfo.RegistryDeprecation.Version, modDeprecationInfo.SourceName)
			// Link and Message are optional fields, if unset they are an empty string by default
			if modDeprecationInfo.RegistryDeprecation.Message != "" {
				modDeprecation = modDeprecation + fmt.Sprintf("\n\n%s", modDeprecationInfo.RegistryDeprecation.Message)
			}
			if modDeprecationInfo.RegistryDeprecation.Link != "" {
				modDeprecation = modDeprecation + fmt.Sprintf("\n\nLink for more information: %s", modDeprecationInfo.RegistryDeprecation.Link)
			}
			deprecationList = append(deprecationList, &ModuleDeprecationDiagnosticExtraDeprecationItem{
				Version:            modDeprecationInfo.RegistryDeprecation.Version,
				SourceName:         modDeprecationInfo.SourceName,
				DeprecationMessage: modDeprecationInfo.RegistryDeprecation.Message,
				Link:               modDeprecationInfo.RegistryDeprecation.Link,
			})
			modDeprecationStrings = append(modDeprecationStrings, modDeprecation)
		}
		deprecationStrings, deprecationStructs := buildChildModuleDeprecations(modDeprecationInfo.ExternalDependencies, []string{modDeprecationInfo.SourceName})
		deprecationList = append(deprecationList, deprecationStructs...)
		modDeprecationStrings = append(modDeprecationStrings, deprecationStrings...)
	}
	deprecationsMessage := ""
	for _, deprecationString := range modDeprecationStrings {
		deprecationsMessage += deprecationString + "\n\n"
	}

	return &hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "Deprecated modules found, consider installing updated versions. The following are affected:",
		Detail:   deprecationsMessage,
		Extra: &ModuleDeprecationDiagnosticExtra{
			MessageCode:  "module_deprecation_warning",
			Deprecations: deprecationList,
		},
	}
}

func buildChildModuleDeprecations(modDeprecations []*ModuleDeprecationInfo, parentMods []string) ([]string, []*ModuleDeprecationDiagnosticExtraDeprecationItem) {
	modDeprecationStrings := []string{}
	var deprecationList []*ModuleDeprecationDiagnosticExtraDeprecationItem
	for _, deprecation := range modDeprecations {
		if deprecation.RegistryDeprecation != nil {
			// mdTODO: Add highlighting here, look up other examples where it's present
			modDeprecation := fmt.Sprintf("\x1b[1mVersion %s of %s %s\x1b[0m", deprecation.RegistryDeprecation.Version, deprecation.SourceName, buildModHierarchy(parentMods, deprecation.SourceName))
			// Link and Message are optional fields, if unset they are an empty string by default
			if deprecation.RegistryDeprecation.Message != "" {
				modDeprecation = modDeprecation + fmt.Sprintf("\n\n%s", deprecation.RegistryDeprecation.Message)
			}
			if deprecation.RegistryDeprecation.Link != "" {
				modDeprecation = modDeprecation + fmt.Sprintf("\n\nLink for more information: %s", deprecation.RegistryDeprecation.Link)
			}
			modDeprecationStrings = append(modDeprecationStrings, modDeprecation)
		}
		deprecationList = append(deprecationList, &ModuleDeprecationDiagnosticExtraDeprecationItem{
			Version:            deprecation.RegistryDeprecation.Version,
			SourceName:         deprecation.SourceName,
			DeprecationMessage: deprecation.RegistryDeprecation.Message,
			Link:               deprecation.RegistryDeprecation.Link,
		})
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
