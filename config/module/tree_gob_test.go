package module

import (
	"bytes"
	"encoding/gob"
	"strings"
	"testing"
)

func TestTreeEncodeDecodeGob(t *testing.T) {
	storage := testStorage(t, nil)
	tree := NewTree("", testConfig(t, "basic"))

	// This should get things
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Encode it.
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(tree); err != nil {
		t.Fatalf("err: %s", err)
	}

	dec := gob.NewDecoder(&buf)
	var actual Tree
	if err := dec.Decode(&actual); err != nil {
		t.Fatalf("err: %s", err)
	}

	actualStr := strings.TrimSpace(actual.String())
	expectedStr := strings.TrimSpace(tree.String())
	if actualStr != expectedStr {
		t.Fatalf("\n%s\n\nexpected:\n\n%s", actualStr, expectedStr)
	}
}
