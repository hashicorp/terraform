package terraform

import (
	"context"
	"log"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
)

// Input asks for input to fill unset required arguments in provider
// configurations.
//
// This modifies the configuration in-place, so asking for Input twice
// may result in different UI output showing different current values.
func (c *Context) Input(mode InputMode) tfdiags.Diagnostics {
	// This function used to be responsible for more than it is now, so its
	// interface is more general than its current functionality requires.
	// It now exists only to handle interactive prompts for provider
	// configurations, with other prompts the responsibility of the CLI
	// layer prior to calling in to this package.
	//
	// (Hopefully in future the remaining functionality here can move to the
	// CLI layer too in order to avoid this odd situation where core code
	// produces UI input prompts.)

	var diags tfdiags.Diagnostics
	defer c.acquireRun("input")()

	if c.uiInput == nil {
		log.Printf("[TRACE] Context.Input: uiInput is nil, so skipping")
		return diags
	}

	ctx := context.Background()

	if mode&InputModeProvider != 0 {
		log.Printf("[TRACE] Context.Input: Prompting for provider arguments")

		// We prompt for input only for provider configurations defined in
		// the root module. At the time of writing that is an arbitrary
		// restriction, but we have future plans to support "count" and
		// "for_each" on modules that will then prevent us from supporting
		// input for child module configurations anyway (since we'd need to
		// dynamic-expand first), and provider configurations in child modules
		// are not recommended since v0.11 anyway, so this restriction allows
		// us to keep this relatively simple without significant hardship.

		pcs := make(map[string]*configs.Provider)
		pas := make(map[string]addrs.ProviderConfig)
		for _, pc := range c.config.Module.ProviderConfigs {
			addr := pc.Addr()
			pcs[addr.String()] = pc
			pas[addr.String()] = addr
			log.Printf("[TRACE] Context.Input: Provider %s declared at %s", addr, pc.DeclRange)
		}
		// We also need to detect _implied_ provider configs from resources.
		// These won't have *configs.Provider objects, but they will still
		// exist in the map and we'll just treat them as empty below.
		for _, rc := range c.config.Module.ManagedResources {
			pa := rc.ProviderConfigAddr()
			if pa.Alias != "" {
				continue // alias configurations cannot be implied
			}
			if _, exists := pcs[pa.String()]; !exists {
				pcs[pa.String()] = nil
				pas[pa.String()] = pa
				log.Printf("[TRACE] Context.Input: Provider %s implied by resource block at %s", pa, rc.DeclRange)
			}
		}
		for _, rc := range c.config.Module.DataResources {
			pa := rc.ProviderConfigAddr()
			if pa.Alias != "" {
				continue // alias configurations cannot be implied
			}
			if _, exists := pcs[pa.String()]; !exists {
				pcs[pa.String()] = nil
				pas[pa.String()] = pa
				log.Printf("[TRACE] Context.Input: Provider %s implied by data block at %s", pa, rc.DeclRange)
			}
		}

		for pk, pa := range pas {
			pc := pcs[pk] // will be nil if this is an implied config

			// Wrap the input into a namespace
			input := &PrefixUIInput{
				IdPrefix:    pk,
				QueryPrefix: pk + ".",
				UIInput:     c.uiInput,
			}

			schema := c.schemas.ProviderConfig(pa.Type.LegacyString())
			if schema == nil {
				// Could either be an incorrect config or just an incomplete
				// mock in tests. We'll let a later pass decide, and just
				// ignore this for the purposes of gathering input.
				log.Printf("[TRACE] Context.Input: No schema available for provider type %q", pa.Type)
				continue
			}

			// For our purposes here we just want to detect if attrbutes are
			// set in config at all, so rather than doing a full decode
			// (which would require us to prepare an evalcontext, etc) we'll
			// use the low-level HCL API to process only the top-level
			// structure.
			var attrExprs hcl.Attributes // nil if there is no config
			if pc != nil && pc.Config != nil {
				lowLevelSchema := schemaForInputSniffing(hcldec.ImpliedSchema(schema.DecoderSpec()))
				content, _, diags := pc.Config.PartialContent(lowLevelSchema)
				if diags.HasErrors() {
					log.Printf("[TRACE] Context.Input: %s has decode error, so ignoring: %s", pa, diags.Error())
					continue
				}
				attrExprs = content.Attributes
			}

			keys := make([]string, 0, len(schema.Attributes))
			for key := range schema.Attributes {
				keys = append(keys, key)
			}
			sort.Strings(keys)

			vals := map[string]cty.Value{}
			for _, key := range keys {
				attrS := schema.Attributes[key]
				if attrS.Optional {
					continue
				}
				if attrExprs != nil {
					if _, exists := attrExprs[key]; exists {
						continue
					}
				}
				if !attrS.Type.Equals(cty.String) {
					continue
				}

				log.Printf("[TRACE] Context.Input: Prompting for %s argument %s", pa, key)
				rawVal, err := input.Input(ctx, &InputOpts{
					Id:          key,
					Query:       key,
					Description: attrS.Description,
				})
				if err != nil {
					log.Printf("[TRACE] Context.Input: Failed to prompt for %s argument %s: %s", pa, key, err)
					continue
				}

				vals[key] = cty.StringVal(rawVal)
			}

			c.providerInputConfig[pk] = vals
			log.Printf("[TRACE] Context.Input: Input for %s: %#v", pk, vals)
		}
	}

	return diags
}

// schemaForInputSniffing returns a transformed version of a given schema
// that marks all attributes as optional, which the Context.Input method can
// use to detect whether a required argument is set without missing arguments
// themselves generating errors.
func schemaForInputSniffing(schema *hcl.BodySchema) *hcl.BodySchema {
	ret := &hcl.BodySchema{
		Attributes: make([]hcl.AttributeSchema, len(schema.Attributes)),
		Blocks:     schema.Blocks,
	}

	for i, attrS := range schema.Attributes {
		ret.Attributes[i] = attrS
		ret.Attributes[i].Required = false
	}

	return ret
}
