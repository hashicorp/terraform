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
	if !strings.Contains(out, "\n\t\"a\";\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\t\"b\";\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\t\"c\";\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\t\"d\";\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\t\"e\";\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\t\"a\" -> \"b\" [label=\"foo\"];\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\t\"a\" -> \"c\" [label=\"Edge\"];\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\t\"b\" -> \"d\" [label=\"Edge\"];\n") {
		t.Fatalf("bad: %v", out)
	}
	if !strings.Contains(out, "\n\t\"b\" -> \"e\" [label=\"Edge\"];\n") {
		t.Fatalf("bad: %v", out)
	}
}
