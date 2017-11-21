package resource

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"syscall"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/logutils"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/terraform"
)

// flagSweep is a flag available when running tests on the command line. It
// contains a comma seperated list of regions to for the sweeper functions to
// run in.  This flag bypasses the normal Test path and instead runs functions designed to
// clean up any leaked resources a testing environment could have created. It is
// a best effort attempt, and relies on Provider authors to implement "Sweeper"
// methods for resources.

// Adding Sweeper methods with AddTestSweepers will
// construct a list of sweeper funcs to be called here. We iterate through
// regions provided by the sweep flag, and for each region we iterate through the
// tests, and exit on any errors. At time of writing, sweepers are ran
// sequentially, however they can list dependencies to be ran first. We track
// the sweepers that have been ran, so as to not run a sweeper twice for a given
// region.
//
// WARNING:
// Sweepers are designed to be destructive. You should not use the -sweep flag
// in any environment that is not strictly a test environment. Resources will be
// destroyed.

var flagSweep = flag.String("sweep", "", "List of Regions to run available Sweepers")
var flagSweepRun = flag.String("sweep-run", "", "Comma seperated list of Sweeper Tests to run")
var sweeperFuncs map[string]*Sweeper

// map of sweepers that have ran, and the success/fail status based on any error
// raised
var sweeperRunList map[string]bool

// type SweeperFunc is a signature for a function that acts as a sweeper. It
// accepts a string for the region that the sweeper is to be ran in. This
// function must be able to construct a valid client for that region.
type SweeperFunc func(r string) error

type Sweeper struct {
	// Name for sweeper. Must be unique to be ran by the Sweeper Runner
	Name string

	// Dependencies list the const names of other Sweeper functions that must be ran
	// prior to running this Sweeper. This is an ordered list that will be invoked
	// recursively at the helper/resource level
	Dependencies []string

	// Sweeper function that when invoked sweeps the Provider of specific
	// resources
	F SweeperFunc
}

func init() {
	sweeperFuncs = make(map[string]*Sweeper)
}

// AddTestSweepers function adds a given name and Sweeper configuration
// pair to the internal sweeperFuncs map. Invoke this function to register a
// resource sweeper to be available for running when the -sweep flag is used
// with `go test`. Sweeper names must be unique to help ensure a given sweeper
// is only ran once per run.
func AddTestSweepers(name string, s *Sweeper) {
	if _, ok := sweeperFuncs[name]; ok {
		log.Fatalf("[ERR] Error adding (%s) to sweeperFuncs: function already exists in map", name)
	}

	sweeperFuncs[name] = s
}

func TestMain(m *testing.M) {
	flag.Parse()
	if *flagSweep != "" {
		// parse flagSweep contents for regions to run
		regions := strings.Split(*flagSweep, ",")

		// get filtered list of sweepers to run based on sweep-run flag
		sweepers := filterSweepers(*flagSweepRun, sweeperFuncs)
		for _, region := range regions {
			region = strings.TrimSpace(region)
			// reset sweeperRunList for each region
			sweeperRunList = map[string]bool{}

			log.Printf("[DEBUG] Running Sweepers for region (%s):\n", region)
			for _, sweeper := range sweepers {
				if err := runSweeperWithRegion(region, sweeper); err != nil {
					log.Fatalf("[ERR] error running (%s): %s", sweeper.Name, err)
				}
			}

			log.Printf("Sweeper Tests ran:\n")
			for s, _ := range sweeperRunList {
				fmt.Printf("\t- %s\n", s)
			}
		}
	} else {
		os.Exit(m.Run())
	}
}

// filterSweepers takes a comma seperated string listing the names of sweepers
// to be ran, and returns a filtered set from the list of all of sweepers to
// run based on the names given.
func filterSweepers(f string, source map[string]*Sweeper) map[string]*Sweeper {
	filterSlice := strings.Split(strings.ToLower(f), ",")
	if len(filterSlice) == 1 && filterSlice[0] == "" {
		// if the filter slice is a single element of "" then no sweeper list was
		// given, so just return the full list
		return source
	}

	sweepers := make(map[string]*Sweeper)
	for name, sweeper := range source {
		for _, s := range filterSlice {
			if strings.Contains(strings.ToLower(name), s) {
				sweepers[name] = sweeper
			}
		}
	}
	return sweepers
}

// runSweeperWithRegion recieves a sweeper and a region, and recursively calls
// itself with that region for every dependency found for that sweeper. If there
// are no dependencies, invoke the contained sweeper fun with the region, and
// add the success/fail status to the sweeperRunList.
func runSweeperWithRegion(region string, s *Sweeper) error {
	for _, dep := range s.Dependencies {
		if depSweeper, ok := sweeperFuncs[dep]; ok {
			log.Printf("[DEBUG] Sweeper (%s) has dependency (%s), running..", s.Name, dep)
			if err := runSweeperWithRegion(region, depSweeper); err != nil {
				return err
			}
		} else {
			log.Printf("[DEBUG] Sweeper (%s) has dependency (%s), but that sweeper was not found", s.Name, dep)
		}
	}

	if _, ok := sweeperRunList[s.Name]; ok {
		log.Printf("[DEBUG] Sweeper (%s) already ran in region (%s)", s.Name, region)
		return nil
	}

	runE := s.F(region)
	if runE == nil {
		sweeperRunList[s.Name] = true
	} else {
		sweeperRunList[s.Name] = false
	}

	return runE
}

const TestEnvVar = "TF_ACC"

// TestProvider can be implemented by any ResourceProvider to provide custom
// reset functionality at the start of an acceptance test.
// The helper/schema Provider implements this interface.
type TestProvider interface {
	TestReset() error
}

// TestCheckFunc is the callback type used with acceptance tests to check
// the state of a resource. The state passed in is the latest state known,
// or in the case of being after a destroy, it is the last known state when
// it was created.
type TestCheckFunc func(*terraform.State) error

// ImportStateCheckFunc is the check function for ImportState tests
type ImportStateCheckFunc func([]*terraform.InstanceState) error

// ImportStateIdFunc is an ID generation function to help with complex ID
// generation for ImportState tests.
type ImportStateIdFunc func(*terraform.State) (string, error)

// TestCase is a single acceptance test case used to test the apply/destroy
// lifecycle of a resource in a specific configuration.
//
// When the destroy plan is executed, the config from the last TestStep
// is used to plan it.
type TestCase struct {
	// IsUnitTest allows a test to run regardless of the TF_ACC
	// environment variable. This should be used with care - only for
	// fast tests on local resources (e.g. remote state with a local
	// backend) but can be used to increase confidence in correct
	// operation of Terraform without waiting for a full acctest run.
	IsUnitTest bool

	// PreCheck, if non-nil, will be called before any test steps are
	// executed. It will only be executed in the case that the steps
	// would run, so it can be used for some validation before running
	// acceptance tests, such as verifying that keys are setup.
	PreCheck func()

	// Providers is the ResourceProvider that will be under test.
	//
	// Alternately, ProviderFactories can be specified for the providers
	// that are valid. This takes priority over Providers.
	//
	// The end effect of each is the same: specifying the providers that
	// are used within the tests.
	Providers         map[string]terraform.ResourceProvider
	ProviderFactories map[string]terraform.ResourceProviderFactory

	// PreventPostDestroyRefresh can be set to true for cases where data sources
	// are tested alongside real resources
	PreventPostDestroyRefresh bool

	// CheckDestroy is called after the resource is finally destroyed
	// to allow the tester to test that the resource is truly gone.
	CheckDestroy TestCheckFunc

	// Steps are the apply sequences done within the context of the
	// same state. Each step can have its own check to verify correctness.
	Steps []TestStep

	// The settings below control the "ID-only refresh test." This is
	// an enabled-by-default test that tests that a refresh can be
	// refreshed with only an ID to result in the same attributes.
	// This validates completeness of Refresh.
	//
	// IDRefreshName is the name of the resource to check. This will
	// default to the first non-nil primary resource in the state.
	//
	// IDRefreshIgnore is a list of configuration keys that will be ignored.
	IDRefreshName   string
	IDRefreshIgnore []string
}

// TestStep is a single apply sequence of a test, done within the
// context of a state.
//
// Multiple TestSteps can be sequenced in a Test to allow testing
// potentially complex update logic. In general, simply create/destroy
// tests will only need one step.
type TestStep struct {
	// ResourceName should be set to the name of the resource
	// that is being tested. Example: "aws_instance.foo". Various test
	// modes use this to auto-detect state information.
	//
	// This is only required if the test mode settings below say it is
	// for the mode you're using.
	ResourceName string

	// PreConfig is called before the Config is applied to perform any per-step
	// setup that needs to happen. This is called regardless of "test mode"
	// below.
	PreConfig func()

	//---------------------------------------------------------------
	// Test modes. One of the following groups of settings must be
	// set to determine what the test step will do. Ideally we would've
	// used Go interfaces here but there are now hundreds of tests we don't
	// want to re-type so instead we just determine which step logic
	// to run based on what settings below are set.
	//---------------------------------------------------------------

	//---------------------------------------------------------------
	// Plan, Apply testing
	//---------------------------------------------------------------

	// Config a string of the configuration to give to Terraform. If this
	// is set, then the TestCase will execute this step with the same logic
	// as a `terraform apply`.
	Config string

	// Check is called after the Config is applied. Use this step to
	// make your own API calls to check the status of things, and to
	// inspect the format of the ResourceState itself.
	//
	// If an error is returned, the test will fail. In this case, a
	// destroy plan will still be attempted.
	//
	// If this is nil, no check is done on this step.
	Check TestCheckFunc

	// Destroy will create a destroy plan if set to true.
	Destroy bool

	// ExpectNonEmptyPlan can be set to true for specific types of tests that are
	// looking to verify that a diff occurs
	ExpectNonEmptyPlan bool

	// ExpectError allows the construction of test cases that we expect to fail
	// with an error. The specified regexp must match against the error for the
	// test to pass.
	ExpectError *regexp.Regexp

	// PlanOnly can be set to only run `plan` with this configuration, and not
	// actually apply it. This is useful for ensuring config changes result in
	// no-op plans
	PlanOnly bool

	// PreventPostDestroyRefresh can be set to true for cases where data sources
	// are tested alongside real resources
	PreventPostDestroyRefresh bool

	// SkipFunc is called before applying config, but after PreConfig
	// This is useful for defining test steps with platform-dependent checks
	SkipFunc func() (bool, error)

	//---------------------------------------------------------------
	// ImportState testing
	//---------------------------------------------------------------

	// ImportState, if true, will test the functionality of ImportState
	// by importing the resource with ResourceName (must be set) and the
	// ID of that resource.
	ImportState bool

	// ImportStateId is the ID to perform an ImportState operation with.
	// This is optional. If it isn't set, then the resource ID is automatically
	// determined by inspecting the state for ResourceName's ID.
	ImportStateId string

	// ImportStateIdPrefix is the prefix added in front of ImportStateId.
	// This can be useful in complex import cases, where more than one
	// attribute needs to be passed on as the Import ID. Mainly in cases
	// where the ID is not known, and a known prefix needs to be added to
	// the unset ImportStateId field.
	ImportStateIdPrefix string

	// ImportStateIdFunc is a function that can be used to dynamically generate
	// the ID for the ImportState tests. It is sent the state, which can be
	// checked to derive the attributes necessary and generate the string in the
	// desired format.
	ImportStateIdFunc ImportStateIdFunc

	// ImportStateCheck checks the results of ImportState. It should be
	// used to verify that the resulting value of ImportState has the
	// proper resources, IDs, and attributes.
	ImportStateCheck ImportStateCheckFunc

	// ImportStateVerify, if true, will also check that the state values
	// that are finally put into the state after import match for all the
	// IDs returned by the Import.
	//
	// ImportStateVerifyIgnore are fields that should not be verified to
	// be equal. These can be set to ephemeral fields or fields that can't
	// be refreshed and don't matter.
	ImportStateVerify       bool
	ImportStateVerifyIgnore []string
}

// Set to a file mask in sprintf format where %s is test name
const EnvLogPathMask = "TF_LOG_PATH_MASK"

func LogOutput(t TestT) (logOutput io.Writer, err error) {
	logOutput = ioutil.Discard

	logLevel := logging.LogLevel()
	if logLevel == "" {
		return
	}

	logOutput = os.Stderr

	if logPath := os.Getenv(logging.EnvLogFile); logPath != "" {
		var err error
		logOutput, err = os.OpenFile(logPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
	}

	if logPathMask := os.Getenv(EnvLogPathMask); logPathMask != "" {
		// Escape special characters which may appear if we have subtests
		testName := strings.Replace(t.Name(), "/", "__", -1)

		logPath := fmt.Sprintf(logPathMask, testName)
		var err error
		logOutput, err = os.OpenFile(logPath, syscall.O_CREAT|syscall.O_RDWR|syscall.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
	}

	// This was the default since the beginning
	logOutput = &logutils.LevelFilter{
		Levels:   logging.ValidLevels,
		MinLevel: logutils.LogLevel(logLevel),
		Writer:   logOutput,
	}

	return
}

// Test performs an acceptance test on a resource.
//
// Tests are not run unless an environmental variable "TF_ACC" is
// set to some non-empty value. This is to avoid test cases surprising
// a user by creating real resources.
//
// Tests will fail unless the verbose flag (`go test -v`, or explicitly
// the "-test.v" flag) is set. Because some acceptance tests take quite
// long, we require the verbose flag so users are able to see progress
// output.
func Test(t TestT, c TestCase) {
	// We only run acceptance tests if an env var is set because they're
	// slow and generally require some outside configuration. You can opt out
	// of this with OverrideEnvVar on individual TestCases.
	if os.Getenv(TestEnvVar) == "" && !c.IsUnitTest {
		t.Skip(fmt.Sprintf(
			"Acceptance tests skipped unless env '%s' set",
			TestEnvVar))
		return
	}

	logWriter, err := LogOutput(t)
	if err != nil {
		t.Error(fmt.Errorf("error setting up logging: %s", err))
	}
	log.SetOutput(logWriter)

	// We require verbose mode so that the user knows what is going on.
	if !testTesting && !testing.Verbose() && !c.IsUnitTest {
		t.Fatal("Acceptance tests must be run with the -v flag on tests")
		return
	}

	// Run the PreCheck if we have it
	if c.PreCheck != nil {
		c.PreCheck()
	}

	providerResolver, err := testProviderResolver(c)
	if err != nil {
		t.Fatal(err)
	}
	opts := terraform.ContextOpts{ProviderResolver: providerResolver}

	// A single state variable to track the lifecycle, starting with no state
	var state *terraform.State

	// Go through each step and run it
	var idRefreshCheck *terraform.ResourceState
	idRefresh := c.IDRefreshName != ""
	errored := false
	for i, step := range c.Steps {
		var err error
		log.Printf("[DEBUG] Test: Executing step %d", i)

		if step.SkipFunc != nil {
			skip, err := step.SkipFunc()
			if err != nil {
				t.Fatal(err)
			}
			if skip {
				log.Printf("[WARN] Skipping step %d", i)
				continue
			}
		}

		if step.Config == "" && !step.ImportState {
			err = fmt.Errorf(
				"unknown test mode for step. Please see TestStep docs\n\n%#v",
				step)
		} else {
			if step.ImportState {
				if step.Config == "" {
					step.Config = testProviderConfig(c)
				}

				// Can optionally set step.Config in addition to
				// step.ImportState, to provide config for the import.
				state, err = testStepImportState(opts, state, step)
			} else {
				state, err = testStepConfig(opts, state, step)
			}
		}

		// If there was an error, exit
		if err != nil {
			// Perhaps we expected an error? Check if it matches
			if step.ExpectError != nil {
				if !step.ExpectError.MatchString(err.Error()) {
					errored = true
					t.Error(fmt.Sprintf(
						"Step %d, expected error:\n\n%s\n\nTo match:\n\n%s\n\n",
						i, err, step.ExpectError))
					break
				}
			} else {
				errored = true
				t.Error(fmt.Sprintf(
					"Step %d error: %s", i, err))
				break
			}
		}

		// If we've never checked an id-only refresh and our state isn't
		// empty, find the first resource and test it.
		if idRefresh && idRefreshCheck == nil && !state.Empty() {
			// Find the first non-nil resource in the state
			for _, m := range state.Modules {
				if len(m.Resources) > 0 {
					if v, ok := m.Resources[c.IDRefreshName]; ok {
						idRefreshCheck = v
					}

					break
				}
			}

			// If we have an instance to check for refreshes, do it
			// immediately. We do it in the middle of another test
			// because it shouldn't affect the overall state (refresh
			// is read-only semantically) and we want to fail early if
			// this fails. If refresh isn't read-only, then this will have
			// caught a different bug.
			if idRefreshCheck != nil {
				log.Printf(
					"[WARN] Test: Running ID-only refresh check on %s",
					idRefreshCheck.Primary.ID)
				if err := testIDOnlyRefresh(c, opts, step, idRefreshCheck); err != nil {
					log.Printf("[ERROR] Test: ID-only test failed: %s", err)
					t.Error(fmt.Sprintf(
						"[ERROR] Test: ID-only test failed: %s", err))
					break
				}
			}
		}
	}

	// If we never checked an id-only refresh, it is a failure.
	if idRefresh {
		if !errored && len(c.Steps) > 0 && idRefreshCheck == nil {
			t.Error("ID-only refresh check never ran.")
		}
	}

	// If we have a state, then run the destroy
	if state != nil {
		lastStep := c.Steps[len(c.Steps)-1]
		destroyStep := TestStep{
			Config:                    lastStep.Config,
			Check:                     c.CheckDestroy,
			Destroy:                   true,
			PreventPostDestroyRefresh: c.PreventPostDestroyRefresh,
		}

		log.Printf("[WARN] Test: Executing destroy step")
		state, err := testStep(opts, state, destroyStep)
		if err != nil {
			t.Error(fmt.Sprintf(
				"Error destroying resource! WARNING: Dangling resources\n"+
					"may exist. The full state and error is shown below.\n\n"+
					"Error: %s\n\nState: %s",
				err,
				state))
		}
	} else {
		log.Printf("[WARN] Skipping destroy test since there is no state.")
	}
}

// testProviderConfig takes the list of Providers in a TestCase and returns a
// config with only empty provider blocks. This is useful for Import, where no
// config is provided, but the providers must be defined.
func testProviderConfig(c TestCase) string {
	var lines []string
	for p := range c.Providers {
		lines = append(lines, fmt.Sprintf("provider %q {}\n", p))
	}

	return strings.Join(lines, "")
}

// testProviderResolver is a helper to build a ResourceProviderResolver
// with pre instantiated ResourceProviders, so that we can reset them for the
// test, while only calling the factory function once.
// Any errors are stored so that they can be returned by the factory in
// terraform to match non-test behavior.
func testProviderResolver(c TestCase) (terraform.ResourceProviderResolver, error) {
	ctxProviders := c.ProviderFactories
	if ctxProviders == nil {
		ctxProviders = make(map[string]terraform.ResourceProviderFactory)
	}

	// add any fixed providers
	for k, p := range c.Providers {
		ctxProviders[k] = terraform.ResourceProviderFactoryFixed(p)
	}

	// reset the providers if needed
	for k, pf := range ctxProviders {
		// we can ignore any errors here, if we don't have a provider to reset
		// the error will be handled later
		p, err := pf()
		if err != nil {
			return nil, err
		}
		if p, ok := p.(TestProvider); ok {
			err := p.TestReset()
			if err != nil {
				return nil, fmt.Errorf("[ERROR] failed to reset provider %q: %s", k, err)
			}
		}
	}

	return terraform.ResourceProviderResolverFixed(ctxProviders), nil
}

// UnitTest is a helper to force the acceptance testing harness to run in the
// normal unit test suite. This should only be used for resource that don't
// have any external dependencies.
func UnitTest(t TestT, c TestCase) {
	c.IsUnitTest = true
	Test(t, c)
}

func testIDOnlyRefresh(c TestCase, opts terraform.ContextOpts, step TestStep, r *terraform.ResourceState) error {
	// TODO: We guard by this right now so master doesn't explode. We
	// need to remove this eventually to make this part of the normal tests.
	if os.Getenv("TF_ACC_IDONLY") == "" {
		return nil
	}

	name := fmt.Sprintf("%s.foo", r.Type)

	// Build the state. The state is just the resource with an ID. There
	// are no attributes. We only set what is needed to perform a refresh.
	state := terraform.NewState()
	state.RootModule().Resources[name] = &terraform.ResourceState{
		Type: r.Type,
		Primary: &terraform.InstanceState{
			ID: r.Primary.ID,
		},
	}

	// Create the config module. We use the full config because Refresh
	// doesn't have access to it and we may need things like provider
	// configurations. The initial implementation of id-only checks used
	// an empty config module, but that caused the aforementioned problems.
	mod, err := testModule(opts, step)
	if err != nil {
		return err
	}

	// Initialize the context
	opts.Module = mod
	opts.State = state
	ctx, err := terraform.NewContext(&opts)
	if err != nil {
		return err
	}
	if diags := ctx.Validate(); len(diags) > 0 {
		if diags.HasErrors() {
			return errwrap.Wrapf("config is invalid: {{err}}", diags.Err())
		}

		log.Printf("[WARN] Config warnings:\n%s", diags.Err().Error())
	}

	// Refresh!
	state, err = ctx.Refresh()
	if err != nil {
		return fmt.Errorf("Error refreshing: %s", err)
	}

	// Verify attribute equivalence.
	actualR := state.RootModule().Resources[name]
	if actualR == nil {
		return fmt.Errorf("Resource gone!")
	}
	if actualR.Primary == nil {
		return fmt.Errorf("Resource has no primary instance")
	}
	actual := actualR.Primary.Attributes
	expected := r.Primary.Attributes
	// Remove fields we're ignoring
	for _, v := range c.IDRefreshIgnore {
		for k, _ := range actual {
			if strings.HasPrefix(k, v) {
				delete(actual, k)
			}
		}
		for k, _ := range expected {
			if strings.HasPrefix(k, v) {
				delete(expected, k)
			}
		}
	}

	if !reflect.DeepEqual(actual, expected) {
		// Determine only the different attributes
		for k, v := range expected {
			if av, ok := actual[k]; ok && v == av {
				delete(expected, k)
				delete(actual, k)
			}
		}

		spewConf := spew.NewDefaultConfig()
		spewConf.SortKeys = true
		return fmt.Errorf(
			"Attributes not equivalent. Difference is shown below. Top is actual, bottom is expected."+
				"\n\n%s\n\n%s",
			spewConf.Sdump(actual), spewConf.Sdump(expected))
	}

	return nil
}

func testModule(
	opts terraform.ContextOpts,
	step TestStep) (*module.Tree, error) {
	if step.PreConfig != nil {
		step.PreConfig()
	}

	cfgPath, err := ioutil.TempDir("", "tf-test")
	if err != nil {
		return nil, fmt.Errorf(
			"Error creating temporary directory for config: %s", err)
	}
	defer os.RemoveAll(cfgPath)

	// Write the configuration
	cfgF, err := os.Create(filepath.Join(cfgPath, "main.tf"))
	if err != nil {
		return nil, fmt.Errorf(
			"Error creating temporary file for config: %s", err)
	}

	_, err = io.Copy(cfgF, strings.NewReader(step.Config))
	cfgF.Close()
	if err != nil {
		return nil, fmt.Errorf(
			"Error creating temporary file for config: %s", err)
	}

	// Parse the configuration
	mod, err := module.NewTreeModule("", cfgPath)
	if err != nil {
		return nil, fmt.Errorf(
			"Error loading configuration: %s", err)
	}

	// Load the modules
	modStorage := &module.Storage{
		StorageDir: filepath.Join(cfgPath, ".tfmodules"),
		Mode:       module.GetModeGet,
	}
	err = mod.Load(modStorage)
	if err != nil {
		return nil, fmt.Errorf("Error downloading modules: %s", err)
	}

	return mod, nil
}

func testResource(c TestStep, state *terraform.State) (*terraform.ResourceState, error) {
	if c.ResourceName == "" {
		return nil, fmt.Errorf("ResourceName must be set in TestStep")
	}

	for _, m := range state.Modules {
		if len(m.Resources) > 0 {
			if v, ok := m.Resources[c.ResourceName]; ok {
				return v, nil
			}
		}
	}

	return nil, fmt.Errorf(
		"Resource specified by ResourceName couldn't be found: %s", c.ResourceName)
}

// ComposeTestCheckFunc lets you compose multiple TestCheckFuncs into
// a single TestCheckFunc.
//
// As a user testing their provider, this lets you decompose your checks
// into smaller pieces more easily.
func ComposeTestCheckFunc(fs ...TestCheckFunc) TestCheckFunc {
	return func(s *terraform.State) error {
		for i, f := range fs {
			if err := f(s); err != nil {
				return fmt.Errorf("Check %d/%d error: %s", i+1, len(fs), err)
			}
		}

		return nil
	}
}

// ComposeAggregateTestCheckFunc lets you compose multiple TestCheckFuncs into
// a single TestCheckFunc.
//
// As a user testing their provider, this lets you decompose your checks
// into smaller pieces more easily.
//
// Unlike ComposeTestCheckFunc, ComposeAggergateTestCheckFunc runs _all_ of the
// TestCheckFuncs and aggregates failures.
func ComposeAggregateTestCheckFunc(fs ...TestCheckFunc) TestCheckFunc {
	return func(s *terraform.State) error {
		var result *multierror.Error

		for i, f := range fs {
			if err := f(s); err != nil {
				result = multierror.Append(result, fmt.Errorf("Check %d/%d error: %s", i+1, len(fs), err))
			}
		}

		return result.ErrorOrNil()
	}
}

// TestCheckResourceAttrSet is a TestCheckFunc which ensures a value
// exists in state for the given name/key combination. It is useful when
// testing that computed values were set, when it is not possible to
// know ahead of time what the values will be.
func TestCheckResourceAttrSet(name, key string) TestCheckFunc {
	return func(s *terraform.State) error {
		is, err := primaryInstanceState(s, name)
		if err != nil {
			return err
		}

		if val, ok := is.Attributes[key]; ok && val != "" {
			return nil
		}

		return fmt.Errorf("%s: Attribute '%s' expected to be set", name, key)
	}
}

// TestCheckResourceAttr is a TestCheckFunc which validates
// the value in state for the given name/key combination.
func TestCheckResourceAttr(name, key, value string) TestCheckFunc {
	return func(s *terraform.State) error {
		is, err := primaryInstanceState(s, name)
		if err != nil {
			return err
		}

		if v, ok := is.Attributes[key]; !ok || v != value {
			if !ok {
				return fmt.Errorf("%s: Attribute '%s' not found", name, key)
			}

			return fmt.Errorf(
				"%s: Attribute '%s' expected %#v, got %#v",
				name,
				key,
				value,
				v)
		}

		return nil
	}
}

// TestCheckNoResourceAttr is a TestCheckFunc which ensures that
// NO value exists in state for the given name/key combination.
func TestCheckNoResourceAttr(name, key string) TestCheckFunc {
	return func(s *terraform.State) error {
		is, err := primaryInstanceState(s, name)
		if err != nil {
			return err
		}

		if _, ok := is.Attributes[key]; ok {
			return fmt.Errorf("%s: Attribute '%s' found when not expected", name, key)
		}

		return nil
	}
}

// TestMatchResourceAttr is a TestCheckFunc which checks that the value
// in state for the given name/key combination matches the given regex.
func TestMatchResourceAttr(name, key string, r *regexp.Regexp) TestCheckFunc {
	return func(s *terraform.State) error {
		is, err := primaryInstanceState(s, name)
		if err != nil {
			return err
		}

		if !r.MatchString(is.Attributes[key]) {
			return fmt.Errorf(
				"%s: Attribute '%s' didn't match %q, got %#v",
				name,
				key,
				r.String(),
				is.Attributes[key])
		}

		return nil
	}
}

// TestCheckResourceAttrPtr is like TestCheckResourceAttr except the
// value is a pointer so that it can be updated while the test is running.
// It will only be dereferenced at the point this step is run.
func TestCheckResourceAttrPtr(name string, key string, value *string) TestCheckFunc {
	return func(s *terraform.State) error {
		return TestCheckResourceAttr(name, key, *value)(s)
	}
}

// TestCheckResourceAttrPair is a TestCheckFunc which validates that the values
// in state for a pair of name/key combinations are equal.
func TestCheckResourceAttrPair(nameFirst, keyFirst, nameSecond, keySecond string) TestCheckFunc {
	return func(s *terraform.State) error {
		isFirst, err := primaryInstanceState(s, nameFirst)
		if err != nil {
			return err
		}
		vFirst, ok := isFirst.Attributes[keyFirst]
		if !ok {
			return fmt.Errorf("%s: Attribute '%s' not found", nameFirst, keyFirst)
		}

		isSecond, err := primaryInstanceState(s, nameSecond)
		if err != nil {
			return err
		}
		vSecond, ok := isSecond.Attributes[keySecond]
		if !ok {
			return fmt.Errorf("%s: Attribute '%s' not found", nameSecond, keySecond)
		}

		if vFirst != vSecond {
			return fmt.Errorf(
				"%s: Attribute '%s' expected %#v, got %#v",
				nameFirst,
				keyFirst,
				vSecond,
				vFirst)
		}

		return nil
	}
}

// TestCheckOutput checks an output in the Terraform configuration
func TestCheckOutput(name, value string) TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Outputs[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Value != value {
			return fmt.Errorf(
				"Output '%s': expected %#v, got %#v",
				name,
				value,
				rs)
		}

		return nil
	}
}

func TestMatchOutput(name string, r *regexp.Regexp) TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Outputs[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if !r.MatchString(rs.Value.(string)) {
			return fmt.Errorf(
				"Output '%s': %#v didn't match %q",
				name,
				rs,
				r.String())
		}

		return nil
	}
}

// TestT is the interface used to handle the test lifecycle of a test.
//
// Users should just use a *testing.T object, which implements this.
type TestT interface {
	Error(args ...interface{})
	Fatal(args ...interface{})
	Skip(args ...interface{})
	Name() string
}

// This is set to true by unit tests to alter some behavior
var testTesting = false

// primaryInstanceState returns the primary instance state for the given resource name.
func primaryInstanceState(s *terraform.State, name string) (*terraform.InstanceState, error) {
	ms := s.RootModule()
	rs, ok := ms.Resources[name]
	if !ok {
		return nil, fmt.Errorf("Not found: %s", name)
	}

	is := rs.Primary
	if is == nil {
		return nil, fmt.Errorf("No primary instance: %s", name)
	}

	return is, nil
}
