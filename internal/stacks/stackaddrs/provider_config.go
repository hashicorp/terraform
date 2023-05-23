package stackaddrs

import (
	"github.com/hashicorp/terraform/internal/addrs"
)

// ProviderConfigRef is a reference-only address type representing a reference
// to a particular provider configuration using its local name, since local
// name is how we refer to providers when they appear in expressions.
//
// The referent of a ProviderConfigRef is a [ProviderConfig], so resolving
// the reference will always require a lookup table from local name to
// fully-qualified provider address.
type ProviderConfigRef struct {
	ProviderLocalName string
	Name              string
}

func (ProviderConfigRef) referenceableSigil() {}

// ProviderConfig is the address of a "provider" block in a stack configuration.
type ProviderConfig struct {
	Provider addrs.Provider
	Name     string
}

func (ProviderConfig) inStackConfigSigil()   {}
func (ProviderConfig) inStackInstanceSigil() {}

// ConfigProviderConfig places a [ProviderConfig] in the context of a particular [Stack].
type ConfigProviderConfig = InStackConfig[ProviderConfig]

// AbsProviderConfig places a [ProviderConfig] in the context of a particular [StackInstance].
type AbsProviderConfig = InStackInstance[ProviderConfig]

// ProviderConfigInstance is the address of a specific provider configuration,
// of which there might potentially be many associated with a given
// [ProviderConfig] if that block uses the "for_each" argument.
type ProviderConfigInstance struct {
	ProviderConfig ProviderConfig
	Key            addrs.InstanceKey
}

func (ProviderConfigInstance) inStackConfigSigil()   {}
func (ProviderConfigInstance) inStackInstanceSigil() {}

// ConfigProviderConfigInstance places a [ProviderConfigInstance] in the context of a particular [Stack].
type ConfigProviderConfigInstance = InStackConfig[ProviderConfigInstance]

// AbsProviderConfigInstance places a [ProviderConfigInstance] in the context of a particular [StackInstance].
type AbsProviderConfigInstance = InStackInstance[ProviderConfigInstance]
