// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package junit_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/command/junit"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/moduletest"
)

// This test cannot access sources when contructing output for XML files. Due to this, the majority of testing
// for TestJUnitXMLFile is in internal/command/test_test.go
// In the junit package we can write some limited tests about XML output as long as there are no errors and/or
// failing tests in the test.
func Test_TestJUnitXMLFile_Save(t *testing.T) {

	cases := map[string]struct {
		filename      string
		runner        *local.TestSuiteRunner
		suite         moduletest.Suite
		expectedOuput []byte
		expectError   bool
	}{
		"<skipped> element can explain when skip is due to the runner being stopped by an interrupt": {
			filename: "output.xml",
			runner: &local.TestSuiteRunner{
				Stopped: true,
			},
			suite: moduletest.Suite{
				Status: moduletest.Skip,
				Files: map[string]*moduletest.File{
					"file1.tftest.hcl": {
						Name:   "file1.tftest.hcl",
						Status: moduletest.Fail,
						Runs: []*moduletest.Run{
							{
								Name:   "my_test",
								Status: moduletest.Skip,
							},
						},
					},
				},
			},
			expectedOuput: []byte(`<?xml version="1.0" encoding="UTF-8"?><testsuites>
  <testsuite name="file1.tftest.hcl" tests="1" skipped="1" failures="0" errors="0">
    <testcase name="my_test" classname="file1.tftest.hcl">
      <skipped message="Testcase skipped due to an interrupt"><![CDATA[Terraform received an interrupt and stopped gracefully. This caused all remaining testcases to be skipped]]></skipped>
    </testcase>
  </testsuite>
</testsuites>`),
		},
		"<skipped> element can explain when skip is due to the previously errored runs/testcases in the file": {
			filename: "output.xml",
			runner:   &local.TestSuiteRunner{},
			suite: moduletest.Suite{
				Status: moduletest.Error,
				Files: map[string]*moduletest.File{
					"file1.tftest.hcl": {
						Name:   "file1.tftest.hcl",
						Status: moduletest.Error,
						Runs: []*moduletest.Run{
							{
								Name:   "my_test_1",
								Status: moduletest.Error,
							},
							{
								Name:   "my_test_2",
								Status: moduletest.Skip,
							},
						},
					},
				},
			},
			expectedOuput: []byte(`<?xml version="1.0" encoding="UTF-8"?><testsuites>
  <testsuite name="file1.tftest.hcl" tests="2" skipped="1" failures="0" errors="1">
    <testcase name="my_test_1" classname="file1.tftest.hcl">
      <error message="Encountered an error"></error>
    </testcase>
    <testcase name="my_test_2" classname="file1.tftest.hcl">
      <skipped message="Testcase skipped due to a previous testcase error"><![CDATA[Previous testcase "my_test_1" ended in error, which caused the remaining tests in the file to be skipped]]></skipped>
    </testcase>
  </testsuite>
</testsuites>`),
		},
		"<skipped> element is present without additional details when contextual data is not available": {
			filename: "output.xml",
			runner:   &local.TestSuiteRunner{
				// No data about being stopped
			},
			suite: moduletest.Suite{
				Status: moduletest.Pending,
				Files: map[string]*moduletest.File{
					"file1.tftest.hcl": {
						Name:   "file1.tftest.hcl",
						Status: moduletest.Pending,
						Runs: []*moduletest.Run{
							{
								Name:   "my_test",
								Status: moduletest.Skip, // Only run present is skipped, no previous errors
							},
						},
					},
				},
			},
			expectedOuput: []byte(`<?xml version="1.0" encoding="UTF-8"?><testsuites>
  <testsuite name="file1.tftest.hcl" tests="1" skipped="1" failures="0" errors="0">
    <testcase name="my_test" classname="file1.tftest.hcl">
      <skipped></skipped>
    </testcase>
  </testsuite>
</testsuites>`),
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			// Setup test
			td := t.TempDir()
			path := fmt.Sprintf("%s/%s", td, tc.filename)

			loader, cleanup := configload.NewLoaderForTests(t)
			defer cleanup()

			j := junit.NewTestJUnitXMLFile(path, loader, tc.runner)

			// Process data & save file
			j.Save(&tc.suite)

			// Assertions
			actualOut, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("error opening XML file: %s", err)
			}

			if !bytes.Equal(actualOut, tc.expectedOuput) {
				t.Fatalf("expected output:\n%s\ngot:\n%s", tc.expectedOuput, actualOut)
			}
		})
	}

}
