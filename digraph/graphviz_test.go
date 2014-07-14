package digraph

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteDot(t *testing.T) {
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
	if err := WriteDot(buf, nlist); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(string(buf.Bytes()))
	expected := strings.TrimSpace(writeDotStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

const writeDotStr = `
digraph {
	"a";
	"a" -> "b" [label="foo"];
	"a" -> "c" [label="Edge"];
	"b";
	"b" -> "d" [label="Edge"];
	"b" -> "e" [label="Edge"];
	"c";
	"d";
	"e";
}
`
