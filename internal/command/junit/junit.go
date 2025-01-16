// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1
package junit

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/moduletest"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestJUnitXMLFile produces a JUnit XML file at the conclusion of a test
// run, summarizing the outcome of the test in a form that can then be
// interpreted by tools which render JUnit XML result reports.
//
// The de-facto convention for JUnit XML is for it to be emitted as a separate
// file as a complement to human-oriented output, rather than _instead of_
// human-oriented output. To meet that expectation the method [TestJUnitXMLFile.Save]
// should be called at the same time as the test's view reaches its "Conclusion" event.
// If that event isn't reached for any reason then no file should be created at
// all, which JUnit XML-consuming tools tend to expect as an outcome of a
// catastrophically-errored test suite.
//
// TestJUnitXMLFile implements the JUnit interface, which allows creation of a local
// file that contains a description of a completed test suite. It is intended only
// for use in conjunction with a View that provides the streaming output of ongoing
// testing events.

type TestJUnitXMLFile struct {
	filename string

	// A config loader is required to access sources, which are used with diagnostics to create XML content
	configLoader *configload.Loader

	// A pointer to the containing test suite runner is needed to monitor details like the command being stopped
	testSuiteRunner moduletest.TestSuiteRunner
}

type JUnit interface {
	Save(*moduletest.Suite) tfdiags.Diagnostics
}

var _ JUnit = (*TestJUnitXMLFile)(nil)

// NewTestJUnitXML returns a [Test] implementation that will, when asked to
// report "conclusion", write a JUnit XML report to the given filename.
//
// If the file already exists then this view will silently overwrite it at the
// point of being asked to write a conclusion. Otherwise it will create the
// file at that time. If creating or overwriting the file fails, a subsequent
// call to method Err will return information about the problem.
func NewTestJUnitXMLFile(filename string, configLoader *configload.Loader, testSuiteRunner moduletest.TestSuiteRunner) *TestJUnitXMLFile {
	return &TestJUnitXMLFile{
		filename:        filename,
		configLoader:    configLoader,
		testSuiteRunner: testSuiteRunner,
	}
}

// Save takes in a test suite, generates JUnit XML summarising the test results,
// and saves the content to the filename specified by user
func (v *TestJUnitXMLFile) Save(suite *moduletest.Suite) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Prepare XML content
	sources := v.configLoader.Parser().Sources()
	xmlSrc, err := junitXMLTestReport(suite, v.testSuiteRunner.IsStopped(), sources)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "error generating JUnit XML test output",
			Detail:   err.Error(),
		})
		return diags
	}

	// Save XML to the specified path
	saveDiags := v.save(xmlSrc)
	diags = append(diags, saveDiags...)

	return diags

}

func (v *TestJUnitXMLFile) save(xmlSrc []byte) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	err := os.WriteFile(v.filename, xmlSrc, 0660)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  fmt.Sprintf("error saving JUnit XML to file %q", v.filename),
			Detail:   err.Error(),
		})
		return diags
	}

	return nil
}

type withMessage struct {
	Message string `xml:"message,attr,omitempty"`
	Body    string `xml:",cdata"`
}

type testCase struct {
	Name      string       `xml:"name,attr"`
	Classname string       `xml:"classname,attr"`
	Skipped   *withMessage `xml:"skipped,omitempty"`
	Failure   *withMessage `xml:"failure,omitempty"`
	Error     *withMessage `xml:"error,omitempty"`
	Stderr    *withMessage `xml:"system-err,omitempty"`

	// RunTime is the time spent executing the run associated
	// with this test case, in seconds with the fractional component
	// representing partial seconds.
	//
	// We assume here that it's not practically possible for an
	// execution to take literally zero fractional seconds at
	// the accuracy we're using here (nanoseconds converted into
	// floating point seconds) and so use zero to represent
	// "not known", and thus omit that case. (In practice many
	// JUnit XML consumers treat the absense of this attribute
	// as zero anyway.)
	RunTime   float64 `xml:"time,attr,omitempty"`
	Timestamp string  `xml:"timestamp,attr,omitempty"`
}

func junitXMLTestReport(suite *moduletest.Suite, suiteRunnerStopped bool, sources map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	enc.EncodeToken(xml.ProcInst{
		Target: "xml",
		Inst:   []byte(`version="1.0" encoding="UTF-8"`),
	})
	enc.Indent("", "  ")

	// Some common element/attribute names we'll use repeatedly below.
	suitesName := xml.Name{Local: "testsuites"}
	suiteName := xml.Name{Local: "testsuite"}
	caseName := xml.Name{Local: "testcase"}
	nameName := xml.Name{Local: "name"}
	testsName := xml.Name{Local: "tests"}
	skippedName := xml.Name{Local: "skipped"}
	failuresName := xml.Name{Local: "failures"}
	errorsName := xml.Name{Local: "errors"}

	enc.EncodeToken(xml.StartElement{Name: suitesName})

	sortedFiles := suiteFilesAsSortedList(suite.Files) // to ensure consistent ordering in XML
	for _, file := range sortedFiles {
		// Each test file is modelled as a "test suite".

		// First we'll count the number of tests and number of failures/errors
		// for the suite-level summary.
		totalTests := len(file.Runs)
		totalFails := 0
		totalErrs := 0
		totalSkipped := 0
		for _, run := range file.Runs {
			switch run.Status {
			case moduletest.Skip:
				totalSkipped++
			case moduletest.Fail:
				totalFails++
			case moduletest.Error:
				totalErrs++
			}
		}
		enc.EncodeToken(xml.StartElement{
			Name: suiteName,
			Attr: []xml.Attr{
				{Name: nameName, Value: file.Name},
				{Name: testsName, Value: strconv.Itoa(totalTests)},
				{Name: skippedName, Value: strconv.Itoa(totalSkipped)},
				{Name: failuresName, Value: strconv.Itoa(totalFails)},
				{Name: errorsName, Value: strconv.Itoa(totalErrs)},
			},
		})

		for i, run := range file.Runs {
			// Each run is a "test case".

			testCase := testCase{
				Name: run.Name,

				// We treat the test scenario filename as the "class name",
				// implying that the run name is the "method name", just
				// because that seems to inspire more useful rendering in
				// some consumers of JUnit XML that were designed for
				// Java-shaped languages.
				Classname: file.Name,
			}
			if execMeta := run.ExecutionMeta; execMeta != nil {
				testCase.RunTime = execMeta.Duration.Seconds()
				testCase.Timestamp = execMeta.StartTimestamp()
			}
			switch run.Status {
			case moduletest.Skip:
				message, body := getSkipDetails(i, file, suiteRunnerStopped)
				testCase.Skipped = &withMessage{
					Message: message,
					Body:    body,
				}
			case moduletest.Fail:
				testCase.Failure = &withMessage{
					Message: "Test run failed",
					// FIXME: What's a useful thing to report in the body
					// here? A summary of the statuses from all of the
					// checkable objects in the configuration?
				}
			case moduletest.Error:
				var diagsStr strings.Builder
				for _, diag := range run.Diagnostics {
					diagsStr.WriteString(format.DiagnosticPlain(diag, sources, 80))
				}
				testCase.Error = &withMessage{
					Message: "Encountered an error",
					Body:    diagsStr.String(),
				}
			}
			if len(run.Diagnostics) != 0 && testCase.Error == nil {
				// If we have diagnostics but the outcome wasn't an error
				// then we're presumably holding diagnostics that didn't
				// cause the test to error, such as warnings. We'll place
				// those into the "system-err" element instead, so that
				// they'll be reported _somewhere_ at least.
				var diagsStr strings.Builder
				for _, diag := range run.Diagnostics {
					diagsStr.WriteString(format.DiagnosticPlain(diag, sources, 80))
				}
				testCase.Stderr = &withMessage{
					Body: diagsStr.String(),
				}
			}
			enc.EncodeElement(&testCase, xml.StartElement{
				Name: caseName,
			})
		}

		enc.EncodeToken(xml.EndElement{Name: suiteName})
	}
	enc.EncodeToken(xml.EndElement{Name: suitesName})
	enc.Close()
	return buf.Bytes(), nil
}

// getSkipDetails checks data about the test suite, file and runs to determine why a given run was skipped
// Test can be skipped due to:
// 1. terraform test recieving an interrupt from users; all unstarted tests will be skipped
// 2. A previous run in a file has failed, causing subsequent run blocks to be skipped
func getSkipDetails(runIndex int, file *moduletest.File, suiteStopped bool) (string, string) {
	if suiteStopped {
		// Test suite experienced an interrupt
		// This block only handles graceful Stop interrupts, as Cancel interrupts will prevent a JUnit file being produced at all
		message := "Testcase skipped due to an interrupt"
		body := "Terraform received an interrupt and stopped gracefully. This caused all remaining testcases to be skipped"

		return message, body
	}

	if file.Status == moduletest.Error {
		// Overall test file marked as errored in the context of a skipped test means tests have been skipped after
		// a previous test/run blocks has errored out
		for i := runIndex; i >= 0; i-- {
			if file.Runs[i].Status == moduletest.Error {
				// Skipped due to error in previous run within the file
				message := "Testcase skipped due to a previous testcase error"
				body := fmt.Sprintf("Previous testcase %q ended in error, which caused the remaining tests in the file to be skipped", file.Runs[i].Name)
				return message, body
			}
		}
	}

	// Unhandled case: This results in <skipped></skipped> with no attributes or body
	return "", ""
}

func suiteFilesAsSortedList(files map[string]*moduletest.File) []*moduletest.File {
	fileNames := make([]string, len(files))
	i := 0
	for k := range files {
		fileNames[i] = k
		i++
	}
	slices.Sort(fileNames)

	sortedFiles := make([]*moduletest.File, len(files))
	for i, name := range fileNames {
		sortedFiles[i] = files[name]
	}
	return sortedFiles
}
