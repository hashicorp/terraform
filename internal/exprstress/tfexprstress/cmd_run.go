package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform/internal/exprstress"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/zclconf/go-cty-debug/ctydebug"
)

func runCommand(ctx context.Context, streams *terminal.Streams, args []string) int {
	if len(args) != 0 {
		streams.Eprintln("Error: The run command doesn't expect any arguments.")
		return 1
	}

	// We'll run one testing loop per CPU so we can work through
	// test cases as quickly as possible.
	// TODO: Maybe make this configurable?
	concurrency := runtime.NumCPU()
	var wg sync.WaitGroup

	streams.Printf("%d concurrent expression stress test workers will run until interrupted\n", concurrency)

	worker := func(workerID int) {
		seed := time.Now().Unix() + int64(workerID*100)
		rand := rand.New(rand.NewSource(seed))
		for {
			select {
			case <-ctx.Done():
				// our caller will arrange for ctx to be cancelled when
				// we should stop running.
				wg.Done()
				return
			default:
				// otherwise we'll try another test case below
			}

			expr := exprstress.GenerateExpression(rand)
			errs := exprstress.TestExpression(expr)
			if len(errs) == 0 {
				continue
			}

			// If we get here then this test case has failed, and so we'll
			// print an error message with a hint at how to convert this
			// test case into a unit test.
			//
			// Since we may have multiple workers concurrently writing to
			// the stderr stream, we'll buffer up what we want to write and
			// then write it out as a single buffer to avoid messages
			// becoming interleaved with one another.
			var buf bytes.Buffer
			src := exprstress.ExpressionSourceBytes(expr)
			expected := expr.ExpectedResult()

			fmt.Fprintf(
				&buf,
				"\n- Test failure! The following test case encountered errors:\n  %s\n",
				formatTestCaseValue(src, expected, 2),
			)
			for _, err := range errs {
				fmt.Fprintf(&buf, "    - %s\n", indentString(err.Error(), 6))
			}

			streams.Stderr.File.Write(buf.Bytes())
		}
	}

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker(i)
	}

	wg.Wait()
	streams.Stderr.File.WriteString("\n")

	return 0
}

func formatTestCaseValue(exprSrc []byte, expected exprstress.Expected, indent int) string {
	var buf strings.Builder
	indentSpaces := strings.Repeat(" ", indent)
	fmt.Fprint(&buf, "exprstress.TestCase{\n")
	fmt.Fprintf(&buf, "%s    ExprSrc: %#q,\n", indentSpaces, exprSrc)
	fmt.Fprintf(&buf, "%s    Expected: %s,\n", indentSpaces, formatExpectedResult(expected, indent+4))
	fmt.Fprintf(&buf, "%s}", indentSpaces)
	return buf.String()
}

func formatExpectedResult(expected exprstress.Expected, indent int) string {
	var buf strings.Builder
	indentSpaces := strings.Repeat(" ", indent)
	fmt.Fprint(&buf, "exprstress.Expected{\n")
	typeStr := indentString(ctydebug.TypeString(expected.Type), indent+4)
	if strings.Contains(typeStr, "\n") {
		fmt.Fprintf(&buf, "%s    Type: %s,\n", indentSpaces, indentString(ctydebug.TypeString(expected.Type), indent+4))
	} else {
		fmt.Fprintf(&buf, "%s    Type:          %s,\n", indentSpaces, indentString(ctydebug.TypeString(expected.Type), indent+4))
	}
	fmt.Fprintf(&buf, "%s    Mode:          %#v,\n", indentSpaces, expected.Mode)
	fmt.Fprintf(&buf, "%s    Sensitive:     %#v,\n", indentSpaces, expected.Sensitive)
	fmt.Fprintf(&buf, "%s    SpecialNumber: %#v,\n", indentSpaces, expected.SpecialNumber)
	fmt.Fprintf(&buf, "%s}", indentSpaces)
	return buf.String()
}

func indentString(s string, indent int) string {
	var buf strings.Builder
	sc := bufio.NewScanner(strings.NewReader(s))
	indentSpaces := strings.Repeat(" ", indent)
	for sc.Scan() {
		buf.WriteString(indentSpaces)
		buf.WriteString(sc.Text())
		buf.WriteByte('\n')
	}
	return strings.TrimSpace(buf.String())
}
