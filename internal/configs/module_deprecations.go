package configs

import "fmt"

type WorkspaceDeprecations struct {
	ModuleDeprecationInfos []*ModuleDeprecationInfo
}

type ModuleDeprecationInfo struct {
	SourceName           string
	RegistryDeprecation  *RegistryModuleDeprecation
	ExternalDependencies []*ModuleDeprecationInfo
}

type RegistryModuleDeprecation struct {
	Version      string
	ExternalLink string
}

func (i *WorkspaceDeprecations) BuildDeprecationWarningString() string {
	modDeprecationStrings := []string{}
	for _, modDeprecationInfo := range i.ModuleDeprecationInfos {
		if modDeprecationInfo.RegistryDeprecation != nil {
			modDeprecationStrings = append(modDeprecationStrings, fmt.Sprintf("Version %s of \"%s\" \nTo learn more visit: %s\n", modDeprecationInfo.RegistryDeprecation.Version, modDeprecationInfo.SourceName, modDeprecationInfo.RegistryDeprecation.ExternalLink))
		}
		modDeprecationStrings = append(modDeprecationStrings, buildChildDeprecationWarnings(modDeprecationInfo.ExternalDependencies, []string{modDeprecationInfo.SourceName})...)
	}
	deprecationsMessage := ""
	for _, deprecationString := range modDeprecationStrings {
		deprecationsMessage += deprecationString + "\n"
	}

	return deprecationsMessage
}

func buildChildDeprecationWarnings(modDeprecations []*ModuleDeprecationInfo, parentMods []string) []string {
	modDeprecationStrings := []string{}
	for _, deprecation := range modDeprecations {
		if deprecation.RegistryDeprecation != nil {
			modDeprecationStrings = append(modDeprecationStrings, fmt.Sprintf("Version %s of \"%s\" %s \nTo learn more visit: %s\n", deprecation.RegistryDeprecation.Version, deprecation.SourceName, buildModHierarchy(parentMods, deprecation.SourceName), deprecation.RegistryDeprecation.ExternalLink))
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
