package junit

import (
	"bytes"
	"encoding/xml"
	"slices"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/moduletest"
)

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
	// JUnit XML consumers treat the absence of this attribute
	// as zero anyway.)
	RunTime float64 `xml:"time,attr,omitempty"`
}

func JUnitXMLTestReport(suite *moduletest.Suite) ([]byte, error) {
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
					// FIXME: Pass in the sources so that these diagnostics
					// can include source snippets when appropriate.
					diagsStr.WriteString(format.DiagnosticPlain(diag, nil, 80))
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
					// FIXME: Pass in the sources so that these diagnostics
					// can include source snippets when appropriate.
					diagsStr.WriteString(format.DiagnosticPlain(diag, nil, 80))
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
