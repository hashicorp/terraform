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

		created := true
		{
			fmt.Print("## Create\n\n")

			itemCh := make(chan testharness.CheckItem)
			logCh := make(chan string)
			cs := testharness.NewCheckStream(itemCh, logCh)

			ctxOpts := &terraform.ContextOpts{
				Module:           mod,
				State:            stateMgr.State(),
				Variables:        variables,
				ProviderResolver: c.providerResolver(),
				Hooks: []terraform.Hook{
					newTestCommandHook(cs),
				},
			}
			ctx, err := terraform.NewContext(ctxOpts)
			if err != nil {
				c.showDiagnostics(err)
				continue
			}

			// Our progress-printing code runs in the background and
			// turns the results of our Terraform operations into synthetic
			// test assertions.
			progressDone := make(chan struct{})
			go func() {
				c.showProgress(itemCh, logCh)
				close(progressDone)
			}()

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

			// This causes our progress printer to exit
			cs.Close()
			<-progressDone
		}

		if created {
			fmt.Print("## Verify\n\n")

			subject := testharness.NewSubject(mod, stateMgr.State(), scenario.Variables)

			itemCh := make(chan testharness.CheckItem)
			logCh := make(chan string)
			cs := testharness.NewCheckStream(itemCh, logCh)
			testharness.TestStream(subject, spec, cs)

			c.showProgress(itemCh, logCh) // blocks until the tests are all complete
		}

		{
			fmt.Print("## Destroy\n\n")

			itemCh := make(chan testharness.CheckItem)
			logCh := make(chan string)
			cs := testharness.NewCheckStream(itemCh, logCh)

			ctxOpts := &terraform.ContextOpts{
				Destroy:          true,
				Module:           mod,
				State:            stateMgr.State(),
				Variables:        variables,
				ProviderResolver: c.providerResolver(),
				Hooks: []terraform.Hook{
					newTestCommandHook(cs),
				},
			}
			ctx, err := terraform.NewContext(ctxOpts)
			if err != nil {
				c.showDiagnostics(err)
				continue
			}

			// Our progress-printing code runs in the background and
			// turns the results of our Terraform operations into synthetic
			// test assertions.
			progressDone := make(chan struct{})
			go func() {
				c.showProgress(itemCh, logCh)
				close(progressDone)
			}()

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

			// This causes our progress printer to exit
			cs.Close()
			<-progressDone
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

// showProgress monitors the two given channels and prints the CheckItems
// and log messages that appear, creating Markdown-like formatting.
//
// This function blocks until both channels are closed.
func (c *TestCommand) showProgress(itemCh <-chan testharness.CheckItem, logCh <-chan string) {
	startTime := time.Now()

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

				durStr := ""
				if item.Time != 0 {
					durStr = fmt.Sprintf(" (in %s)", item.Time)
				}

				fmt.Printf("* %s %s%s%s\n", check, exclam, item.Caption, durStr)
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
	endTime := time.Now()
	fmt.Printf("\nTotal assertions: %d (%d passed, %d failed, %d skipped, %d errored) in %s\n\n", total, successes, failures, skips, errors, endTime.Sub(startTime))
	os.Stderr.Sync()
	os.Stdout.Sync()
}

// testCommandHook is a terraform.Hook that writes items to a
// testharness.CheckStream, making a Terraform operation appear as a
// synthetic set of test assertions.
//
// After instantiating a testCommandHook it must be initialized with Reset
// before using it. Reset can be called again later to start fresh with
// a new CheckStream.
type testCommandHook struct {
	startTimes map[string]time.Time
	applyCount int
	cs         testharness.CheckStream

	terraform.NilHook
}

func newTestCommandHook(cs testharness.CheckStream) *testCommandHook {
	return &testCommandHook{
		cs:         cs,
		startTimes: make(map[string]time.Time),
	}
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

	h.cs.Write(testharness.CheckItem{
		Result:  testharness.Success,
		Caption: fmt.Sprintf("%s is read", info.ResourceAddress()),
		Time:    delta,
	})

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

	result := testharness.Success
	if err != nil {
		result = testharness.Error
	}

	verb := "created"
	if s == nil || s.ID == "" {
		verb = "destroyed"
	}

	h.cs.Write(testharness.CheckItem{
		Result:  result,
		Caption: fmt.Sprintf("%s is %s", info.ResourceAddress(), verb),
		Time:    delta,
	})

	return terraform.HookActionContinue, nil
}
