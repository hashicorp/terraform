package terraform

import (
	"testing"
)

func TestPrefixUIOutput_impl(t *testing.T) {
	var _ UIOutput = new(PrefixUIOutput)
}

func testPrefixUIOutput(t *testing.T) {
	output := new(MockUIOutput)
	prefix := &PrefixUIOutput{
		Prefix:   "foo",
		UIOutput: output,
	}

	prefix.Output("foo")
	if output.OutputMessage != "foofoo" {
		t.Fatalf("bad: %#v", output)
	}
}
