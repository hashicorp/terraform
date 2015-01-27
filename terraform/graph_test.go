package terraform

import (
	"strings"
	"testing"
)

func TestGraphAdd(t *testing.T) {
	// Test Add since we override it and want to make sure we don't break it.
	var g Graph
	g.Add(42)
	g.Add(84)

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testGraphAddStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

const testGraphAddStr = `
42
84
`
