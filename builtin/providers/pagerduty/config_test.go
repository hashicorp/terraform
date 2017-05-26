package pagerduty

import (
	"testing"
)

// Test config with an empty token
func TestConfigEmptyToken(t *testing.T) {
	config := Config{
		Token: "",
	}

	if _, err := config.Client(); err == nil {
		t.Fatalf("expected error, but got nil")
	}
}

// Test config with invalid token but with SkipCredsValidation
func TestConfigSkipCredsValidation(t *testing.T) {
	config := Config{
		Token:               "foo",
		SkipCredsValidation: true,
	}

	if _, err := config.Client(); err != nil {
		t.Fatalf("error: expected the client to not fail: %v", err)
	}
}
