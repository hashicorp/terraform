package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressaddr"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressgen"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressprovider"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
)

// graphSoakCommand implements the "stresstest graph soak" command, which
// repeatedly generates random configuration series and exercises them in
// order to try to find failures.
type graphSoakCommand struct {
}

var _ cli.Command = (*graphSoakCommand)(nil)

func (c *graphSoakCommand) Run(args []string) int {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	seriesAddr := stressaddr.RandomConfigSeries(rnd)

	fmt.Printf("Series %s\n", seriesAddr)

	series := stressgen.GenerateConfigSeries(seriesAddr)

	provider := stressprovider.New()
	providerFactories := map[addrs.Provider]providers.Factory{
		addrs.MustParseProviderSourceString("terraform.io/stresstest/stressful"): func() (providers.Interface, error) {
			return provider.NewInstance(), nil
		},
	}

	log.Printf("Testing series %s", seriesAddr)
	priorState := states.NewState() // Initial state is empty
	for _, config := range series.Steps {
		log.Printf("Testing configuration %s", config.Addr)
		snap := config.ConfigSnapshot()

		/*for k, mod := range snap.Modules {
			fmt.Printf("# %s\n\n%s\n", k, mod.Files["test.tf"])
		}*/
		//return 0
		loader := configload.NewLoaderFromSnapshot(snap)
		cfg, hclDiags := loader.LoadConfig(".")
		if hclDiags.HasErrors() {
			log.Printf("[BUG] Generated an invalid configuration for %s: %s", config.Addr, hclDiags.Error())
			continue
		}

		vars := make(terraform.InputValues)
		for name, val := range config.VariableValues() {
			vars[name] = &terraform.InputValue{
				Value:      val,
				SourceType: terraform.ValueFromCLIArg, // lies, but harmless here
			}
		}

		var plan *plans.Plan
		{
			ctx, diags := terraform.NewContext(&terraform.ContextOpts{
				Config:    cfg,
				Variables: vars,
				Providers: providerFactories,
			})
			if diags.HasErrors() {
				log.Printf("[BUG] Failed to create a terraform.Context for planning %s: %s", config.Addr, diags.Err().Error())
				continue
			}

			plan, diags = ctx.Plan()
			if diags.HasErrors() {
				log.Printf("Series %s failed planning %s: %s", seriesAddr, config.Addr, diags.Err().Error())
				continue
			}
		}

		var newState *states.State
		{
			ctx, diags := terraform.NewContext(&terraform.ContextOpts{
				Config:    cfg,
				Variables: vars,
				Changes:   plan.Changes,
				Providers: providerFactories,
			})
			if diags.HasErrors() {
				log.Printf("[BUG] Failed to create a terraform.Context for applying %s: %s", config.Addr, diags.Err().Error())
				continue
			}

			newState, diags = ctx.Apply()
			if diags.HasErrors() {
				log.Printf("Series %s failed applying %s: %s", seriesAddr, config.Addr, diags.Err().Error())
				continue
			}
		}

		// All of the object instances in the configuration must now agree
		// that the new state matches their expectations.
		for _, err := range config.CheckNewState(priorState, newState) {
			log.Printf("Incorrect state for series %s: %s", seriesAddr, err)
		}

		priorState = newState // next step will use this new state
	}

	return 0
}

func (c *graphSoakCommand) Synopsis() string {
	return "Repeatedly exercise random configurations"
}

func (c *graphSoakCommand) Help() string {
	return strings.TrimSpace(`
Usage: stresstest graph soak

...
`)
}
