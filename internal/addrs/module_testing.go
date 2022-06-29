package addrs

import (
	"fmt"
)

// ModuleTestScenario represents the address of a module testing scenario.
type ModuleTestScenario struct {
	// Path is the normalized path of the test scenario, using the same
	// syntax as for moduletest.Scenario.Path.
	Path string
}

func (s ModuleTestScenario) TestStep(name string) ModuleTestStep {
	return ModuleTestStep{
		Scenario: s,
		Name:     name,
	}
}

// InitPseudoStep returns the address of the "pseudo-step" that represents
// the initialization process for this scenario.
//
// Init is not a real step so it cannot actually have any test cases or
// checks associated with it, but it can potentially encounter errors which
// we'll report as if they were a part of this step.
func (s ModuleTestScenario) InitPseudoStep() ModuleTestStep {
	return ModuleTestStep{
		Scenario: s,
		Name:     "(init)", // This must match moduletest.InitPseudoStepName
	}
}

func (s ModuleTestScenario) TestCase(stepName string, object ConfigCheckable) ModuleTestCase {
	return ModuleTestCase{
		Step:   s.TestStep(stepName),
		Object: object,
	}
}

func (s ModuleTestScenario) String() string {
	return s.Path
}

func (s ModuleTestScenario) UniqueKey() UniqueKey {
	// A module test scenario is comparable and so can be its own UniqueKey
	return s
}

func (s ModuleTestScenario) uniqueKeySigil() {}

// ModuleTestStep represents the address of a step or pseudo-step within a
// module testing scenario.
type ModuleTestStep struct {
	Scenario ModuleTestScenario

	// Name is either the name of the step as defined in moduletest.Step.Name,
	// or it's the reserved pseudo-step name "(init)" to represent the
	// result of the initialization process even though it isn't truly a step.
	Name string
}

func (s ModuleTestStep) TestCase(object ConfigCheckable) ModuleTestCase {
	return ModuleTestCase{
		Step:   s,
		Object: object,
	}
}

func (s ModuleTestStep) String() string {
	if s.Name == "" {
		// The empty string is the reserved name for the unnamed "default"
		// testing step we generate for any scenario that doesn't have an
		// explicit configuration to specify its named steps.
		return s.Scenario.String()
	}
	return fmt.Sprintf("%s.%s", s.Scenario.String(), s.Name)
}

func (s ModuleTestStep) UniqueKey() UniqueKey {
	// A module test scenario is comparable and so can be its own UniqueKey
	return s
}

func (s ModuleTestStep) uniqueKeySigil() {}

// ModuleTestCase represents the address of an individual test case.
//
// We define "test case" as an individual checkable configuration object
// within a particular step of a particular test scenario.
type ModuleTestCase struct {
	Step ModuleTestStep

	// Object is the configuration object that the test case relates to.
	//
	// All of the results for dynamic checkable objects (Checkable) for a
	// particular ConfigCheckable aggregate under the same test case, so
	// that our set of test cases will remain relatively consistent between
	// runs even though sometimes we won't get far enough to actually
	// expand the dynamic objects for all test cases.
	Object ConfigCheckable
}

// String returns a string representation of the fully-qualified test case
// address.
//
// We don't typically use this fully-qualified form because we'll normally
// report test case results in a context that implies which step they
// belong to, but a fully-qualfied address might be useful for debug logging
// or similar.
func (c ModuleTestCase) String() string {
	return fmt.Sprintf("%s#%s", c.Step, c.Object)
}

// moduleTestCaseKey is the UniqueKey implementation for ModuleTestCase.
type moduleTestCaseKey struct {
	StepKey   UniqueKey
	ObjectKey UniqueKey
}

func (s ModuleTestCase) UniqueKey() UniqueKey {
	return moduleTestCaseKey{
		StepKey:   s.Step.UniqueKey(),
		ObjectKey: s.Object.UniqueKey(),
	}
}

func (k moduleTestCaseKey) uniqueKeySigil() {}
