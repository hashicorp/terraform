package terraform

import (
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
)

// Input asks for input to fill variables and provider configurations.
// This modifies the configuration in-place, so asking for Input twice
// may result in different UI output showing different current values.
func (c *Context) Input(mode InputMode) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	defer c.acquireRun("input")()

	if c.uiInput == nil {
		log.Printf("[TRACE] Context.Input: uiInput is nil, so skipping")
		return diags
	}

	if mode&InputModeVar != 0 {
		log.Printf("[TRACE] Context.Input: Prompting for variables")

		// Walk the variables first for the root module. We walk them in
		// alphabetical order for UX reasons.
		configs := c.config.Module.Variables
		names := make([]string, 0, len(configs))
		for name := range configs {
			names = append(names, name)
		}
		sort.Strings(names)
	Variables:
		for _, n := range names {
			v := configs[n]

			// If we only care about unset variables, then we should set any
			// variable that is already set.
			if mode&InputModeVarUnset != 0 {
				if _, isSet := c.variables[n]; isSet {
					continue
				}
			}

			// this should only happen during tests
			if c.uiInput == nil {
				log.Println("[WARN] Context.uiInput is nil during input walk")
				continue
			}

			// Ask the user for a value for this variable
			var rawValue string
			retry := 0
			for {
				var err error
				rawValue, err = c.uiInput.Input(&InputOpts{
					Id:          fmt.Sprintf("var.%s", n),
					Query:       fmt.Sprintf("var.%s", n),
					Description: v.Description,
				})
				if err != nil {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Failed to request interactive input",
						fmt.Sprintf("Terraform attempted to request a value for var.%s interactively, but encountered an error: %s.", n, err),
					))
					return diags
				}

				if rawValue == "" && v.Default == cty.NilVal {
					// Redo if it is required, but abort if we keep getting
					// blank entries
					if retry > 2 {
						diags = diags.Append(tfdiags.Sourceless(
							tfdiags.Error,
							"Required variable not assigned",
							fmt.Sprintf("The variable %q is required, so Terraform cannot proceed without a defined value for it.", n),
						))
						continue Variables
					}
					retry++
					continue
				}

				break
			}

			val, valDiags := v.ParsingMode.Parse(n, rawValue)
			diags = diags.Append(valDiags)
			if diags.HasErrors() {
				continue
			}

			c.variables[n] = &InputValue{
				Value:      val,
				SourceType: ValueFromInput,
			}
		}
	}

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

			schema := c.schemas.ProviderConfig(pa.Type)
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
				rawVal, err := input.Input(&InputOpts{
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
