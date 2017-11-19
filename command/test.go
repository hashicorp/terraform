package command

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/testharness"
	"github.com/hashicorp/terraform/tfdiags"
)

// TestCommand is a cli.Command implementation that tests the specifications
// for a configuration against one or more test scenarios.
type TestCommand struct {
	Meta
}

func (c *TestCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.flagSet("test")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	scenarioNames := cmdFlags.Args()

	spec, diags := testharness.LoadSpecDir(".")
	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	scenarios := spec.Scenarios()

	if len(scenarioNames) == 0 {
		if len(scenarios) == 0 {
			c.showDiagnostics(fmt.Errorf("The current module has no test scenarios defined"))
			return 1
		}

		scenarioNames = make([]string, 0, len(scenarios))
		for name := range scenarios {
			scenarioNames = append(scenarioNames, name)
		}
		sort.Strings(scenarioNames)
	} else {
		// Ensure that all of the user's given scenario names are valid
		var diags tfdiags.Diagnostics
		for _, name := range scenarioNames {
			if _, defined := scenarios[name]; !defined {
				diags = diags.Append(fmt.Errorf("There is no test scenario named %q", name))
			}
		}
		if diags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	}

	// TODO: Eventually this should have multiple selectable formatters to
	// support JSON output, XUnit output, TAP output, etc, but for now we
	// just have hardcoded Markdown-ish output for prototyping purposes.
	// This Markdown-ish output is intended to be both readable in terminal
	// and do something reasonable when pasted into e.g. a GitHub PR comment.
	for _, scenarioName := range scenarioNames {
		scenario := scenarios[scenarioName]

		fmt.Printf("\n# Scenario %q\n\n", scenarioName)

		// We re-load the configuration each time, even though that shouldn't
		// really be necessary, just because currently parts of the
		// configuration get mutated during work and so there is a small risk
		// that state will "bleed" from one run to the next. Perhaps in future
		// the config loader and core will be reworked to use a more functional
		// style, and then this won't be necessary anymore.
		mod, err := c.Module(".")
		if err != nil {
			err = errwrap.Wrapf("Failed to load root module: {{err}}", err)
			c.showDiagnostics(err)
			continue
		}

		variables := make(map[string]interface{})
		for name, val := range scenario.Variables {
			variables[name] = hcl2shim.ConfigValueFromHCL2(val)
		}

		stateMgr := state.InmemState{}
		hook := &testCommandHook{}
		hooks := []terraform.Hook{hook}

		created := true
		{
			fmt.Print("## Create\n\n")

			startTime := time.Now()

			hook.Reset()
			ctxOpts := &terraform.ContextOpts{
				Module:           mod,
				State:            stateMgr.State(),
				Variables:        variables,
				ProviderResolver: c.providerResolver(),
				Hooks:            hooks,
			}
			ctx, err := terraform.NewContext(ctxOpts)
			if err != nil {
				c.showDiagnostics(err)
				continue
			}

			warns, errs := ctx.Validate()
			for _, warn := range warns {
				c.showDiagnostics(tfdiags.SimpleWarning(warn))
			}
			for _, err := range errs {
				c.showDiagnostics(err)
			}
			if len(errs) > 0 {
				continue
			}

			_, err = ctx.Refresh()
			if err != nil {
				c.showDiagnostics(err)
				continue
			}

			_, err = ctx.Plan()
			if err != nil {
				c.showDiagnostics(err)
				continue
			}

			// TODO: Should run Apply more cautiously, like "terraform apply"
			// does, so that we can cleanly stop (cleaning up any resources we
			// already created) if we get SIGINT.
			_, err = ctx.Apply()
			if err != nil {
				c.showDiagnostics(err)

				created = false // we won't try to verify, but will still destroy
			}

			stateMgr.WriteState(ctx.State())

			endTime := time.Now()

			fmt.Printf("\nTotal created: %d in %s\n\n", hook.ApplyCount(), endTime.Sub(startTime))
		}

		if created {
			fmt.Print("## Verify\n\n")

			subject := testharness.NewSubject(mod, stateMgr.State(), scenario.Variables)

			itemCh := make(chan testharness.CheckItem)
			logCh := make(chan string)
			cs := testharness.NewCheckStream(itemCh, logCh)
			testharness.TestStream(subject, spec, cs)

			logOpen := false
			var successes, failures, errors, skips int
			for {
				select {
				case item, ok := <-itemCh:
					if ok {
						if logOpen {
							// End the open log block before we produce more items
							fmt.Fprint(os.Stderr, "```\n\n")
							os.Stderr.Sync()
							logOpen = false
						}

						check := "[x]"
						if item.Result != testharness.Success {
							check = "[ ]"
						}
						exclam := ""
						switch item.Result {
						case testharness.Success:
							successes++
						case testharness.Failure:
							failures++
						case testharness.Error:
							errors++
							exclam = "**(ERROR)** "
						case testharness.Skipped:
							skips++
							exclam = "(SKIPPED) "
						}
						fmt.Printf("* %s %s%s\n", check, exclam, item.Caption)
					} else {
						itemCh = nil
					}
				case msg, ok := <-logCh:
					if ok {
						if !logOpen {
							os.Stdout.Sync()
							// Open a log block before we print our message
							fmt.Fprint(os.Stderr, "\n```\n")
							logOpen = true
						}
						fmt.Fprintln(os.Stderr, msg)
					} else {
						logCh = nil
					}
				}

				if itemCh == nil && logCh == nil {
					break
				}
			}

			total := successes + failures + skips + errors
			fmt.Printf("\nTotal assertions: %d (%d passed, %d failed, %d skipped, %d errored)\n\n", total, successes, failures, skips, errors)
		}

		{
			fmt.Print("## Destroy\n\n")

			startTime := time.Now()

			hook.Reset()
			ctxOpts := &terraform.ContextOpts{
				Destroy:          true,
				Module:           mod,
				State:            stateMgr.State(),
				Variables:        variables,
				ProviderResolver: c.providerResolver(),
				Hooks:            hooks,
			}
			ctx, err := terraform.NewContext(ctxOpts)
			if err != nil {
				c.showDiagnostics(err)
				continue
			}

			_, err = ctx.Plan()
			if err != nil {
				c.showDiagnostics(err)
				continue
			}

			// TODO: Should run Apply more cautiously, like "terraform apply"
			// does, so that we can cleanly stop (cleaning up any resources we
			// already created) if we get SIGINT.
			_, err = ctx.Apply()
			if err != nil {
				c.showDiagnostics(err)

				// TODO: Dangling resources probably exist here, so we should
				// print out information about them so that users can track
				// them down and destroy them manually.
			}

			stateMgr.WriteState(ctx.State())

			endTime := time.Now()

			fmt.Printf("\nTotal destroyed: %d in %s\n\n", hook.ApplyCount(), endTime.Sub(startTime))
		}
	}

	return 0
}

func (c *TestCommand) Help() string {
	helpText := `
Usage: terraform test [SCENARIOS...]

  Runs the current module's test specifications against the given test
  scenarios.

  This command will use each of the selected scenarios to create a temporary
  set of resources to run the module's test specifications against, and then
  destroy those resources.

  If no scenarios are given, the test specifications are run against all of
  the scenarios declared across all test specification files.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCommand) Synopsis() string {
	return "Run test specifications in test scenarios"
}

type testCommandHook struct {
	startTimes map[string]time.Time
	applyCount int

	terraform.NilHook
}

func (h *testCommandHook) Reset() {
	h.startTimes = make(map[string]time.Time)
	h.applyCount = 0
}

func (h *testCommandHook) ApplyCount() int {
	return h.applyCount
}

func (h *testCommandHook) PreRefresh(info *terraform.InstanceInfo, s *terraform.InstanceState) (terraform.HookAction, error) {
	h.startTimes["r"+info.ResourceAddress().String()] = time.Now()
	return terraform.HookActionContinue, nil
}

func (h *testCommandHook) PostRefresh(info *terraform.InstanceInfo, s *terraform.InstanceState) (terraform.HookAction, error) {
	startTime := h.startTimes["r"+info.ResourceAddress().String()]
	endTime := time.Now()
	delta := endTime.Sub(startTime)
	fmt.Printf("* [x] %s is read (%s)\n", info.ResourceAddress(), delta)
	return terraform.HookActionContinue, nil
}

func (h *testCommandHook) PreApply(info *terraform.InstanceInfo, s *terraform.InstanceState, d *terraform.InstanceDiff) (terraform.HookAction, error) {
	h.startTimes["a"+info.ResourceAddress().String()] = time.Now()
	return terraform.HookActionContinue, nil
}

func (h *testCommandHook) PostApply(info *terraform.InstanceInfo, s *terraform.InstanceState, err error) (terraform.HookAction, error) {
	startTime := h.startTimes["a"+info.ResourceAddress().String()]
	endTime := time.Now()
	delta := endTime.Sub(startTime)
	h.applyCount++

	check := "[x]"
	if err != nil {
		check = "[ ] **(ERROR)**"
	}

	verb := "created"
	if s == nil || s.ID == "" {
		verb = "destroyed"
	}

	fmt.Printf("* %s %s is %s (%s)\n", check, info.ResourceAddress(), verb, delta)

	return terraform.HookActionContinue, nil
}
