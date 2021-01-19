package stressrun

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressaddr"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressgen"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressprovider"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// RunSeries tries to plan and apply the series of configuration steps
// associated with the given series, returning a Log describing the outcome.
func RunSeries(addr stressaddr.ConfigSeries) Log {
	var log Log

	series := stressgen.GenerateConfigSeries(addr)
	provider := stressprovider.New()
	providerFactories := map[addrs.Provider]providers.Factory{
		addrs.MustParseProviderSourceString("terraform.io/stresstest/stressful"): func() (providers.Interface, error) {
			return provider.NewInstance(), nil
		},
	}
	remainingSteps := series.Steps
	priorState := states.NewState() // Initial state is empty

	for len(remainingSteps) > 0 {
		var generatedConfig *stressgen.Config
		generatedConfig, remainingSteps = remainingSteps[0], remainingSteps[1:]

		logStep := func(status LogStatus, message string) LogStep {
			return LogStep{
				Time:          time.Now(),
				Message:       message,
				Status:        status,
				Config:        generatedConfig,
				StateSnapshot: priorState.DeepCopy(),
				RemoteObjects: provider.CurrentObjects(),
			}
		}

		// To avoid the need to write configuration to disk, we (ab?)use the
		// same "snapshot" mechanism we use to read configuration out of a
		// saved plan file.
		snap := generatedConfig.ConfigSnapshot()
		loader := configload.NewLoaderFromSnapshot(snap)
		cfg, hclDiags := loader.LoadConfig(".")
		if hclDiags.HasErrors() {
			var diags tfdiags.Diagnostics
			diags = diags.Append(hclDiags)
			log = append(log, logStep(StepFailed, fmt.Sprintf(
				"Generated invalid configuration for %s\n\n%s",
				generatedConfig.Addr, renderDiagnostics(diags, loader.Sources()),
			)))
			break
		}

		vars := make(terraform.InputValues)
		for name, val := range generatedConfig.VariableValues() {
			vars[name] = &terraform.InputValue{
				Value:      val,
				SourceType: terraform.ValueFromCLIArg, // lies, but harmless here
			}
		}

		// First we need to create a plan. From the perspective of our logs,
		// this is also where we show the configuration having changed between
		// steps.
		ctx, diags := terraform.NewContext(&terraform.ContextOpts{
			Config:    cfg,
			Variables: vars,
			Providers: providerFactories,
		})
		if diags.HasErrors() {
			log = append(log, logStep(StepFailed, fmt.Sprintf(
				"Failed to create terraform.Context to plan %s\n\n%s",
				generatedConfig.Addr, renderDiagnostics(diags, loader.Sources()),
			)))
			break
		}

		plan, diags := ctx.Plan()
		if diags.HasErrors() {
			log = append(log, logStep(StepFailed, fmt.Sprintf(
				"Planning failed for %s\n\n%s",
				generatedConfig.Addr, renderDiagnostics(diags, loader.Sources()),
			)))
			continue
		}

		log = append(log, logStep(StepSucceeded, fmt.Sprintf(
			// TODO: It'd be nice to include a rendered plan in here too,
			// so the person reading the logs doesn't need to guess.
			"Created a plan for %s",
			generatedConfig.Addr,
		)))

		// No we'll apply the plan we created, which if successful appears in
		// the logs as a step with the same configuration but potentially a
		// new state and new fake remote objects.
		ctx, diags = terraform.NewContext(&terraform.ContextOpts{
			Config:    cfg,
			Variables: vars,
			Changes:   plan.Changes,
			Providers: providerFactories,
		})
		if diags.HasErrors() {
			log = append(log, logStep(StepFailed, fmt.Sprintf(
				"Failed to create terraform.Context to apply %s\n\n%s",
				generatedConfig.Addr, renderDiagnostics(diags, loader.Sources()),
			)))
			break
		}

		newState, diags := ctx.Apply()
		priorState = newState // all of our remaining logs will refer to the new state
		if diags.HasErrors() {
			log = append(log, logStep(StepFailed, fmt.Sprintf(
				"Apply failed for %s\n\n%s",
				generatedConfig.Addr, renderDiagnostics(diags, loader.Sources()),
			)))
			break
		}

		log = append(log, logStep(StepSucceeded, fmt.Sprintf(
			"Applied the plan for %s",
			generatedConfig.Addr,
		)))

		priorState = newState
	}

	// If there are any steps left in remainingSteps then they were evidently
	// blocked by an earlier failure, so we'll record them in the log as
	// blocked to give the caller a complete record.
	for _, config := range remainingSteps {
		log = append(log, LogStep{
			Time:    time.Now(),
			Message: fmt.Sprintf("Did not reach %s due to an earlier error", config.Addr),
			Status:  StepBlocked,
			Config:  config,

			// We include the state and remote objects from the most
			// recently-completed step, just so that callers don't have to
			// treat the "blocked" status as special when processing logs.
			// These blocked steps didn't run, so it's reasonable to say that
			// the state and remote objects were unchanged.
			StateSnapshot: priorState.DeepCopy(),
			RemoteObjects: provider.CurrentObjects(),
		})
	}

	return log
}
