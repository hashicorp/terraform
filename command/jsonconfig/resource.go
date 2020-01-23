package jsonconfig

// Resource is the representation of a resource in the config
type resource struct {
	// Address is the absolute resource address
	Address string `json:"address,omitempty"`

	// Mode can be "managed" or "data"
	Mode string `json:"mode,omitempty"`

	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`

	// ProviderConfigKey is the key into "provider_configs" (shown above) for
	// the provider configuration that this resource is associated with.
	//
	// NOTE: If a given resource is in a ModuleCall, and the provider was
	// configured outside of the module (in a higher level configuration file),
	// the ProviderConfigKey will not match a key in the ProviderConfigs map.
	ProviderConfigKey string `json:"provider_config_key,omitempty"`

	// Provisioners is an optional field which describes any provisioners.
	// Connection info will not be included here.
	Provisioners []provisioner `json:"provisioners,omitempty"`

	// Expressions" describes the resource-type-specific  content of the
	// configuration block.
	Expressions map[string]interface{} `json:"expressions,omitempty"`

	// SchemaVersion indicates which version of the resource type schema the
	// "values" property conforms to.
	SchemaVersion uint64 `json:"schema_version"`

	// CountExpression and ForEachExpression describe the expressions given for
	// the corresponding meta-arguments in the resource configuration block.
	// These are omitted if the corresponding argument isn't set.
	CountExpression   *expression `json:"count_expression,omitempty"`
	ForEachExpression *expression `json:"for_each_expression,omitempty"`

	DependsOn []string `json:"depends_on,omitempty"`
}

// resources is a slice of resource that implements the Sort interface.
type resources []resource

func (rs resources) Less(i, j int) bool {
	return rs[i].Address < rs[j].Address
}

func (rs resources) Len() int {
	return len(rs)
}

func (rs resources) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}
