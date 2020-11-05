package logging

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestPanicRecorder(t *testing.T) {
	rec := panics.registerPlugin("test")

	output := []string{
		"panic: test",
		"  stack info",
	}

	for _, line := range output {
		rec(line)
	}

	expected := fmt.Sprintf(pluginPanicOutput, "test", strings.Join(output, "\n"))

	res := PluginPanics()
	if len(res) == 0 {
		t.Fatal("no output")
	}

	if res[0] != expected {
		t.Fatalf("expected: %q\ngot: %q", expected, res[0])
	}
}

func TestPanicLimit(t *testing.T) {
	rec := panics.registerPlugin("test")

	rec("panic: test")

	for i := 0; i < 200; i++ {
		rec(fmt.Sprintf("LINE: %d", i))
	}

	res := PluginPanics()
	// take the extra content into account
	max := strings.Count(pluginPanicOutput, "\n") + panics.maxLines
	for _, out := range res {
		found := strings.Count(out, "\n")
		if found > max {
			t.Fatalf("expected no more than %d lines, got: %d", max, found)
		}
	}
}

func TestLogPanicWrapper(t *testing.T) {
	var buf bytes.Buffer
	logger := hclog.NewInterceptLogger(&hclog.LoggerOptions{
		Name:        "test",
		Level:       hclog.Debug,
		Output:      &buf,
		DisableTime: true,
	})

	wrapped := (&logPanicWrapper{
		Logger: logger,
	}).Named("test")

	wrapped.Debug("panic: invalid foo of bar")
	wrapped.Debug("\tstack trace")

	expected := `[DEBUG] test.test: PANIC: invalid foo of bar
[DEBUG] test.test: 	stack trace
`

	got := buf.String()

	if expected != got {
		t.Fatalf("Expected:\n%q\nGot:\n%q", expected, got)
	}

}
