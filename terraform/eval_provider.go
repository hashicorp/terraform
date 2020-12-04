package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/providers"
)

func buildProviderConfig(ctx EvalContext, addr addrs.AbsProviderConfig, config *configs.Provider) hcl.Body {
	var configBody hcl.Body
	if config != nil {
		configBody = config.Config
	}

	var inputBody hcl.Body
	inputConfig := ctx.ProviderInput(addr)
	if len(inputConfig) > 0 {
		inputBody = configs.SynthBody("<input-prompt>", inputConfig)
	}

	switch {
	case configBody != nil && inputBody != nil:
		log.Printf("[TRACE] buildProviderConfig for %s: merging explicit config and input", addr)
		// Note that the inputBody is the _base_ here, because configs.MergeBodies
		// expects the base have all of the required fields, while these are
		// forced to be optional for the override. The input process should
		// guarantee that we have a value for each of the required arguments and
		// that in practice the sets of attributes in each body will be
		// disjoint.
		return configs.MergeBodies(inputBody, configBody)
	case configBody != nil:
		log.Printf("[TRACE] buildProviderConfig for %s: using explicit config only", addr)
		return configBody
	case inputBody != nil:
		log.Printf("[TRACE] buildProviderConfig for %s: using input only", addr)
		return inputBody
	default:
		log.Printf("[TRACE] buildProviderConfig for %s: no configuration at all", addr)
		return hcl.EmptyBody()
	}
}

// GetProvider returns the providers.Interface and schema for a given provider.
func GetProvider(ctx EvalContext, addr addrs.AbsProviderConfig) (providers.Interface, *ProviderSchema, error) {
	if addr.Provider.Type == "" {
		// Should never happen
		panic("GetProvider used with uninitialized provider configuration address")
	}
	provider := ctx.Provider(addr)
	if provider == nil {
		return nil, &ProviderSchema{}, fmt.Errorf("provider %s not initialized", addr)
	}
	// Not all callers require a schema, so we will leave checking for a nil
	// schema to the callers.
	schema := ctx.ProviderSchema(addr)
	return provider, schema, nil
}
