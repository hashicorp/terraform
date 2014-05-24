package digraph

import (
	"bytes"
	"strings"
	"testing"
)

func Test_GenerateDot(t *testing.T) {
	nodes := ParseBasic(`a -> b ; foo
a -> c
b -> d
b -> e
`)
	var nlist []Node
	for _, n := range nodes {
		nlist = append(nlist, n)
	}

	buf := bytes.NewBuffer(nil)
	GenerateDot(nlist, buf)

	out := string(buf.Bytes())
	if !strings.HasPrefix(out, "digraph {\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.HasSuffix(out, "\n}\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\ta;\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\tb;\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\tc;\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\td;\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\te;\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\ta -> b [label=\"foo\"];\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\ta -> c [label=\"Edge\"];\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\tb -> d [label=\"Edge\"];\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\tb -> e [label=\"Edge\"];\n") {
		t.Fatalf("bad: %v", out)
	}
}
