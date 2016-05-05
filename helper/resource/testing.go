package resource

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/terraform"
)

const TestEnvVar = "TF_ACC"

// UnitTestOverride is a value that when set in TestEnvVar indicates that this
// is a unit test borrowing the acceptance testing framework.
const UnitTestOverride = "UnitTestOverride"

// TestCheckFunc is the callback type used with acceptance tests to check
// the state of a resource. The state passed in is the latest state known,
// or in the case of being after a destroy, it is the last known state when
// it was created.
type TestCheckFunc func(*terraform.State) error

// TestCase is a single acceptance test case used to test the apply/destroy
// lifecycle of a resource in a specific configuration.
//
// When the destroy plan is executed, the config from the last TestStep
// is used to plan it.
type TestCase struct {
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
	// PreConfig is called before the Config is applied to perform any per-step
	// setup that needs to happen
	PreConfig func()

	// Config a string of the configuration to give to Terraform.
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
	// slow and generally require some outside configuration.
	if os.Getenv(TestEnvVar) == "" {
		t.Skip(fmt.Sprintf(
			"Acceptance tests skipped unless env '%s' set",
			TestEnvVar))
		return
	}

	isUnitTest := (os.Getenv(TestEnvVar) == UnitTestOverride)

	logWriter, err := logging.LogOutput()
	if err != nil {
		t.Error(fmt.Errorf("error setting up logging: %s", err))
	}
	log.SetOutput(logWriter)

	// We require verbose mode so that the user knows what is going on.
	if !testTesting && !testing.Verbose() && !isUnitTest {
		t.Fatal("Acceptance tests must be run with the -v flag on tests")
		return
	}

	// Run the PreCheck if we have it
	if c.PreCheck != nil {
		c.PreCheck()
	}

	// Build our context options that we can
	ctxProviders := c.ProviderFactories
	if ctxProviders == nil {
		ctxProviders = make(map[string]terraform.ResourceProviderFactory)
		for k, p := range c.Providers {
			ctxProviders[k] = terraform.ResourceProviderFactoryFixed(p)
		}
	}
	opts := terraform.ContextOpts{Providers: ctxProviders}

	// A single state variable to track the lifecycle, starting with no state
	var state *terraform.State

	// Go through each step and run it
	var idRefreshCheck *terraform.ResourceState
	idRefresh := c.IDRefreshName != ""
	errored := false
	for i, step := range c.Steps {
		var err error
		log.Printf("[WARN] Test: Executing step %d", i)
		state, err = testStep(opts, state, step)
		if err != nil {
			errored = true
			t.Error(fmt.Sprintf(
				"Step %d error: %s", i, err))
			break
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
						"ID-Only refresh test failure: %s", err))
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
		destroyStep := TestStep{
			Config:  c.Steps[len(c.Steps)-1].Config,
			Check:   c.CheckDestroy,
			Destroy: true,
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

// UnitTest is a helper to force the acceptance testing harness to run in the
// normal unit test suite. This should only be used for resource that don't
// have any external dependencies.
func UnitTest(t TestT, c TestCase) {
	oldEnv := os.Getenv(TestEnvVar)
	if err := os.Setenv(TestEnvVar, UnitTestOverride); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Setenv(TestEnvVar, oldEnv); err != nil {
			t.Fatal(err)
		}
	}()
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
	ctx := terraform.NewContext(&opts)
	if ws, es := ctx.Validate(); len(ws) > 0 || len(es) > 0 {
		if len(es) > 0 {
			estrs := make([]string, len(es))
			for i, e := range es {
				estrs[i] = e.Error()
			}
			return fmt.Errorf(
				"Configuration is invalid.\n\nWarnings: %#v\n\nErrors: %#v",
				ws, estrs)
		}

		log.Printf("[WARN] Config warnings: %#v", ws)
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

func testStep(
	opts terraform.ContextOpts,
	state *terraform.State,
	step TestStep) (*terraform.State, error) {
	mod, err := testModule(opts, step)
	if err != nil {
		return state, err
	}

	// Build the context
	opts.Module = mod
	opts.State = state
	opts.Destroy = step.Destroy
	ctx := terraform.NewContext(&opts)
	if ws, es := ctx.Validate(); len(ws) > 0 || len(es) > 0 {
		if len(es) > 0 {
			estrs := make([]string, len(es))
			for i, e := range es {
				estrs[i] = e.Error()
			}
			return state, fmt.Errorf(
				"Configuration is invalid.\n\nWarnings: %#v\n\nErrors: %#v",
				ws, estrs)
		}
		log.Printf("[WARN] Config warnings: %#v", ws)
	}

	// Refresh!
	state, err = ctx.Refresh()
	if err != nil {
		return state, fmt.Errorf(
			"Error refreshing: %s", err)
	}

	// Plan!
	if p, err := ctx.Plan(); err != nil {
		return state, fmt.Errorf(
			"Error planning: %s", err)
	} else {
		log.Printf("[WARN] Test: Step plan: %s", p)
	}

	// We need to keep a copy of the state prior to destroying
	// such that destroy steps can verify their behaviour in the check
	// function
	stateBeforeApplication := state.DeepCopy()

	// Apply!
	state, err = ctx.Apply()
	if err != nil {
		return state, fmt.Errorf("Error applying: %s", err)
	}

	// Check! Excitement!
	if step.Check != nil {
		if step.Destroy {
			if err := step.Check(stateBeforeApplication); err != nil {
				return state, fmt.Errorf("Check failed: %s", err)
			}
		} else {
			if err := step.Check(state); err != nil {
				return state, fmt.Errorf("Check failed: %s", err)
			}
		}
	}

	// Now, verify that Plan is now empty and we don't have a perpetual diff issue
	// We do this with TWO plans. One without a refresh.
	var p *terraform.Plan
	if p, err = ctx.Plan(); err != nil {
		return state, fmt.Errorf("Error on follow-up plan: %s", err)
	}
	if p.Diff != nil && !p.Diff.Empty() {
		if step.ExpectNonEmptyPlan {
			log.Printf("[INFO] Got non-empty plan, as expected:\n\n%s", p)
		} else {
			return state, fmt.Errorf(
				"After applying this step, the plan was not empty:\n\n%s", p)
		}
	}

	// And another after a Refresh.
	state, err = ctx.Refresh()
	if err != nil {
		return state, fmt.Errorf(
			"Error on follow-up refresh: %s", err)
	}
	if p, err = ctx.Plan(); err != nil {
		return state, fmt.Errorf("Error on second follow-up plan: %s", err)
	}
	if p.Diff != nil && !p.Diff.Empty() {
		if step.ExpectNonEmptyPlan {
			log.Printf("[INFO] Got non-empty plan, as expected:\n\n%s", p)
		} else {
			return state, fmt.Errorf(
				"After applying this step and refreshing, "+
					"the plan was not empty:\n\n%s", p)
		}
	}

	// Made it here, but expected a non-empty plan, fail!
	if step.ExpectNonEmptyPlan && (p.Diff == nil || p.Diff.Empty()) {
		return state, fmt.Errorf("Expected a non-empty plan, but got an empty plan!")
	}

	// Made it here? Good job test step!
	return state, nil
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
	modStorage := &getter.FolderStorage{
		StorageDir: filepath.Join(cfgPath, ".tfmodules"),
	}
	err = mod.Load(modStorage, module.GetModeGet)
	if err != nil {
		return nil, fmt.Errorf("Error downloading modules: %s", err)
	}

	return mod, nil
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

func TestCheckResourceAttr(name, key, value string) TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s", name)
		}

		if is.Attributes[key] != value {
			return fmt.Errorf(
				"%s: Attribute '%s' expected %#v, got %#v",
				name,
				key,
				value,
				is.Attributes[key])
		}

		return nil
	}
}

func TestMatchResourceAttr(name, key string, r *regexp.Regexp) TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		is := rs.Primary
		if is == nil {
			return fmt.Errorf("No primary instance: %s", name)
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

// TestCheckOutput checks an output in the Terraform configuration
func TestCheckOutput(name, value string) TestCheckFunc {
	return func(s *terraform.State) error {
		ms := s.RootModule()
		rs, ok := ms.Outputs[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs != value {
			return fmt.Errorf(
				"Output '%s': expected %#v, got %#v",
				name,
				value,
				rs)
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
}

// This is set to true by unit tests to alter some behavior
var testTesting = false
