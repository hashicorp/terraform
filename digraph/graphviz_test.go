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

	actualLines := strings.Split(actual, "\n")
	expectedLines := strings.Split(expected, "\n")

	if actualLines[0] != expectedLines[0] ||
		actualLines[len(actualLines)-1] != expectedLines[len(expectedLines)-1] ||
		len(actualLines) != len(expectedLines) {
		t.Fatalf("bad: %s", actual)
	}

	count := 0
	for _, el := range expectedLines[1 : len(expectedLines)-1] {
		for _, al := range actualLines[1 : len(actualLines)-1] {
			if el == al {
				count++
				break
			}
		}
	}

	if count != len(expectedLines)-2 {
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
