package command

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providercache"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestCommand is the implementation of "terraform test".
type TestCommand struct {
	Meta
}

func (c *TestCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	args, diags := arguments.ParseTest(rawArgs)
	view := views.NewTest(c.View, args.Output)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Warning,
		`The "terraform test" command is experimental`,
		"We'd like to invite adventurous module authors to write integration tests for their modules using this command, but all of the behaviors of this command are currently experimental and may change based on feedback.\n\nFor more information on the testing experiment, including ongoing research goals and avenues for feedback, see:\n    https://www.terraform.io/docs/language/modules/testing-experiment.html",
	))

	ctx, cancel := c.InterruptibleContext()
	defer cancel()

	results, moreDiags := c.run(ctx, args)
	diags = diags.Append(moreDiags)

	initFailed := diags.HasErrors()
	view.Diagnostics(diags)
	diags = view.Results(results)
	resultsFailed := diags.HasErrors()
	view.Diagnostics(diags) // possible additional errors from saving the results

	var testsFailed bool
	for _, suite := range results {
		for _, component := range suite.Components {
			for _, assertion := range component.Assertions {
				if !assertion.Outcome.SuiteCanPass() {
					testsFailed = true
				}
			}
		}
	}

	// Lots of things can possibly have failed
	if initFailed || resultsFailed || testsFailed {
		return 1
	}
	return 0
}

func (c *TestCommand) run(ctx context.Context, args arguments.Test) (results map[string]*moduletest.Suite, diags tfdiags.Diagnostics) {
	suiteNames, err := c.collectSuiteNames()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error while searching for test configurations",
			fmt.Sprintf("While attempting to scan the 'tests' subdirectory for potential test configurations, Terraform encountered an error: %s.", err),
		))
		return nil, diags
	}

	ret := make(map[string]*moduletest.Suite, len(suiteNames))
	for _, suiteName := range suiteNames {
		if ctx.Err() != nil {
			// If the context has already failed in some way then we'll
			// halt early and report whatever's already happened.
			break
		}
		suite, moreDiags := c.runSuite(ctx, suiteName)
		diags = diags.Append(moreDiags)
		ret[suiteName] = suite
	}

	return ret, diags
}

func (c *TestCommand) runSuite(ctx context.Context, suiteName string) (*moduletest.Suite, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := moduletest.Suite{
		Name:       suiteName,
		Components: map[string]*moduletest.Component{},
	}

	// In order to make this initial round of "terraform test" pretty self
	// contained while it's experimental, it's largely just mimicking what
	// would happen when running the main Terraform workflow commands, which
	// comes at the expense of a few irritants that we'll hopefully resolve
	// in future iterations as the design solidifies:
	// - We need to install remote modules separately for each of the
	//   test suites, because we don't have any sense of a shared cache
	//   of modules that multiple configurations can refer to at once.
	// - We _do_ have a sense of a cache of remote providers, but it's fixed
	//   at being specifically a two-level cache (global vs. directory-specific)
	//   and so we can't easily capture a third level of "all of the test suites
	//   for this module" that sits between the two. Consequently, we need to
	//   dynamically choose between creating a directory-specific "global"
	//   cache or using the user's existing global cache, to avoid any
	//   situation were we'd be re-downloading the same providers for every
	//   one of the test suites.
	// - We need to do something a bit horrid in order to have our test
	//   provider instance persist between the plan and apply steps, because
	//   normally that is the exact opposite of what we want.
	// The above notes are here mainly as an aid to someone who might be
	// planning a subsequent phase of this R&D effort, to help distinguish
	// between things we're doing here because they are valuable vs. things
	// we're doing just to make it work without doing any disruptive
	// refactoring.

	suiteDirs, moreDiags := c.prepareSuiteDir(ctx, suiteName)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		// Generate a special failure representing the test initialization
		// having failed, since we therefore won'tbe able to run the actual
		// tests defined inside.
		ret.Components["(init)"] = &moduletest.Component{
			Assertions: map[string]*moduletest.Assertion{
				"(init)": {
					Outcome:     moduletest.Error,
					Description: "terraform init",
					Message:     "failed to install test suite dependencies",
					Diagnostics: diags,
				},
			},
		}
		return &ret, nil
	}

	// When we run the suite itself, we collect up diagnostics associated
	// with individual components, so ret.Components may or may not contain
	// failed/errored components after runTestSuite returns.
	var finalState *states.State
	ret.Components, finalState = c.runTestSuite(ctx, suiteDirs)

	// Regardless of the success or failure of the test suite, if there are
	// any objects left in the state then we'll generate a top-level error
	// about each one to minimize the chance of the user failing to notice
	// that there are leftover objects that might continue to cost money
	// unless manually deleted.
	for _, ms := range finalState.Modules {
		for _, rs := range ms.Resources {
			for instanceKey, is := range rs.Instances {
				var objs []*states.ResourceInstanceObjectSrc
				if is.Current != nil {
					objs = append(objs, is.Current)
				}
				for _, obj := range is.Deposed {
					objs = append(objs, obj)
				}
				for _, obj := range objs {
					// Unfortunately we don't have provider schemas out here
					// and so we're limited in what we can achieve with these
					// ResourceInstanceObjectSrc values, but we can try some
					// heuristicy things to try to give some useful information
					// in common cases.
					var k, v string
					if ty, err := ctyjson.ImpliedType(obj.AttrsJSON); err == nil {
						if approxV, err := ctyjson.Unmarshal(obj.AttrsJSON, ty); err == nil {
							k, v = format.ObjectValueIDOrName(approxV)
						}
					}

					var detail string
					if k != "" {
						// We can be more specific if we were able to infer
						// an identifying attribute for this object.
						detail = fmt.Sprintf(
							"Due to errors during destroy, test suite %q has left behind an object for %s, with the following identity:\n    %s = %q\n\nYou will need to delete this object manually in the remote system, or else it may have an ongoing cost.",
							suiteName,
							rs.Addr.Instance(instanceKey),
							k, v,
						)
					} else {
						// If our heuristics for finding a suitable identifier
						// failed then unfortunately we must be more vague.
						// (We can't just print the entire object, because it
						// might be overly large and it might contain sensitive
						// values.)
						detail = fmt.Sprintf(
							"Due to errors during destroy, test suite %q has left behind an object for %s. You will need to delete this object manually in the remote system, or else it may have an ongoing cost.",
							suiteName,
							rs.Addr.Instance(instanceKey),
						)
					}
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Error,
						"Failed to clean up after tests",
						detail,
					))
				}
			}
		}
	}

	return &ret, diags
}

func (c *TestCommand) prepareSuiteDir(ctx context.Context, suiteName string) (testCommandSuiteDirs, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	configDir := filepath.Join("tests", suiteName)
	log.Printf("[TRACE] terraform test: Prepare directory for suite %q in %s", suiteName, configDir)

	suiteDirs := testCommandSuiteDirs{
		SuiteName: suiteName,
		ConfigDir: configDir,
	}

	// Before we can run a test suite we need to make sure that we have all of
	// its dependencies available, so the following is essentially an
	// abbreviated form of what happens during "terraform init", with some
	// extra trickery in places.

	// First, module installation. This will include linking in the module
	// under test, but also includes grabbing the dependencies of that module
	// if it has any.
	suiteDirs.ModulesDir = filepath.Join(configDir, ".terraform", "modules")
	os.MkdirAll(suiteDirs.ModulesDir, 0755) // if this fails then we'll ignore it and let InstallModules below fail instead
	reg := c.registryClient()
	moduleInst := initwd.NewModuleInstaller(suiteDirs.ModulesDir, reg)
	_, moreDiags := moduleInst.InstallModules(ctx, configDir, true, nil)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return suiteDirs, diags
	}

	// The installer puts the files in a suitable place on disk, but we
	// still need to actually load the configuration. We need to do this
	// with a separate config loader because the Meta.configLoader instance
	// is intended for interacting with the current working directory, not
	// with the test suite subdirectories.
	loader, err := configload.NewLoader(&configload.Config{
		ModulesDir: suiteDirs.ModulesDir,
		Services:   c.Services,
	})
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to create test configuration loader",
			fmt.Sprintf("Failed to prepare loader for test configuration %s: %s.", configDir, err),
		))
		return suiteDirs, diags
	}
	cfg, hclDiags := loader.LoadConfig(configDir)
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return suiteDirs, diags
	}
	suiteDirs.Config = cfg

	// With the full configuration tree available, we can now install
	// the necessary providers. We'll use a separate local cache directory
	// here, because the test configuration might have additional requirements
	// compared to the module itself.
	suiteDirs.ProvidersDir = filepath.Join(configDir, ".terraform", "providers")
	os.MkdirAll(suiteDirs.ProvidersDir, 0755) // if this fails then we'll ignore it and operations below fail instead
	localCacheDir := providercache.NewDir(suiteDirs.ProvidersDir)
	providerInst := c.providerInstaller().Clone(localCacheDir)
	if !providerInst.HasGlobalCacheDir() {
		// If the user already configured a global cache directory then we'll
		// just use it for caching the test providers too, because then we
		// can potentially reuse cache entries they already have. However,
		// if they didn't configure one then we'll still establish one locally
		// in the working directory, which we'll then share across all tests
		// to avoid downloading the same providers repeatedly.
		cachePath := filepath.Join(c.DataDir(), "testing-providers") // note this is _not_ under the suite dir
		err := os.MkdirAll(cachePath, 0755)
		// If we were unable to create the directory for any reason then we'll
		// just proceed without a cache, at the expense of repeated downloads.
		// (With that said, later installing might end up failing for the
		// same reason anyway...)
		if err == nil || os.IsExist(err) {
			cacheDir := providercache.NewDir(cachePath)
			providerInst.SetGlobalCacheDir(cacheDir)
		}
	}
	reqs, hclDiags := cfg.ProviderRequirements()
	diags = diags.Append(hclDiags)
	if diags.HasErrors() {
		return suiteDirs, diags
	}

	// For test suites we only retain the "locks" in memory for the duration
	// for one run, just to make sure that we use the same providers when we
	// eventually run the test suite.
	locks := depsfile.NewLocks()
	evts := &providercache.InstallerEvents{
		QueryPackagesFailure: func(provider addrs.Provider, err error) {
			if err != nil && provider.IsDefault() && provider.Type == "test" {
				// This is some additional context for the failure error
				// we'll generate afterwards. Not the most ideal UX but
				// good enough for this prototype implementation, to help
				// hint about the special builtin provider we use here.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Warning,
					"Probably-unintended reference to \"hashicorp/test\" provider",
					"For the purposes of this experimental implementation of module test suites, you must use the built-in test provider terraform.io/builtin/test, which requires an explicit required_providers declaration.",
				))
			}
		},
	}
	ctx = evts.OnContext(ctx)
	locks, err = providerInst.EnsureProviderVersions(ctx, locks, reqs, providercache.InstallUpgrades)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to install required providers",
			fmt.Sprintf("Couldn't install necessary providers for test configuration %s: %s.", configDir, err),
		))
		return suiteDirs, diags
	}
	suiteDirs.ProviderLocks = locks
	suiteDirs.ProviderCache = localCacheDir

	return suiteDirs, diags
}

func (c *TestCommand) runTestSuite(ctx context.Context, suiteDirs testCommandSuiteDirs) (map[string]*moduletest.Component, *states.State) {
	log.Printf("[TRACE] terraform test: Run test suite %q", suiteDirs.SuiteName)

	ret := make(map[string]*moduletest.Component)

	// To collect test results we'll use an instance of the special "test"
	// provider, which records the intention to make a test assertion during
	// planning and then hopefully updates that to an actual assertion result
	// during apply, unless an apply error causes the graph walk to exit early.
	// For this to work correctly, we must ensure we're using the same provider
	// instance for both plan and apply.
	testProvider := moduletest.NewProvider()

	// synthError is a helper to return early with a synthetic failing
	// component, for problems that prevent us from even discovering what an
	// appropriate component and assertion name might be.
	state := states.NewState()
	synthError := func(name string, desc string, msg string, diags tfdiags.Diagnostics) (map[string]*moduletest.Component, *states.State) {
		key := "(" + name + ")" // parens ensure this can't conflict with an actual component/assertion key
		ret[key] = &moduletest.Component{
			Assertions: map[string]*moduletest.Assertion{
				key: {
					Outcome:     moduletest.Error,
					Description: desc,
					Message:     msg,
					Diagnostics: diags,
				},
			},
		}
		return ret, state
	}

	// NOTE: This function intentionally deviates from the usual pattern of
	// gradually appending more diagnostics to the same diags, because
	// here we're associating each set of diagnostics with the specific
	// operation it belongs to.

	providerFactories, diags := c.testSuiteProviders(suiteDirs, testProvider)
	if diags.HasErrors() {
		// It should be unusual to get in here, because testSuiteProviders
		// should rely only on things guaranteed by prepareSuiteDir, but
		// since we're doing external I/O here there is always the risk that
		// the filesystem changes or fails between setting up and using the
		// providers.
		return synthError(
			"init",
			"terraform init",
			"failed to resolve the required providers",
			diags,
		)
	}

	plan, diags := c.testSuitePlan(ctx, suiteDirs, providerFactories)
	if diags.HasErrors() {
		// It should be unusual to get in here, because testSuitePlan
		// should rely only on things guaranteed by prepareSuiteDir, but
		// since we're doing external I/O here there is always the risk that
		// the filesystem changes or fails between setting up and using the
		// providers.
		return synthError(
			"plan",
			"terraform plan",
			"failed to create a plan",
			diags,
		)
	}

	// Now we'll apply the plan. Once we try to apply, we might've created
	// real remote objects, and so we must try to run destroy even if the
	// apply returns errors, and we must return whatever state we end up
	// with so the caller can generate additional loud errors if anything
	// is left in it.

	state, diags = c.testSuiteApply(ctx, plan, suiteDirs, providerFactories)
	if diags.HasErrors() {
		// We don't return here, unlike the others above, because we want to
		// continue to the destroy below even if there are apply errors.
		synthError(
			"apply",
			"terraform apply",
			"failed to apply the created plan",
			diags,
		)
	}

	// By the time we get here, the test provider will have gathered up all
	// of the planned assertions and the final results for any assertions that
	// were not blocked by an error. This also resets the provider so that
	// the destroy operation below won't get tripped up on stale results.
	ret = testProvider.Reset()

	state, diags = c.testSuiteDestroy(ctx, state, suiteDirs, providerFactories)
	if diags.HasErrors() {
		synthError(
			"destroy",
			"terraform destroy",
			"failed to destroy objects created during test (NOTE: leftover remote objects may still exist)",
			diags,
		)
	}

	return ret, state
}

func (c *TestCommand) testSuiteProviders(suiteDirs testCommandSuiteDirs, testProvider *moduletest.Provider) (map[addrs.Provider]providers.Factory, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := make(map[addrs.Provider]providers.Factory)

	// We can safely use the internal providers returned by Meta here because
	// the built-in provider versions can never vary based on the configuration
	// and thus we don't need to worry about potential version differences
	// between main module and test suite modules.
	for name, factory := range c.internalProviders() {
		ret[addrs.NewBuiltInProvider(name)] = factory
	}

	// For the remaining non-builtin providers, we'll just take whatever we
	// recorded earlier in the in-memory-only "lock file". All of these should
	// typically still be available because we would've only just installed
	// them, but this could fail if e.g. the filesystem has been somehow
	// damaged in the meantime.
	for provider, lock := range suiteDirs.ProviderLocks.AllProviders() {
		version := lock.Version()
		cached := suiteDirs.ProviderCache.ProviderVersion(provider, version)
		if cached == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Required provider not found",
				fmt.Sprintf("Although installation previously succeeded for %s v%s, it no longer seems to be present in the cache directory.", provider.ForDisplay(), version.String()),
			))
			continue // potentially collect up multiple errors
		}

		// NOTE: We don't consider the checksums for test suite dependencies,
		// because we're creating a fresh "lock file" each time we run anyway
		// and so they wouldn't actually guarantee anything useful.

		ret[provider] = providerFactory(cached)
	}

	// We'll replace the test provider instance with the one our caller
	// provided, so it'll be able to interrogate the test results directly.
	ret[addrs.NewBuiltInProvider("test")] = func() (providers.Interface, error) {
		return testProvider, nil
	}

	return ret, diags
}

type testSuiteRunContext struct {
	Core *terraform.Context

	PlanMode   plans.Mode
	Config     *configs.Config
	InputState *states.State
	Changes    *plans.Changes
}

func (c *TestCommand) testSuiteContext(suiteDirs testCommandSuiteDirs, providerFactories map[addrs.Provider]providers.Factory, state *states.State, plan *plans.Plan, destroy bool) (*testSuiteRunContext, tfdiags.Diagnostics) {
	var changes *plans.Changes
	if plan != nil {
		changes = plan.Changes
	}

	planMode := plans.NormalMode
	if destroy {
		planMode = plans.DestroyMode
	}

	tfCtx, diags := terraform.NewContext(&terraform.ContextOpts{
		Providers: providerFactories,

		// We just use the provisioners from the main Meta here, because
		// unlike providers provisioner plugins are not automatically
		// installable anyway, and so we'll need to hunt for them in the same
		// legacy way that normal Terraform operations do.
		Provisioners: c.provisionerFactories(),

		Meta: &terraform.ContextMeta{
			Env: "test_" + suiteDirs.SuiteName,
		},
	})
	if diags.HasErrors() {
		return nil, diags
	}
	return &testSuiteRunContext{
		Core: tfCtx,

		PlanMode:   planMode,
		Config:     suiteDirs.Config,
		InputState: state,
		Changes:    changes,
	}, diags
}

func (c *TestCommand) testSuitePlan(ctx context.Context, suiteDirs testCommandSuiteDirs, providerFactories map[addrs.Provider]providers.Factory) (*plans.Plan, tfdiags.Diagnostics) {
	log.Printf("[TRACE] terraform test: create plan for suite %q", suiteDirs.SuiteName)
	runCtx, diags := c.testSuiteContext(suiteDirs, providerFactories, nil, nil, false)
	if diags.HasErrors() {
		return nil, diags
	}

	// We'll also validate as part of planning, to ensure that the test
	// configuration would pass "terraform validate". This is actually
	// largely redundant with the runCtx.Core.Plan call below, but was
	// included here originally because Plan did _originally_ assume that
	// an earlier Validate had already passed, but now does its own
	// validation work as (mostly) a superset of validate.
	moreDiags := runCtx.Core.Validate(runCtx.Config)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	plan, moreDiags := runCtx.Core.Plan(
		runCtx.Config, runCtx.InputState, &terraform.PlanOpts{Mode: runCtx.PlanMode},
	)
	diags = diags.Append(moreDiags)
	return plan, diags
}

func (c *TestCommand) testSuiteApply(ctx context.Context, plan *plans.Plan, suiteDirs testCommandSuiteDirs, providerFactories map[addrs.Provider]providers.Factory) (*states.State, tfdiags.Diagnostics) {
	log.Printf("[TRACE] terraform test: apply plan for suite %q", suiteDirs.SuiteName)
	runCtx, diags := c.testSuiteContext(suiteDirs, providerFactories, nil, plan, false)
	if diags.HasErrors() {
		// To make things easier on the caller, we'll return a valid empty
		// state even in this case.
		return states.NewState(), diags
	}

	state, moreDiags := runCtx.Core.Apply(plan, runCtx.Config)
	diags = diags.Append(moreDiags)
	return state, diags
}

func (c *TestCommand) testSuiteDestroy(ctx context.Context, state *states.State, suiteDirs testCommandSuiteDirs, providerFactories map[addrs.Provider]providers.Factory) (*states.State, tfdiags.Diagnostics) {
	log.Printf("[TRACE] terraform test: plan to destroy any existing objects for suite %q", suiteDirs.SuiteName)
	runCtx, diags := c.testSuiteContext(suiteDirs, providerFactories, state, nil, true)
	if diags.HasErrors() {
		return state, diags
	}

	plan, moreDiags := runCtx.Core.Plan(
		runCtx.Config, runCtx.InputState, &terraform.PlanOpts{Mode: runCtx.PlanMode},
	)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return state, diags
	}

	log.Printf("[TRACE] terraform test: apply the plan to destroy any existing objects for suite %q", suiteDirs.SuiteName)
	runCtx, moreDiags = c.testSuiteContext(suiteDirs, providerFactories, state, plan, true)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return state, diags
	}

	state, moreDiags = runCtx.Core.Apply(plan, runCtx.Config)
	diags = diags.Append(moreDiags)
	return state, diags
}

func (c *TestCommand) collectSuiteNames() ([]string, error) {
	items, err := ioutil.ReadDir("tests")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	ret := make([]string, 0, len(items))
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		name := item.Name()
		suitePath := filepath.Join("tests", name)
		tfFiles, err := filepath.Glob(filepath.Join(suitePath, "*.tf"))
		if err != nil {
			// We'll just ignore it and treat it like a dir with no .tf files
			tfFiles = nil
		}
		tfJSONFiles, err := filepath.Glob(filepath.Join(suitePath, "*.tf.json"))
		if err != nil {
			// We'll just ignore it and treat it like a dir with no .tf.json files
			tfJSONFiles = nil
		}
		if (len(tfFiles) + len(tfJSONFiles)) == 0 {
			// Not a test suite, then.
			continue
		}
		ret = append(ret, name)
	}

	return ret, nil
}

func (c *TestCommand) Help() string {
	helpText := `
Usage: terraform test [options]

  This is an experimental command to help with automated integration
  testing of shared modules. The usage and behavior of this command is
  likely to change in breaking ways in subsequent releases, as we
  are currently using this command primarily for research purposes.

  In its current experimental form, "test" will look under the current
  working directory for a subdirectory called "tests", and then within
  that directory search for one or more subdirectories that contain
  ".tf" or ".tf.json" files. For any that it finds, it will perform
  Terraform operations similar to the following sequence of commands
  in each of those directories:
      terraform validate
      terraform apply
      terraform destroy

  The test configurations should not declare any input variables and
  should at least contain a call to the module being tested, which
  will always be available at the path ../.. due to the expected
  filesystem layout.

  The tests are considered to be successful if all of the above steps
  succeed.

  Test configurations may optionally include uses of the special
  built-in test provider terraform.io/builtin/test, which allows
  writing explicit test assertions which must also all pass in order
  for the test run to be considered successful.

  This initial implementation is intended as a minimally-viable
  product to use for further research and experimentation, and in
  particular it currently lacks the following capabilities that we
  expect to consider in later iterations, based on feedback:
    - Testing of subsequent updates to existing infrastructure,
      where currently it only supports initial creation and
      then destruction.
    - Testing top-level modules that are intended to be used for
      "real" environments, which typically have hard-coded values
      that don't permit creating a separate "copy" for testing.
    - Some sort of support for unit test runs that don't interact
      with remote systems at all, e.g. for use in checking pull
      requests from untrusted contributors.

  In the meantime, we'd like to hear feedback from module authors
  who have tried writing some experimental tests for their modules
  about what sorts of tests you were able to write, what sorts of
  tests you weren't able to write, and any tests that you were
  able to write but that were difficult to model in some way.

Options:

  -compact-warnings  Use a more compact representation for warnings, if
                     this command produces only warnings and no errors.

  -junit-xml=FILE    In addition to the usual output, also write test
                     results to the given file path in JUnit XML format.
                     This format is commonly supported by CI systems, and
                     they typically expect to be given a filename to search
                     for in the test workspace after the test run finishes.

  -no-color          Don't include virtual terminal formatting sequences in
                     the output.
`
	return strings.TrimSpace(helpText)
}

func (c *TestCommand) Synopsis() string {
	return "Experimental support for module integration testing"
}

type testCommandSuiteDirs struct {
	SuiteName string

	ConfigDir    string
	ModulesDir   string
	ProvidersDir string

	Config        *configs.Config
	ProviderCache *providercache.Dir
	ProviderLocks *depsfile.Locks
}
