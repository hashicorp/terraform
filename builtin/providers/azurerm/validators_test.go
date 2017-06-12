package azurerm

import "testing"

func TestValidateName_invalid(t *testing.T) {
	type testCases struct {
		Value    string
		ErrCount int
	}

	invalidCases := []testCases{
		{
			Value:    "",
			ErrCount: 1,
		},
		{
			Value:    " ",
			ErrCount: 1,
		},
		{
			Value:    "abc 123",
			ErrCount: 1,
		},
		{
			Value:    "abc123!",
			ErrCount: 1,
		},
		{
			Value:    "{invalid}",
			ErrCount: 1,
		},
	}

	for _, tc := range invalidCases {
		_, errors := validateName(tc.Value, "json")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q to trigger a validation error.", tc.Value)
		}
	}
}

func TestValidateName_valid(t *testing.T) {
	type testCases struct {
		Value    string
		ErrCount int
	}

	validCases := []testCases{
		{
			Value:    "abc123",
			ErrCount: 0,
		},
		{
			Value:    "abc-123",
			ErrCount: 0,
		},
		{
			Value:    "abc_123",
			ErrCount: 0,
		},
	}

	for _, tc := range validCases {
		_, errors := validateName(tc.Value, "json")
		if len(errors) != tc.ErrCount {
			t.Fatalf("Expected %q not to trigger a validation error.", tc.Value)
		}
	}
}
