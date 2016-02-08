package atlas

import (
	"strings"
	"testing"
)

func TestParseSlug_emptyString(t *testing.T) {
	_, _, err := ParseSlug("")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "missing slug"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestParseSlug_noSlashes(t *testing.T) {
	_, _, err := ParseSlug("bacon")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "malformed slug"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestParseSlug_multipleSlashes(t *testing.T) {
	_, _, err := ParseSlug("bacon/is/delicious/but/this/is/not/valid")
	if err == nil {
		t.Fatal("expected error, but nothing was returned")
	}

	expected := "malformed slug"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected %q to contain %q", err.Error(), expected)
	}
}

func TestParseSlug_goodString(t *testing.T) {
	user, name, err := ParseSlug("hashicorp/project")
	if err != nil {
		t.Fatal(err)
	}

	if user != "hashicorp" {
		t.Fatalf("expected %q to be %q", user, "hashicorp")
	}

	if name != "project" {
		t.Fatalf("expected %q to be %q", name, "project")
	}
}
