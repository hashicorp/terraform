package addrs

import (
	"fmt"
)

// ProviderConfig is the address of a provider configuration.
type ProviderConfig struct {
	Type string

	// If not empty, Alias identifies which non-default (aliased) provider
	// configuration this address refers to.
	Alias string
}

// Absolute returns an AbsProviderConfig from the receiver and the given module
// instance address.
func (pc ProviderConfig) Absolute(module ModuleInstance) AbsProviderConfig {
	return AbsProviderConfig{
		Module:         module,
		ProviderConfig: pc,
	}
}

func (pc ProviderConfig) String() string {
	if pc.Alias != "" {
		return fmt.Sprintf("provider.%s.%s", pc.Type, pc.Alias)
	}

	return "provider." + pc.Type
}

// AbsProviderConfig is the absolute address of a provider configuration
// within a particular module instance.
type AbsProviderConfig struct {
	Module         ModuleInstance
	ProviderConfig ProviderConfig
}

func (pc AbsProviderConfig) String() string {
	return fmt.Sprintf("%s.%s", pc.Module.String(), pc.ProviderConfig.String())
}
