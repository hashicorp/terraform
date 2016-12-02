package config

import (
	"testing"
)

// TestRawConfig is used to create a RawConfig for testing.
func TestRawConfig(t *testing.T, c map[string]interface{}) *RawConfig {
	cfg, err := NewRawConfig(c)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return cfg
}
