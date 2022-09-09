package internal

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/nsf/jsondiff"
)

var (
	// defaultFields is the set of fields that are ignored by default for any
	// files by the given names.
	defaultFields = map[string][]string{
		"apply.json": {
			"0",
			"*.@timestamp",
		},
		"plan.json": {
			"terraform_version",
		},
		"state.json": {
			"terraform_version",
		},
	}
)

// Test defines a single equivalence test within our framework.
//
// Each test has a Name that references the directory that contains our testing
// data. Within this directory there should be a `spec.json` file which is
// read in the TestSpecification object.
//
// The Directory variable references the parent directory of the test, so the
// full path for a given test case is paths.Join(test.Directory, test.Name).
type Test struct {
	Name          string
	Directory     string
	Specification TestSpecification
}

// TestSpecification defines the specification for a given test case.
//
// For each test we have a set of GoldenFiles that we care about when comparing,
// and we have a set of IgnoreFields for each golden file that we should ignore
// when diffing.
//
// The GoldenFiles and IgnoreFields should be specified as in addition to the
// defaults provided by the framework for each of these. The default golden
// files are apply.json, state.json, and plan.json. The default ignored fields
// for each of these are specified in the defaultFields variable at the top
// of this file.
type TestSpecification struct {
	GoldenFiles  []string            `json:"golden-files"`
	IgnoreFields map[string][]string `json:"ignore-fields"`
}

// TestOutput provides the output of executing a given test.
//
// The output is a map of filenames (from the golden file list in the
// specification) to unmarshalled JSON data for that file.
type TestOutput struct {
	Test  Test
	files map[string]interface{}
}

// ReadTests accepts a directory and returns the set of test cases specified
// within this directory.
func ReadTests(directory string) ([]Test, error) {
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	var tests []Test
	for _, file := range files {
		if file.IsDir() {
			data, err := ioutil.ReadFile(path.Join(directory, file.Name(), "spec.json"))
			if err != nil {
				return nil, err
			}

			var specification TestSpecification
			if err := json.Unmarshal(data, &specification); err != nil {
				return nil, err
			}

			tests = append(tests, Test{
				Name:          file.Name(),
				Specification: specification,
				Directory:     directory,
			})
		}
	}
	return tests, nil
}

// Run executes the given Test using the Terraform binary specified by the
// parameter.
//
// The first error is any error reported by the Golang binary itself, the second
// error will contain any stderr reported by the Terraform binary.
func (test Test) Run(binary string) (TestOutput, error, error) {
	tmp, err := os.MkdirTemp(test.Directory, test.Name)
	if err != nil {
		return TestOutput{}, err, nil
	}
	defer os.RemoveAll(tmp)

	testDirectory := path.Join(test.Directory, test.Name)
	if err = filepath.WalkDir(testDirectory, cp(testDirectory, tmp, []string{"spec.json"})); err != nil {
		return TestOutput{}, err, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return TestOutput{}, err, nil
	}

	// We've got all our files copied into the temporary directory, so now we
	// just have to execute terraform and get the golden files.
	if err := os.Chdir(tmp); err != nil {
		return TestOutput{}, err, nil
	}
	defer os.Chdir(wd)

	init := exec.Command(binary, "init")
	plan := exec.Command(binary, "plan", "-out=equivalence_test_plan")
	apply := exec.Command(binary, "apply", "-json", "equivalence_test_plan")
	showState := exec.Command(binary, "show", "-json")
	showPlan := exec.Command(binary, "show", "-json", "equivalence_test_plan")

	initCapture := Capture(init)
	if err := init.Run(); err != nil {
		return TestOutput{}, err, initCapture.ToError()
	}

	planCapture := Capture(plan)
	if err := plan.Run(); err != nil {
		return TestOutput{}, err, planCapture.ToError()
	}

	applyCapture := Capture(apply)
	showStateCapture := Capture(showState)
	showPlanCapture := Capture(showPlan)

	if err := apply.Run(); err != nil {
		return TestOutput{}, err, applyCapture.ToError()
	}
	if err := showState.Run(); err != nil {
		return TestOutput{}, err, showStateCapture.ToError()
	}
	if err := showPlan.Run(); err != nil {
		return TestOutput{}, err, showPlanCapture.ToError()
	}

	files := map[string]interface{}{}
	if files["apply.json"], err = applyCapture.ToJson(true); err != nil {
		return TestOutput{}, err, nil
	}
	if files["state.json"], err = showStateCapture.ToJson(false); err != nil {
		return TestOutput{}, err, nil
	}
	if files["plan.json"], err = showPlanCapture.ToJson(false); err != nil {
		return TestOutput{}, err, nil
	}

	for _, golden := range test.Specification.GoldenFiles {
		var data interface{}
		raw, err := os.ReadFile(golden)
		if err != nil {
			return TestOutput{}, err, nil
		}
		if err := json.Unmarshal(raw, &data); err != nil {

		}
		files[golden] = data
	}

	return TestOutput{
		Test:  test,
		files: files,
	}, nil, nil
}

func (output TestOutput) GetFiles() (map[string]interface{}, error) {
	ret := map[string]interface{}{}
	for file, contents := range output.files {
		var ignoreFields []string
		ignoreFields = append(ignoreFields, defaultFields[file]...)
		ignoreFields = append(ignoreFields, output.Test.Specification.IgnoreFields[file]...)

		var err error
		if ret[file], err = StripJson(ignoreFields, contents); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

// Diff will report the difference between this TestOutput and the output
// already stored in the golden directory specified by the parameter.
func (output TestOutput) Diff(goldens string) (map[string]string, error) {
	files, err := output.GetFiles()
	if err != nil {
		return nil, err
	}

	ret := map[string]string{}
	for file, contents := range files {
		target := path.Join(goldens, output.Test.Name, file)

		golden, err := os.ReadFile(target)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		}

		actual, err := json.MarshalIndent(contents, "", "  ")
		if err != nil {
			return nil, err
		}

		if golden == nil {
			// Then this means we don't have a golden file for this yet (as in
			// this is the first time we are using it). Let's just pretend it
			// was empty.
			ret[file] = "  (new file)  "
			continue
		}

		opts := jsondiff.DefaultJSONOptions()
		opts.SkipMatches = true
		opts.Indent = "  "

		diff, pretty := jsondiff.Compare(golden, actual, &opts)
		switch diff {
		case jsondiff.BothArgsAreInvalidJson, jsondiff.SecondArgIsInvalidJson, jsondiff.FirstArgIsInvalidJson:
			return nil, errors.New(pretty)
		case jsondiff.FullMatch:
			ret[file] = "  (no change)  "
		default:
			ret[file] = pretty
		}
	}
	return ret, nil
}

func (output TestOutput) Update(goldens string) error {
	tmp, err := os.MkdirTemp(goldens, output.Test.Name)
	if err != nil {
		return err
	}

	// We won't RemoveAll with tmp automatically, as there will be a point where
	// the original file has been deleted and tmp is all we have in which case
	// we don't want to delete tmp if anything goes wrong moving tmp into the
	// original location. tmp can be used by the user to recover manually.

	files, err := output.GetFiles()
	if err != nil {
		return err
	}

	for file, contents := range files {
		data, err := json.MarshalIndent(contents, "", "  ")
		if err != nil {
			os.RemoveAll(tmp)
			return err
		}

		target := path.Join(tmp, file)
		if _, err := os.Stat(filepath.Dir(target)); os.IsNotExist(err) {
			// This means the parent directory for the target file doesn't exist
			// so let's make it.
			if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
				os.RemoveAll(tmp)
				return err
			}
		}

		if err := os.WriteFile(target, data, os.ModePerm); err != nil {
			os.RemoveAll(tmp)
			return err
		}
	}

	// Now we've copied all the new golden files into our temporary directory,
	// we just need to move everything over to the original.
	if err := os.RemoveAll(path.Join(goldens, output.Test.Name)); err != nil {
		os.RemoveAll(tmp)
		return err
	}

	// From now, any failures are bad. We have removed the old directory so we
	// have lost the previous state of the golden files. If anything goes wrong
	// at this point we won't delete the tmp directory so that the user can
	// recover the failed test case manually by moving the tmp directory over
	// themselves.

	if err := os.Mkdir(path.Join(goldens, output.Test.Name), os.ModePerm); err != nil {
		return err
	}

	if err = filepath.WalkDir(tmp, cp(tmp, path.Join(goldens, output.Test.Name), nil)); err != nil {
		return err
	}

	if err := os.RemoveAll(tmp); err != nil {
		return err
	}

	return nil
}
