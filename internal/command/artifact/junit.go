package artifact

import (
	"bytes"
	"encoding/xml"
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
// TestJUnitXMLFile implements the Artifact interface, which allows creation of local
// files that contains a description of a completed test suite. It is intended only
// for use in conjunction with a View that provides the streaming output of ongoing
// testing events.

type TestJUnitXMLFile struct {
	filename string

	// A config loader is required to access sources, which are used with diagnostics to create XML content
	configLoader *configload.Loader
}

type Artifact interface {
	Save(*moduletest.Suite) tfdiags.Diagnostics
}

var _ Artifact = (*TestJUnitXMLFile)(nil)

// NewTestJUnitXML returns a [Test] implementation that will, when asked to
// report "conclusion", write a JUnit XML report to the given filename.
//
// If the file already exists then this view will silently overwrite it at the
// point of being asked to write a conclusion. Otherwise it will create the
// file at that time. If creating or overwriting the file fails, a subsequent
// call to method Err will return information about the problem.
func NewTestJUnitXMLFile(filename string, configLoader *configload.Loader) *TestJUnitXMLFile {
	return &TestJUnitXMLFile{
		filename:     filename,
		configLoader: configLoader,
	}
}

func (v *TestJUnitXMLFile) Save(suite *moduletest.Suite) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// Prepare XML content
	sources := v.configLoader.Parser().Sources()
	xmlSrc, err := junitXMLTestReport(suite, sources)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "error generating JUnit XML test output",
			Detail:   err.Error(),
		})
		return diags
	}

	// Save XML to the specified path
	err = os.WriteFile(v.filename, xmlSrc, 0660)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "error generating JUnit XML test output",
			Detail:   err.Error(),
		})
		return diags
	}

	return diags
}

func junitXMLTestReport(suite *moduletest.Suite, sources map[string][]byte) ([]byte, error) {
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

		for _, run := range file.Runs {
			// Each run is a "test case".

			type WithMessage struct {
				Message string `xml:"message,attr,omitempty"`
				Body    string `xml:",cdata"`
			}
			type TestCase struct {
				Name      string       `xml:"name,attr"`
				Classname string       `xml:"classname,attr"`
				Skipped   *WithMessage `xml:"skipped,omitempty"`
				Failure   *WithMessage `xml:"failure,omitempty"`
				Error     *WithMessage `xml:"error,omitempty"`
				Stderr    *WithMessage `xml:"system-err,omitempty"`

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

			testCase := TestCase{
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
				testCase.Skipped = &WithMessage{
					// FIXME: Is there something useful we could say here about
					// why the test was skipped?
				}
			case moduletest.Fail:
				testCase.Failure = &WithMessage{
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
				testCase.Error = &WithMessage{
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
				testCase.Stderr = &WithMessage{
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
