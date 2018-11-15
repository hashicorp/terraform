package jsonplan

// Config represents the complete configuration source
type config struct {
	ProviderConfigs []providerConfig `json:"provider_config,omitempty"`
	RootModule      configRootModule `json:"root_module,omitempty"`
}

// ProviderConfig describes all of the provider configurations throughout the
// configuration tree, flattened into a single map for convenience since
// provider configurations are the one concept in Terraform that can span across
// module boundaries.
type providerConfig struct {
	Name          string      `json:"name,omitempty"`
	Alias         string      `json:"alias,omitempty"`
	ModuleAddress string      `json:"module_address,omitempty"`
	Expressions   expressions `json:"expressions,omitempty"`
}

type configRootModule struct {
	Outputs     []map[string]output `json:"outputs,omitempty"`
	Resources   []resource          `json:"resources,omitempty"`
	ModuleCalls []moduleCall        `json:"module_calls,omitempty"`
}

type configOutput struct {
	Sensitive  bool       `json:"sensitive,omitempty"`
	Expression expression `json:"expression,omitempty"`
}
