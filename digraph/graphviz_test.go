package digraph

import (
  "bytes"
  "reflect"
  "sort"
  "strings"
  "testing"
)

var testTable = []struct {
  BasicData           string // Basic graph data
  ExpectedDigraphData string // Digraph data which should be generated from BasicData
}{
  {
    `a -> b ; foo
     a -> c
     b -> d
     b -> e
    `,
    `digraph {
     "a";
     "a" -> "b" [label="foo"];
     "a" -> "c" [label="Edge"];
     "b";
     "b" -> "d" [label="Edge"];
     "b" -> "e" [label="Edge"];
     "c";
     "d";
     "e";
    }`,
  },
  {
    `a -> c ; foo
     a -> d
     b -> c
     b -> e
     a -> f
    `,
    `digraph {
     "a";
     "a" -> "c" [label="foo"];
     "a" -> "f" [label="Edge"];
     "a" -> "d" [label="Edge"];
     "b";
     "b" -> "c" [label="Edge"];
     "b" -> "e" [label="Edge"];
     "c";
     "d";
     "e";
     "f";
    }`,
  },
}

// Nieve normalizer. Takes a string, splits it and sorts it.
func normalize(input string) []string {
  out := strings.Split(input, "\n")

  // trim each line
  for i, str := range out {
    out[i] = strings.TrimSpace(str)
  }

  sort.Strings(out)
  return out
}

func TestWriteDot(t *testing.T) {
  // Build []Node from BasicNode map
  var buildNodes = func(nodes map[string]*BasicNode) []Node {
    var nlist []Node
    for _, n := range nodes {
      nlist = append(nlist, n)
    }
    return nlist
  }

  // Get a normalized string representation of the file
  var writeFile = func(nlist []Node) string {
    buf := bytes.NewBuffer(nil)
    if err := WriteDot(buf, nlist); err != nil {
      t.Fatalf("err: %s", err)
    }
    return strings.TrimSpace(string(buf.Bytes()))
  }

  // For each entry in the test table construct an
  // actual and expected values and compare.
  for _, data := range testTable {
    nodes := buildNodes(ParseBasic(data.BasicData))
    actual := normalize(writeFile(nodes))
    expected := normalize(strings.TrimSpace(data.ExpectedDigraphData))

    // Deep equal the array values
    if !reflect.DeepEqual(actual, expected) {
      t.Logf("Expected:\n%s", expected)
      t.Fatalf("Bad:\n%s", actual)
    }
  }
}
