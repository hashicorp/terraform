package structure

import (
	"testing"
)

func TestSuppressJsonDiff_same(t *testing.T) {
	original := `{ "enabled": true }`
	new := `{ "enabled": true }`
	expected := true

	actual := SuppressJsonDiff("test", original, new, nil)
	if actual != expected {
		t.Fatal("[ERROR] Identical JSON values shouldn't cause a diff")
	}
}

func TestSuppressJsonDiff_sameWithWhitespace(t *testing.T) {
	original := `{
	  "enabled": true
	}`
	new := `{ "enabled": true }`
	expected := true

	actual := SuppressJsonDiff("test", original, new, nil)
	if actual != expected {
		t.Fatal("[ERROR] Identical JSON values shouldn't cause a diff")
	}
}

func TestSuppressJsonDiff_differentValue(t *testing.T) {
	original := `{ "enabled": true }`
	new := `{ "enabled": false }`
	expected := false

	actual := SuppressJsonDiff("test", original, new, nil)
	if actual != expected {
		t.Fatal("[ERROR] Different JSON values should cause a diff")
	}
}

func TestSuppressJsonDiff_newValue(t *testing.T) {
	original := `{ "enabled": true }`
	new := `{ "enabled": false, "world": "round" }`
	expected := false

	actual := SuppressJsonDiff("test", original, new, nil)
	if actual != expected {
		t.Fatal("[ERROR] Different JSON values should cause a diff")
	}
}
