package moduletest

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// LoadScenarios searches the given base directory for test scenarios, and
// returns the top-level definitions of them.
//
// LoadScenarios does not actually load the main Terraform configuration
// associated with each scenario, so the success of LoadScenarios does not
// imply that any of the test configurations are valid, but it does imply
// that the test scenario definitions are valid.
func LoadScenarios(baseDir string) (map[addrs.ModuleTestScenario]*Scenario, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	testsDir := filepath.Join(baseDir, "tests")

	items, err := ioutil.ReadDir(testsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			diags = diags.Append(tfdiags.Sourceless(
				// This is a warning rather than an error because we will sometimes
				// load scenarios in situations that are not testing-focused and
				// we should ideally still be able to continue with other work
				// even if we couldn't determine a set of tests, but we also don't
				// want to leave the user completely baffled as to why we didn't
				// detect any tests in a directory that appears to exist.
				tfdiags.Warning,
				"Failed to discover testing scenarios",
				fmt.Sprintf("Failed to read %s to discover testing scenarios for this module: %s.", testsDir, err),
			))
		}
		return nil, diags
	}

	ret := make(map[addrs.ModuleTestScenario]*Scenario, len(items))
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		name := item.Name()
		scenarioPath := filepath.Join(testsDir, name)
		tfFiles, err := filepath.Glob(filepath.Join(scenarioPath, "*.tf"))
		if err != nil {
			// We'll just ignore it and treat it like a dir with no .tf files
			tfFiles = nil
		}
		tfJSONFiles, err := filepath.Glob(filepath.Join(scenarioPath, "*.tf.json"))
		if err != nil {
			// We'll just ignore it and treat it like a dir with no .tf.json files
			tfJSONFiles = nil
		}
		if (len(tfFiles) + len(tfJSONFiles)) == 0 {
			// Not a test suite, then.
			continue
		}

		// Our canonical test path is a relative path from baseDir always
		// using forward slashes, just so our test result reports will not
		// vary in the test naming depending on which platform the user ran
		// the tests on.
		relPath, err := filepath.Rel(baseDir, scenarioPath)
		if err != nil {
			// Since we generated this path by concatenating subdirectories
			// onto the base directory, it should always be possible to
			// compute a relative path, and so we shouldn't get here.
			diags = diags.Append(fmt.Errorf("failed to determine relative path for %s: %s", scenarioPath, err))
			continue
		}
		relPath = filepath.ToSlash(relPath)

		scenario := &Scenario{
			Path:    relPath,
			BaseDir: filepath.ToSlash(filepath.Clean(baseDir)),

			// We don't yet have a scenario description language, and so we
			// just assume a basic default pair of steps for all scenarios.
			// If we do add such a language in future, this would be the
			// appropriate point to try to load a scenario configuration file
			// and use the steps defined in there instead of the default
			// step hard-coded below.
			Steps: []*Step{
				{
					Name:     DefaultStepName,
					PlanMode: plans.NormalMode,
				},
				{
					Name:     CleanupStepName,
					PlanMode: plans.DestroyMode,
				},
			},
		}
		for _, step := range scenario.Steps {
			step.Scenario = scenario
		}

		ret[scenario.Addr()] = scenario
	}

	return ret, diags
}
