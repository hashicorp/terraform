package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

// This is the directory where our test fixtures are.
const fixtureDir = "./test-fixtures"

func TestLoadConfig(t *testing.T) {
	c, err := LoadConfig(filepath.Join(fixtureDir, "config"))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	expected := &Config{
		Providers: map[string]string{
			"aws": "foo",
			"do":  "bar",
		},
	}

	if !reflect.DeepEqual(c, expected) {
		t.Fatalf("bad: %#v", c)
	}
}
