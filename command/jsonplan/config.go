package jsonplan

// Config represents the complete configuration source
type config struct {
	ProviderConfigs []providerConfig `json:"provider_config"`
	RootModule      configRootModule `json:"root_module"`
}

// ProviderConfig describes all of the provider configurations throughout the
// configuration tree, flattened into a single map for convenience since
// provider configurations are the one concept in Terraform that can span across
// module boundaries.
type providerConfig struct {
	Name          string
	Alias         string
	ModuleAddress string
	Expressions   expressions
}

type configRootModule struct {
	Outputs     []map[string]output
	Resources   []resource
	ModuleCalls []moduleCall
}

type configOutput struct {
	Sensitive  bool
	Expression expression
}
