package resource

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/terraform"
)

const TestEnvVar = "TF_ACC"

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

	logWriter, err := logging.LogOutput()
	if err != nil {
		t.Error(fmt.Errorf("error setting up logging: %s", err))
	}
	log.SetOutput(logWriter)

	// We require verbose mode so that the user knows what is going on.
	if !testTesting && !testing.Verbose() {
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
	for i, step := range c.Steps {
		var err error
		log.Printf("[WARN] Test: Executing step %d", i)
		state, err = testStep(opts, state, step)
		if err != nil {
			t.Error(fmt.Sprintf(
				"Step %d error: %s", i, err))
			break
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

func testStep(
	opts terraform.ContextOpts,
	state *terraform.State,
	step TestStep) (*terraform.State, error) {
	if step.PreConfig != nil {
		step.PreConfig()
	}

	cfgPath, err := ioutil.TempDir("", "tf-test")
	if err != nil {
		return state, fmt.Errorf(
			"Error creating temporary directory for config: %s", err)
	}
	defer os.RemoveAll(cfgPath)

	// Write the configuration
	cfgF, err := os.Create(filepath.Join(cfgPath, "main.tf"))
	if err != nil {
		return state, fmt.Errorf(
			"Error creating temporary file for config: %s", err)
	}

	_, err = io.Copy(cfgF, strings.NewReader(step.Config))
	cfgF.Close()
	if err != nil {
		return state, fmt.Errorf(
			"Error creating temporary file for config: %s", err)
	}

	// Parse the configuration
	mod, err := module.NewTreeModule("", cfgPath)
	if err != nil {
		return state, fmt.Errorf(
			"Error loading configuration: %s", err)
	}

	// Load the modules
	modStorage := &getter.FolderStorage{
		StorageDir: filepath.Join(cfgPath, ".tfmodules"),
	}
	err = mod.Load(modStorage, module.GetModeGet)
	if err != nil {
		return state, fmt.Errorf("Error downloading modules: %s", err)
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
	if p.Diff != nil && !p.Diff.Empty() && !step.ExpectNonEmptyPlan {
		return state, fmt.Errorf(
			"After applying this step, the plan was not empty:\n\n%s", p)
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
	if p.Diff != nil && !p.Diff.Empty() && !step.ExpectNonEmptyPlan {
		return state, fmt.Errorf(
			"After applying this step and refreshing, the plan was not empty:\n\n%s", p)
	}

	// Made it here, but expected a non-empty plan, fail!
	if step.ExpectNonEmptyPlan && (p.Diff == nil || p.Diff.Empty()) {
		return state, fmt.Errorf("Expected a non-empty plan, but got an empty plan!")
	}

	// Made it here? Good job test step!
	return state, nil
}

// ComposeTestCheckFunc lets you compose multiple TestCheckFuncs into
// a single TestCheckFunc.
//
// As a user testing their provider, this lets you decompose your checks
// into smaller pieces more easily.
func ComposeTestCheckFunc(fs ...TestCheckFunc) TestCheckFunc {
	return func(s *terraform.State) error {
		for _, f := range fs {
			if err := f(s); err != nil {
				return err
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
