package atlas

import "testing"

func TestMaskString_emptyString(t *testing.T) {
	result := maskString("")
	expected := "*** (masked)"

	if result != expected {
		t.Errorf("expected %s to be %s", result, expected)
	}
}

func TestMaskString_threeString(t *testing.T) {
	result := maskString("123")
	expected := "*** (masked)"

	if result != expected {
		t.Errorf("expected %s to be %s", result, expected)
	}
}

func TestMaskString_longerString(t *testing.T) {
	result := maskString("ABCD1234")
	expected := "ABC*** (masked)"

	if result != expected {
		t.Errorf("expected %s to be %s", result, expected)
	}
}
