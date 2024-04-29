package configs

import "fmt"

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

func (i *WorkspaceDeprecationInfo) HasDeprecations() bool {
	for _, deprecationInfo := range i.ModuleDeprecationInfos {
		if deprecationInfo.hasDeprecations() {
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
		if dependencyDeprecationInfo.hasDeprecations() {
			return true
		}
	}
	return false
}

func (i *WorkspaceDeprecationInfo) BuildDeprecationWarningString() string {
	modDeprecationStrings := []string{}
	for _, modDeprecationInfo := range i.ModuleDeprecationInfos {
		if modDeprecationInfo != nil && modDeprecationInfo.RegistryDeprecation != nil {
			// mdTODO: Add highlighting here, look up other examples where it's present
			modDeprecation := fmt.Sprintf("\x1b[1mVersion %s of %s\x1b[0m", modDeprecationInfo.RegistryDeprecation.Version, modDeprecationInfo.SourceName)
			// Link and Message are optional fields, if unset they are an empty string by default
			if modDeprecationInfo.RegistryDeprecation.Message != "" {
				modDeprecation = modDeprecation + fmt.Sprintf("\n\n%s", modDeprecationInfo.RegistryDeprecation.Message)
			}
			if modDeprecationInfo.RegistryDeprecation.Link != "" {
				modDeprecation = modDeprecation + fmt.Sprintf("\n\nLink for more information: %s", modDeprecationInfo.RegistryDeprecation.Link)
			}
			modDeprecationStrings = append(modDeprecationStrings, modDeprecation)
		}
		modDeprecationStrings = append(modDeprecationStrings, buildChildDeprecationWarnings(modDeprecationInfo.ExternalDependencies, []string{modDeprecationInfo.SourceName})...)
	}
	deprecationsMessage := ""
	for _, deprecationString := range modDeprecationStrings {
		deprecationsMessage += deprecationString + "\n\n"
	}

	return deprecationsMessage
}

func buildChildDeprecationWarnings(modDeprecations []*ModuleDeprecationInfo, parentMods []string) []string {
	modDeprecationStrings := []string{}
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
		newParentMods := append(parentMods, deprecation.SourceName)
		modDeprecationStrings = append(modDeprecationStrings, buildChildDeprecationWarnings(deprecation.ExternalDependencies, newParentMods)...)
	}
	return modDeprecationStrings
}

func buildModHierarchy(parentMods []string, modName string) string {
	heirarchy := ""
	for _, parent := range parentMods {
		heirarchy += fmt.Sprintf("%s -> ", parent)
	}
	heirarchy += modName
	return fmt.Sprintf("(Root: %s)", heirarchy)
}
