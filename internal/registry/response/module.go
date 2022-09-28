package response

import (
	"time"
)

// Module is the response structure with the data for a single module version.
type Module struct {
	ID string `json:"id"`

	//---------------------------------------------------------------
	// Metadata about the overall module.

	Owner       string    `json:"owner"`
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Provider    string    `json:"provider"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	Downloads   int       `json:"downloads"`
	Verified    bool      `json:"verified"`
}

// ModuleDetail represents a module in full detail.
type ModuleDetail struct {
	Module

	//---------------------------------------------------------------
	// Metadata about the overall module. This is only available when
	// requesting the specific module (not in list responses).

	// Root is the root module.
	Root *ModuleSubmodule `json:"root"`

	// Submodules are the other submodules that are available within
	// this module.
	Submodules []*ModuleSubmodule `json:"submodules"`

	//---------------------------------------------------------------
	// The fields below are only set when requesting this specific
	// module. They are available to easily know all available versions
	// and providers without multiple API calls.

	Providers []string `json:"providers"` // All available providers
	Versions  []string `json:"versions"`  // All versions
}

// ModuleSubmodule is the metadata about a specific submodule within
// a module. This includes the root module as a special case.
type ModuleSubmodule struct {
	Path   string `json:"path"`
	Readme string `json:"readme"`
	Empty  bool   `json:"empty"`

	Inputs       []*ModuleInput    `json:"inputs"`
	Outputs      []*ModuleOutput   `json:"outputs"`
	Dependencies []*ModuleDep      `json:"dependencies"`
	Resources    []*ModuleResource `json:"resources"`
}

// ModuleInput is an input for a module.
type ModuleInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Default     string `json:"default"`
}

// ModuleOutput is an output for a module.
type ModuleOutput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ModuleDep is an output for a module.
type ModuleDep struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	Version string `json:"version"`
}

// ModuleProviderDep is the output for a provider dependency
type ModuleProviderDep struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ModuleResource is an output for a module.
type ModuleResource struct {
	Name string `json:"name"`
	Type string `json:"type"`
}
